package rabbit

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

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
	runner := &mockRunner{count: 0, exitStatus: 0}
	consumer.runner = runner
	consumer.Run()
	assert.NotNil(t, consumer.runner, "Consumer runner must not be null")
	ch, err := factory.conns["default"].Channel()
	failIfErr(t, err, "Error opening a channel")
	for i := 0; i < 10; i++ {
		err = ch.Publish("upload-picture", "android.profile.upload", false, false, amqp.Publishing{
			Body: []byte(`{"fooo": "bazzz"}`),
		})
		failIfErr(t, err, "error publishing to rabbitMQ")
	}
	<-time.After(100 * time.Millisecond)
	assert.EqualValues(t, 10, runner.messagesProcessed())
	for _, cfg := range factory.config.Consumers {
		ch.QueueDelete(cfg.Queue.Name, false, false, false)
	}
	for name := range factory.config.Exchanges {
		ch.ExchangeDelete(name, false, false)
	}
}

type mockRunner struct {
	count      int64
	exitStatus int
}

func (m *mockRunner) Process(ctx context.Context, b []byte) int {
	atomic.AddInt64(&m.count, 1)
	return m.exitStatus
}

func (m *mockRunner) messagesProcessed() int64 {
	return atomic.LoadInt64(&m.count)
}
