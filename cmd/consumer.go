package cmd

import (
	"fmt"

	"github.com/leandro-lugaresi/rabbit-cannon/config"
	"github.com/rs/zerolog/log"
	"github.com/streadway/amqp"
)

type consumer struct {
	config  config.ConsumerConfig
	channel *amqp.Channel
}

func newConsumer(cnf config.ConsumerConfig, con *amqp.Connection) (*consumer, error) {
	log.Info().Msg("Opening rabbitMQ channel")
	ch, err := con.Channel()
	if nil != err {
		return nil, fmt.Errorf("Failed to open a RabbitMQ channel: %s", err.Error())
	}
	if len(cnf.Exchange.Name) > 0 {
		log.Info().Msgf("Declaring queue \"%s\"", cnf.Exchange.Name)
		ch.ExchangeDeclare(
			cnf.Exchange.Name,
			cnf.Exchange.Type,
			cnf.Exchange.Options.Durable,
			cnf.Exchange.Options.AutoDelete,
			cnf.Exchange.Options.Internal,
			cnf.Exchange.Options.NoWait,
			cnf.Exchange.Options.Args)
	}
	log.Info().Msgf("Declaring queue \"%s\"", cnf.Queue.Name)
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
	log.Info().Msg("Adding queue binds")
	for _, k := range cnf.Queue.RoutingKeys {
		ch.QueueBind(q.Name, k, cnf.Exchange.Name, cnf.Queue.BindingOptions.NoWait, cnf.Queue.BindingOptions.Args)
	}
	log.Info().Msg("Setting QoS")
	if err := ch.Qos(cnf.PrefetchCount, cnf.PrefetchSize, false); err != nil {
		return nil, fmt.Errorf("Failed to set QoS: %s", err.Error())
	}
	return &consumer{channel: ch, config: cnf}, nil
}
