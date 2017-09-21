package rabbit

import (
	"strings"
	"sync/atomic"

	"gopkg.in/tomb.v2"

	"github.com/leandro-lugaresi/message-cannon/runner"
	"github.com/pkg/errors"
	"github.com/speps/go-hashids"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

// Factory is the block responsible for create consumers and restart the rabbitMQ connections.
type Factory struct {
	config Config
	conns  map[string]*amqp.Connection
	log    *zap.Logger
	number int64
}

// NewFactory will open the initial connections and start the recover connections procedure.
func NewFactory(config Config, log *zap.Logger) (*Factory, error) {
	conns := make(map[string]*amqp.Connection, 0)
	for name, cfgConn := range config.Connections {
		conn, err := amqp.Dial(cfgConn.DSN)
		if err != nil {
			return nil, errors.Wrapf(err, "error opening the connection \"%s\"", name)
		}
		conns[name] = conn
	}
	f := &Factory{
		config,
		conns,
		log,
		1,
	}
	return f, nil
}

// CreateConsumers will iterate over config and create all the consumers
func (f *Factory) CreateConsumers() ([]*consumer, error) {
	var consumers []*consumer
	for name, cfg := range f.config.Consumers {
		consumer, err := f.newConsumer(name, cfg)
		if err != nil {
			return consumers, err
		}
		consumers = append(consumers, consumer)
	}
	return consumers, nil
}

// CreateConsumer create a new consumer for a specific name using the config provided.
func (f *Factory) CreateConsumer(name string) (*consumer, error) {
	cfg, ok := f.config.Consumers[name]
	if !ok {
		return nil, errors.Errorf("consumer \"%s\" did not exist", name)
	}
	return f.newConsumer(name, cfg)
}

// Name return the factory name
func (f *Factory) Name() string {
	return "rabbitmq"
}

func (f *Factory) newConsumer(name string, cfg ConsumerConfig) (*consumer, error) {
	conn, ok := f.conns[cfg.Connection]
	if !ok {
		available := []string{}
		for cname := range f.conns {
			available = append(available, cname)
		}
		return nil, errors.Errorf(
			"connection name for consumer(%s) did not exist, connections names available: %s",
			name,
			strings.Join(available, ", "))
	}
	f.log.Debug("opening one connection channel", zap.String("connection", cfg.Connection))
	ch, err := conn.Channel()
	if nil != err {
		return nil, errors.Wrap(err, "failed to open the rabbitMQ channel")
	}

	f.log.Debug("declaring a queue", zap.String("queue", cfg.Queue.Name), zap.String("consumer", name))
	q, err := ch.QueueDeclare(
		cfg.Queue.Name,
		cfg.Queue.Options.Durable,
		cfg.Queue.Options.AutoDelete,
		cfg.Queue.Options.Exclusive,
		cfg.Queue.Options.NoWait,
		cfg.Queue.Options.Args)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to declare the queue \"%s\"", cfg.Queue.Name)
	}

	f.log.Debug("adding queue binds", zap.String("queue", q.Name), zap.String("consumer", name))
	for _, b := range cfg.Queue.Bindings {
		err := f.declareExchange(ch, b.Exchange)
		if err != nil {
			return nil, err
		}
		for _, k := range b.RoutingKeys {
			err := ch.QueueBind(q.Name, k, b.Exchange,
				b.Options.NoWait, b.Options.Args)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to bind the queue \"%s\" to exchange: \"%s\"", q.Name, b.Exchange)
			}
		}
	}

	f.log.Debug("setting QoS",
		zap.Int("count", cfg.PrefetchCount),
		zap.Int("size", cfg.PrefetchSize),
		zap.String("consumer", name))
	if err := ch.Qos(cfg.PrefetchCount, cfg.PrefetchSize, false); err != nil {
		return nil, errors.Wrap(err, "failed to set QoS")
	}
	hash, err := hashids.New()
	if err != nil {
		f.log.Warn("Problem generating the hash", zap.Error(err))
	}
	atomic.AddInt64(&f.number, 1)
	hashcounter, err := hash.EncodeInt64([]int64{f.number})
	if err != nil {
		f.log.Warn("Problem generating the hash", zap.Error(err))
	}
	runner, err := runner.New(f.log.With(zap.String("consumer", name)), cfg.Runner)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create the runner")
	}
	return &consumer{
		queue:       cfg.Queue.Name,
		name:        name,
		hash:        hashcounter,
		opts:        cfg.Options,
		factoryName: f.Name(),
		channel:     ch,
		t:           tomb.Tomb{},
		runner:      runner,
	}, nil
}

func (f *Factory) declareExchange(ch *amqp.Channel, name string) error {
	if len(name) == 0 {
		f.log.Warn("received a black exchange. wrong config?")
		return nil
	}
	f.log.Debug("declaring an exchange", zap.String("exchange", name))
	ex, ok := f.config.Exchanges[name]
	if !ok {
		f.log.Warn("exchange config didn't exist, we will try to continue")
		return nil
	}
	err := ch.ExchangeDeclare(
		name,
		ex.Type,
		ex.Options.Durable,
		ex.Options.AutoDelete,
		ex.Options.Internal,
		ex.Options.NoWait,
		ex.Options.Args)
	if nil != err {
		return errors.Wrapf(err, "failed to declare the exchange %s", name)
	}
	return nil
}
