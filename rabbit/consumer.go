package cmd

import (
	"context"

	"github.com/leandro-lugaresi/rabbit-cannon/config"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/streadway/amqp"
)

type consumer struct {
	config  config.ConsumerConfig
	channel *amqp.Channel
	log     zerolog.Logger
}

func newConsumer(name string, cnf config.ConsumerConfig, conn *amqp.Connection, log zerolog.Logger) (*consumer, error) {
	log.Info().Str("consumer", name).Msg("Opening rabbitMQ channel")
	ch, err := conn.Channel()
	if nil != err {
		return nil, errors.Wrap(err, "Failed to open a RabbitMQ channel")
	}
	if len(cnf.Exchange.Name) > 0 {
		log.Info().Str("consumer", name).Msgf("Declaring exchange \"%s\"", cnf.Exchange.Name)
		err = ch.ExchangeDeclare(
			cnf.Exchange.Name,
			cnf.Exchange.Type,
			cnf.Exchange.Options.Durable,
			cnf.Exchange.Options.AutoDelete,
			cnf.Exchange.Options.Internal,
			cnf.Exchange.Options.NoWait,
			cnf.Exchange.Options.Args)
		if nil != err {
			return nil, errors.Wrapf(err, "Failed to declare the exchange %s", cnf.Exchange.Name)
		}
	}
	log.Info().Str("consumer", name).Msgf("Declaring queue \"%s\"", cnf.Queue.Name)
	q, err := ch.QueueDeclare(
		cnf.Queue.Name,
		cnf.Queue.Options.Durable,
		cnf.Queue.Options.AutoDelete,
		cnf.Queue.Options.Exclusive,
		cnf.Queue.Options.NoWait,
		cnf.Queue.Options.Args)
	if err != nil {
		return nil, err
	}
	log.Info().Str("consumer", name).Msg("Adding queue binds")
	for _, k := range cnf.Queue.RoutingKeys {
		err := ch.QueueBind(q.Name, k, cnf.Exchange.Name, cnf.Queue.BindingOptions.NoWait, cnf.Queue.BindingOptions.Args)
		if nil != err {
			return nil, errors.Wrapf(err, "Failed to bind the queue \"%s\" to exchange: \"%s\"", cnf.Queue.Name, cnf.Exchange.Name)
		}
	}
	log.Info().Str("consumer", name).Msg("Setting QoS")
	if err := ch.Qos(cnf.PrefetchCount, cnf.PrefetchSize, false); err != nil {
		return nil, errors.Wrap(err, "Failed to set QoS")
	}
	return &consumer{cnf, ch, log}, nil
}

func (c *consumer) consume(ctx context.Context) error {
	return nil
}
