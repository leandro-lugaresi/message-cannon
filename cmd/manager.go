package cmd

import (
	"github.com/leandro-lugaresi/rabbit-cannon/config"
	"github.com/streadway/amqp"
)

type manager struct {
	connections map[string]*amqp.Connection
	config      config.Config
	consumers   []*consumer
}
