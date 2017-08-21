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
	log.Info().Msg("Channel opened")
	log.Info().Msg("Setting QoS")
	if err := ch.Qos(cnf.PrefetchCount, cnf.PrefetchSize, false); err != nil {
		return nil, fmt.Errorf("Failed to set QoS: %s", err.Error())
	}
	log.Info().Msg("Succeeded setting QoS")
	log.Info().Msgf("Declaring queue \"%s\"", cnf.Queue.Name)
	_, err = ch.QueueDeclare(
		cnf.Queue.Name,
		cnf.Queue.Options.Durable,
		cnf.Queue.Options.AutoDelete,
		cnf.Queue.Options.Exclusive,
		cnf.Queue.Options.NoWait,
		cnf.Queue.Options.Args)

}
