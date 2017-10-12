package rabbit

import (
	"strings"
	"sync/atomic"
	"time"

	"gopkg.in/tomb.v2"

	"github.com/leandro-lugaresi/message-cannon/runner"
	"github.com/leandro-lugaresi/message-cannon/supervisor"
	"github.com/pkg/errors"
	"github.com/rafaeljesus/retry-go"
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
	conns := make(map[string]*amqp.Connection)
	for name, cfgConn := range config.Connections {
		conn, err := openConnection(cfgConn.DSN, 3, time.Second)
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
func (f *Factory) CreateConsumers() ([]supervisor.Consumer, error) {
	var consumers []supervisor.Consumer
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
func (f *Factory) CreateConsumer(name string) (supervisor.Consumer, error) {
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
	_, ok := f.conns[cfg.Connection]
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
	var ch *amqp.Channel
	var errCH error
	conn := f.conns[cfg.Connection]
	ch, errCH = conn.Channel()
	if errCH != nil && errCH.Error() == amqp.ErrClosed.Error() {
		cfgConn := f.config.Connections[cfg.Connection]
		conn, err := openConnection(cfgConn.DSN, 5, time.Second)
		if err != nil {
			return nil, errors.Wrapf(err, "error reopening the connection \"%s\"", cfg.Connection)
		}
		f.conns[cfg.Connection] = conn
		ch, errCH = conn.Channel()
	}
	if errCH != nil {
		return nil, errors.Wrap(errCH, "failed to open the rabbitMQ channel")
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

	for _, b := range cfg.Queue.Bindings {
		f.log.Debug("adding queue bind",
			zap.String("queue", q.Name),
			zap.String("consumer", name),
			zap.String("exchange", b.Exchange))
		err = f.declareExchange(ch, b.Exchange)
		if err != nil {
			return nil, err
		}
		for _, k := range b.RoutingKeys {
			err = ch.QueueBind(q.Name, k, b.Exchange,
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
	if err = ch.Qos(cfg.PrefetchCount, cfg.PrefetchSize, false); err != nil {
		return nil, errors.Wrap(err, "failed to set QoS")
	}
	hash := hashids.New()
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
		l:           f.log.With(zap.String("consumer", name)),
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

func openConnection(dsn string, retries int, sleep time.Duration) (*amqp.Connection, error) {
	var conn *amqp.Connection
	err := retry.Do(func() error {
		var err error
		conn, err = amqp.Dial(dsn)
		return err
	}, retries, sleep)
	return conn, err
}
