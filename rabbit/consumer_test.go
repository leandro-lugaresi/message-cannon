package rabbit

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/leandro-lugaresi/message-cannon/supervisor"
	rabbithole "github.com/michaelklishin/rabbit-hole"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
)

func Test_consumer(t *testing.T) {
	c := getConfig(t, "valid_queue_and_exchange_config.yml")
	factory, err := NewFactory(c, zap.NewNop())
	failIfErr(t, err, "Failed to create the factory")
	cons, err := factory.CreateConsumer("test1")
	failIfErr(t, err, "Failed to create all the consumers")
	assert.NotNil(t, cons.(*consumer).runner, "Consumer runner must not be null")
	runner := &mockRunner{count: 0, exitStatus: 0}
	cons.(*consumer).runner = runner
	cons.Run()
	ch, err := factory.conns["default"].Channel()
	failIfErr(t, err, "Error opening a channel")
	for i := 0; i < 10; i++ {
		err = ch.Publish("upload-picture", "android.profile.upload", false, false, amqp.Publishing{
			Body: []byte(`{"fooo": "bazzz"}`),
		})
		failIfErr(t, err, "error publishing to rabbitMQ")
	}
	<-time.After(200 * time.Millisecond)
	assert.EqualValues(t, 10, runner.messagesProcessed())
	for _, cfg := range factory.config.Consumers {
		_, err := ch.QueueDelete(cfg.Queue.Name, false, false, false)
		failIfErr(t, err)
	}
	for name := range factory.config.Exchanges {
		err := ch.ExchangeDelete(name, false, false)
		failIfErr(t, err)
	}
}

func Test_consumer_reconnect(t *testing.T) {
	// Initial state
	c := getConfig(t, "valid_queue_and_exchange_config.yml")
	log, _ := zap.NewDevelopment()
	factory, err := NewFactory(c, log)
	failIfErr(t, err, "Failed to create the factory")

	// start the supervisor
	sup := supervisor.NewManager(50*time.Millisecond, log)
	err = sup.Start([]supervisor.Factory{factory})
	failIfErr(t, err, "Failed to start the supervisor")

	// Emulate added some load
	ch, err := factory.conns["default"].Channel()
	failIfErr(t, err, "Error opening a channel")
	for i := 0; i < 10; i++ {
		err = ch.Publish("upload-picture", "android.profile.upload", false, false, amqp.Publishing{
			Body: []byte(fmt.Sprintf(`{"info": %d, "exitcode": 0, "sleep": 1000}`, i)),
		})
		failIfErr(t, err, "error publishing to rabbitMQ")
	}

	// get the http client and force close the connection
	client, err := rabbithole.NewClient("http://127.0.0.1:15672", "guest", "guest")
	failIfErr(t, err, "Fail to create the rabbithole client")
	conns, err := client.ListConnections()
	failIfErr(t, err, "fail to get all the connections")
	for _, conn := range conns {
		_, err = client.CloseConnection(conn.Name)
		failIfErr(t, err, "fail to close the connection ", conn.Name)
	}

	// Wait the supervisor reconect the dead consumer and process the messages
	time.Sleep(4 * time.Second)
	queueInf, err := client.GetQueue("/", "upload-picture")
	failIfErr(t, err, "fail to get the queue info")
	assert.Equal(t, 0, queueInf.Messages, "We expect all the messages processed")
	assert.Equal(t, 1, queueInf.Consumers, "We expect one consumer up and running")

	_, err = client.DeleteQueue("/", "upload-picture")
	failIfErr(t, err, "failed to delete the queue")
	err = sup.Stop()
	failIfErr(t, err, "fail to close the supervisor and consumers")
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
