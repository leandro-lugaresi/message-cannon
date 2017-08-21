package config

import (
	"time"

	"github.com/streadway/amqp"
)

// DeliveryMode describes an AMQP message delivery mode.
type DeliveryMode int

// List of available values for `delivery_mode` producer option.
const (
	NonPersistent DeliveryMode = 1
	Persistent                 = 2
)

// Config describes all available options for amqp connection creation.
type Config struct {
	Connections map[string]Connection `mapstructure:"connections"`
	Consumers   Consumers             `mapstructure:"consumers"`
}

// Connections describe the connections used by consumers.
type Connections map[string]Connection

// Connection describe a config for one connection.
type Connection struct {
	DSN            string        `mapstructure:"dsn"`
	ReconnectDelay time.Duration `mapstructure:"reconnect_delay"`
}

// Consumers describes configuration list for consumers.
type Consumers map[string]ConsumerConfig

// ConsumerConfig describes consumer's configuration.
type ConsumerConfig struct {
	Connection    string         `mapstructure:"connection"`
	Workers       int            `mapstructure:"workers"`
	PrefetchCount int            `mapstructure:"prefetch_count"`
	PrefetchSize  int            `mapstructure:"prefetch_size"`
	Exchange      ExchangeConfig `mapstructure:"exchange"`
	Queue         QueueConfig    `mapstructure:"queue"`
	Options       Options        `mapstructure:"options"`
	Callback      string         `mapstructure:"callback"`
}

// ExchangeConfig describes exchange's configuration.
type ExchangeConfig struct {
	Name    string  `mapstructure:"name"`
	Type    string  `mapstructure:"type"`
	Options Options `mapstructure:"options"`
}

// QueueConfig describes queue's configuration.
type QueueConfig struct {
	Name           string   `mapstructure:"name"`
	RoutingKeys    []string `mapstructure:"routing_keys"`
	BindingOptions Options  `mapstructure:"binding_options"`
	Options        Options  `mapstructure:"options"`
}

// Options describes optionals configuration for consumer, queue, bindings and exchanges.
type Options struct {
	Durable    bool       `mapstructure:"durable"`
	Internal   bool       `mapstructure:"internal"`
	AutoDelete bool       `mapstructure:"auto_delete"`
	Exclusive  bool       `mapstructure:"exclusive"`
	NoWait     bool       `mapstructure:"no_wait"`
	NoLocal    bool       `mapstructure:"no_local"`
	AutoAck    bool       `mapstructure:"auto_ack"`
	Args       amqp.Table `mapstructure:"args"`
}
