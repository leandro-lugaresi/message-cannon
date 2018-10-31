package rabbit

import (
	"net"
	"strconv"
	"strings"
	"sync/atomic"

	"gopkg.in/tomb.v2"

	"github.com/leandro-lugaresi/hub"
	"github.com/leandro-lugaresi/message-cannon/runner"
	"github.com/leandro-lugaresi/message-cannon/supervisor"
	"github.com/pkg/errors"
	retry "github.com/rafaeljesus/retry-go"
	"github.com/streadway/amqp"
)

// Factory is the block responsible for create consumers and restart the rabbitMQ connections.
type Factory struct {
	config Config
	conns  map[string]*amqp.Connection
	hub    *hub.Hub
	number int64
}

// NewFactory will open the initial connections and start the recover connections procedure.
func NewFactory(config Config, h *hub.Hub) (*Factory, error) {
	setConfigDefaults(&config)
	conns := make(map[string]*amqp.Connection)
	for name, cfgConn := range config.Connections {
		h.Publish(hub.Message{
			Name: "rabbit.opening_connection.info",
			Body: []byte("opening connection with rabbitMQ"),
			Fields: hub.Fields{
				"sleep":      cfgConn.Sleep,
				"timeout":    cfgConn.Timeout,
				"connection": name,
			},
		})
		conn, err := openConnection(cfgConn, config.Version)
		if err != nil {
			return nil, errors.Wrapf(err, "error opening the connection \"%s\"", name)
		}
		conns[name] = conn
	}
	f := &Factory{
		config,
		conns,
		h,
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
	ch, err := f.getChannel(cfg.Connection)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open the rabbitMQ channel for consumer %s", name)
	}
	if len(cfg.DeadLetter) > 0 {
		err = f.declareDeadLetters(ch, cfg.DeadLetter)
		if err != nil {
			return nil, err
		}
	}
	err = f.declareQueue(ch, cfg.Queue)
	if err != nil {
		return nil, err
	}
	f.hub.Publish(hub.Message{
		Name: "rabbit.declare.debug",
		Body: []byte("setting QoS"),
		Fields: hub.Fields{
			"count":    cfg.PrefetchCount,
			"consumer": name,
		},
	})
	if err = ch.Qos(cfg.PrefetchCount, 0, false); err != nil {
		return nil, errors.Wrap(err, "failed to set QoS")
	}

	runner, err := runner.New(cfg.Runner, f.hub.With(hub.Fields{"consumer": name}))
	if err != nil {
		return nil, errors.Wrap(err, "Failed creating a runner")
	}
	f.hub.Publish(hub.Message{
		Name: "rabbit.declare.debug",
		Body: []byte("consumer created"),
		Fields: hub.Fields{
			"max-workers": cfg.MaxWorkers,
			"consumer":    name,
		},
	})
	return &consumer{
		queue:       cfg.Queue.Name,
		name:        name,
		hash:        strconv.FormatInt(atomic.AddInt64(&f.number, 1), 10),
		opts:        cfg.Options,
		factoryName: f.Name(),
		channel:     ch,
		t:           tomb.Tomb{},
		runner:      runner,
		hub:         f.hub.With(hub.Fields{"consumer": name}),
		workerPool:  make(pool, cfg.MaxWorkers),
		timeout:     cfg.Runner.Timeout,
	}, nil
}

func (f *Factory) declareExchange(ch *amqp.Channel, name string) error {
	if len(name) == 0 {
		f.hub.Publish(hub.Message{
			Name:   "rabbit.declare.warning",
			Body:   []byte("receive a blank exchange. Wrong config?"),
			Fields: hub.Fields{},
		})
		return nil
	}
	ex, ok := f.config.Exchanges[name]
	if !ok {
		f.hub.Publish(hub.Message{
			Name:   "rabbit.declare.warning",
			Body:   []byte("exchange config didn't exist, we will try to continue"),
			Fields: hub.Fields{"name": name},
		})
		return nil
	}
	f.hub.Publish(hub.Message{
		Name: "rabbit.declare.info",
		Body: []byte("declaring exchange"),
		Fields: hub.Fields{
			"ex":      name,
			"type":    ex.Type,
			"options": ex.Options,
		},
	})
	err := ch.ExchangeDeclare(
		name,
		ex.Type,
		ex.Options.Durable,
		ex.Options.AutoDelete,
		ex.Options.Internal,
		ex.Options.NoWait,
		assertRightTableTypes(ex.Options.Args))
	if err != nil {
		return errors.Wrapf(err, "failed to declare the exchange %s", name)
	}
	return nil
}

