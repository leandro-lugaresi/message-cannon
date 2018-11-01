package rabbit

import (
	"fmt"
	"time"

	"github.com/creasty/defaults"
	"github.com/leandro-lugaresi/message-cannon/runner"
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
	// Connections describe the connections used by consumers.
	Connections map[string]Connection `mapstructure:"connections" default:"{}"`
	// Exchanges have all the exchanges used by consumers.
	// This exchanges are declared on startup of the rabbitMQ factory.
	Exchanges map[string]ExchangeConfig `mapstructure:"exchanges" default:"{}"`
	// DeadLetters have all the deadletters queues used internally by other queues
	// This will be declared at startup of the rabbitMQ factory
	DeadLetters map[string]DeadLetter `mapstructure:"dead_letters" default:"{}"`
	// Consumers describes configuration list for consumers.
	Consumers map[string]ConsumerConfig `mapstructure:"consumers" default:"{}"`
	//Versioning internal config - used to mount the user agents
	Version string
}

// Connection describe a config for one connection.
type Connection struct {
	DSN     string        `mapstructure:"dsn"`
	Timeout time.Duration `mapstructure:"timeout" default:"2s"`
	Sleep   time.Duration `mapstructure:"sleep" default:"500ms"`
	Retries int           `mapstructure:"retries" default:"5"`
}

// ConsumerConfig describes consumer's configuration.
type ConsumerConfig struct {
	Connection    string        `mapstructure:"connection"`
	MaxWorkers    int           `mapstructure:"workers" default:"1"`
	PrefetchCount int           `mapstructure:"prefetch_count" default:"10"`
	DeadLetter    string        `mapstructure:"dead_letter"`
	Queue         QueueConfig   `mapstructure:"queue"`
	Options       Options       `mapstructure:"options"`
	Runner        runner.Config `mapstructure:"runner"`
}

// ExchangeConfig describes exchange's configuration.
type ExchangeConfig struct {
	Type    string  `mapstructure:"type"`
	Options Options `mapstructure:"options"`
}

// DeadLetter describe all the dead letters queues to be declared before declare other queues.
type DeadLetter struct {
	Queue QueueConfig `mapstructure:"queue"`
}

// QueueConfig describes queue's configuration.
type QueueConfig struct {
	Name     string    `mapstructure:"name"`
	Bindings []Binding `mapstructure:"bindings"`
	Options  Options   `mapstructure:"options"`
}

// Binding describe how a queue connects to a exchange.
type Binding struct {
	Exchange    string   `mapstructure:"exchange"`
	RoutingKeys []string `mapstructure:"routing_keys"`
	Options     Options  `mapstructure:"options"`
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
	Args       amqp.Table `mapstructure:"args" default:"{}"`
}

func setConfigDefaults(config *Config) error {
	if err := defaults.Set(config); err != nil {
		return err
	}

	for k, cfg := range config.Connections {
		if err := defaults.Set(&cfg); err != nil {
			return err
		}
		config.Connections[k] = cfg
	}

	for k, cfg := range config.Consumers {
		if err := defaults.Set(&cfg); err != nil {
			return err
		}
		if len(cfg.Runner.Options.Headers) == 0 {
			cfg.Runner.Options.Headers = map[string]string{}
		}
		_, exist := cfg.Runner.Options.Headers["User-Agent"]
		if cfg.Runner.Type == "http" && !exist {
			cfg.Runner.Options.Headers["User-Agent"] = fmt.Sprint("message-cannon/", config.Version)
		}
		config.Consumers[k] = cfg
	}

	for k, cfg := range config.DeadLetters {
		if err := defaults.Set(&cfg); err != nil {
			return err
		}
		config.DeadLetters[k] = cfg
	}

	for k, cfg := range config.Exchanges {
		if err := defaults.Set(&cfg); err != nil {
			return err
		}
		config.Exchanges[k] = cfg
	}
	return nil
}
