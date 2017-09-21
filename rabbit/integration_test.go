package rabbit

import (
	"testing"

	"go.uber.org/zap"

	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
)

func Test_consumer(t *testing.T) {
	c := getConfig(t, "valid_queue_and_exchange_config.yml")
	factory, err := NewFactory(c, zap.NewNop())
	failIfErr(t, err, "Failed to create the factory")
	consumer, err := factory.CreateConsumer("test1")
	failIfErr(t, err, "Failed to create all the consumers")
	assert.NotNil(t, consumer.runner, "Consumer runner must not be null")
	ch, err := factory.conns["default"].Channel()
	t.Run("When call run we will process the messages", func(t *testing.T) {
		err = ch.Publish("upload-picture", "android.profile.upload", true, true, amqp.Publishing{
			Body: []byte("{\"fooo\": \"bar\"}"),
		})
		failIfErr(t, err, "error publishing to rabbitMQ")
		consumer.Run()
	})

	failIfErr(t, err, "Error opening a channel")
	for _, cfg := range factory.config.Consumers {
		ch.QueueDelete(cfg.Queue.Name, false, false, false)
	}
	for name := range factory.config.Exchanges {
		ch.ExchangeDelete(name, false, false)
	}
}