func (f *Factory) declareQueue(ch *amqp.Channel, queue QueueConfig) error {
	f.hub.Publish(hub.Message{
		Name: "rabbit.declare.info",
		Body: []byte("declaring queue"),
		Fields: hub.Fields{
			"queue":   queue.Name,
			"options": queue.Options,
		},
	})
	q, err := ch.QueueDeclare(
		queue.Name,
		queue.Options.Durable,
		queue.Options.AutoDelete,
		queue.Options.Exclusive,
		queue.Options.NoWait,
		assertRightTableTypes(queue.Options.Args))
	if err != nil {
		return errors.Wrapf(err, "failed to declare the queue \"%s\"", queue.Name)
	}

	for _, b := range queue.Bindings {
		f.hub.Publish(hub.Message{
			Name: "rabbit.declare.debug",
			Body: []byte("declaring queue bind"),
			Fields: hub.Fields{
				"queue":    queue.Name,
				"exchange": b.Exchange,
			},
		})
		err = f.declareExchange(ch, b.Exchange)
		if err != nil {
			return err
		}
		for _, k := range b.RoutingKeys {
			err = ch.QueueBind(q.Name, k, b.Exchange,
				b.Options.NoWait, assertRightTableTypes(b.Options.Args))
			if err != nil {
				return errors.Wrapf(err, "failed to bind the queue \"%s\" to exchange: \"%s\"", q.Name, b.Exchange)
			}
		}
	}
	return nil
}

func (f *Factory) declareDeadLetters(ch *amqp.Channel, name string) error {
	f.hub.Publish(hub.Message{
		Name:   "rabbit.declare.debug",
		Body:   []byte("declaring deadletter"),
		Fields: hub.Fields{"dlx": name},
	})
	dead, ok := f.config.DeadLetters[name]
	if !ok {
		f.hub.Publish(hub.Message{
			Name:   "rabbit.declare.warning",
			Body:   []byte("deadletter config didn't exist, we will try to continue"),
			Fields: hub.Fields{"dlx": name},
		})
		return nil
	}
	err := f.declareQueue(ch, dead.Queue)
	return errors.Wrapf(err, "failed to declare the queue for deadletter %s", name)
}

func (f *Factory) getChannel(connectionName string) (*amqp.Channel, error) {
	_, ok := f.conns[connectionName]
	if !ok {
		available := []string{}
		for name := range f.conns {
			available = append(available, name)
		}
		return nil, errors.Errorf(
			"connection (%s) did not exist, connections names available: %s",
			connectionName,
			strings.Join(available, ", "))
	}

	var ch *amqp.Channel
	var errCH error
	conn := f.conns[connectionName]
	ch, errCH = conn.Channel()
	// Reconnect the connection when receive an connection closed error
	if errCH != nil && errCH.Error() == amqp.ErrClosed.Error() {
		cfgConn := f.config.Connections[connectionName]
		f.hub.Publish(hub.Message{
			Name: "rabbit.reopening_connection.info",
			Body: []byte("reopening one connection closed"),
			Fields: hub.Fields{
				"sleep":      cfgConn.Sleep,
				"timeout":    cfgConn.Timeout,
				"connection": connectionName,
			},
		})
		conn, err := openConnection(cfgConn, f.config.Version)
		if err != nil {
			return nil, errors.Wrapf(err, "error reopening the connection \"%s\"", connectionName)
		}
		f.conns[connectionName] = conn
		ch, errCH = conn.Channel()
	}
	return ch, errCH
}

func openConnection(config Connection, version string) (*amqp.Connection, error) {
	var conn *amqp.Connection
	err := retry.Do(func() error {
		var err error
		conn, err = amqp.DialConfig(config.DSN, amqp.Config{
			Dial: func(network, addr string) (net.Conn, error) {
				return net.DialTimeout(network, addr, config.Timeout)
			},
			Properties: amqp.Table{
				"product": "message-cannon",
				"version": version,
			},
		})
		return err
	}, 5, config.Sleep)
	return conn, err
}

func assertRightTableTypes(args amqp.Table) amqp.Table {
	nArgs := amqp.Table{}
	for k, v := range args {
		switch v := v.(type) {
		case int:
			nArgs[k] = int64(v)
		default:
			nArgs[k] = v
		}
	}
	return nArgs
}
