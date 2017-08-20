package cmd

import "time"

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

// Traverses the config tree and fixes option keys name.
func (config Config) normalize() {
	config.Consumers.normalize()
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

func (consumers Consumers) normalize() {
	for _, consumer := range consumers {
		consumer.normalize()
	}
}

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

func (config ConsumerConfig) normalize() {
	config.Exchange.normalize()
	config.Queue.normalize()
	config.Options.normalizeKeys()
}

// ExchangeConfig describes exchange's configuration.
type ExchangeConfig struct {
	Name    string  `mapstructure:"name"`
	Type    string  `mapstructure:"type"`
	Options Options `mapstructure:"options"`
}

func (config ExchangeConfig) normalize() {
	config.Options.normalizeKeys()
}

// QueueConfig describes queue's configuration.
type QueueConfig struct {
	Exchange       string  `mapstructure:"exchange"`
	Name           string  `mapstructure:"name"`
	RoutingKey     string  `mapstructure:"routing_key"`
	BindingOptions Options `mapstructure:"binding_options"`
	Options        Options `mapstructure:"options"`
}

func (config QueueConfig) normalize() {
	config.BindingOptions.normalizeKeys()
	config.Options.normalizeKeys()
}

// Options describes optional configuration.
type Options map[string]interface{}

// Map from lowercase option name to the expected name.
var capitalizationMap = map[string]string{
	"autodelete":       "autoDelete",
	"auto_delete":      "autoDelete",
	"contentencoding":  "contentEncoding",
	"content_encoding": "contentEncoding",
	"contenttype":      "contentType",
	"content_type":     "contentType",
	"deliverymode":     "deliveryMode",
	"delivery_mode":    "deliveryMode",
	"noack":            "noAck",
	"no_ack":           "noAck",
	"nolocal":          "noLocal",
	"no_local":         "noLocal",
	"nowait":           "noWait",
	"no_wait":          "noWait",
}

// By default yaml reader unmarshals keys in lowercase,
// but AMQP client looks for keys in camelcase.
func (options Options) normalizeKeys() {
	for name, value := range options {
		if correctName, needFix := capitalizationMap[name]; needFix {
			delete(options, name)
			options[correctName] = value
		}
	}
}
