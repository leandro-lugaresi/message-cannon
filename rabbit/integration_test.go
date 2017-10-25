package rabbit

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/leandro-lugaresi/message-cannon/supervisor"
	rabbithole "github.com/michaelklishin/rabbit-hole"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/ory-am/dockertest.v3"
)

func TestIntegrationSuite(t *testing.T) {
	// -> Setup
	dockerPool, err := dockertest.NewPool("")
	require.NoError(t, err, "Coud not connect to docker")
	resource, err := dockerPool.Run("rabbitmq", "3.6.12-management", []string{})
	require.NoError(t, err, "Could not start resource")
	config := getConfig(t, "valid_queue_and_exchange_config.yml")
	cfg := config.Connections["default"]
	cfg.DSN = fmt.Sprintf("amqp://localhost:%s", resource.GetPort("5672/tcp"))
	config.Connections["default"] = cfg
	// -> TearDown
	defer func() {
		if err := dockerPool.Purge(resource); err != nil {
			t.Errorf("Could not purge resource: %s", err)
		}
	}()
	t.Run("TestFactoryShouldReturnConnectionErrors", func(t *testing.T) {
		c := getConfig(t, "valid_queue_and_exchange_config.yml")
		t.Run("when we pass an invalid port", func(t *testing.T) {
			conn := c.Connections["default"]
			conn.DSN = "amqp://guest:guest@localhost:80/"
			c.Connections["default"] = conn
			_, err := NewFactory(c, zap.NewNop())
			require.NotNil(t, err)
			assert.Contains(t, err.Error(), "error opening the connection \"default\": ")
		})
		t.Run("when we pass an invalid host", func(t *testing.T) {
			conn := c.Connections["default"]
			conn.DSN = "amqp://guest:guest@10.255.255.1:5672/"
			c.Connections["default"] = conn
			_, err := NewFactory(c, zap.NewNop())
			assert.EqualError(t, err, "error opening the connection \"default\": dial tcp 10.255.255.1:5672: i/o timeout")
		})
	})
	t.Run("TestFactory", func(t *testing.T) {
		c := getConfig(t, "valid_two_connections_config.yml")
		conn := c.Connections["default"]
		conn.DSN = fmt.Sprintf("amqp://localhost:%s", resource.GetPort("5672/tcp"))
		c.Connections["default"] = conn
		conn = c.Connections["test1"]
		conn.DSN = fmt.Sprintf("amqp://localhost:%s", resource.GetPort("5672/tcp"))
		c.Connections["test1"] = conn
		factory, err := NewFactory(c, zap.NewNop())
		require.NoError(t, err, "Failed to create the rabbitMQ factory")
		require.Len(t, factory.conns, 2)
		var consumers []supervisor.Consumer
		consumers, err = factory.CreateConsumers()
		require.NoError(t, err, "Failed to create all the consumers")
		require.Len(t, consumers, 2, "When call CreateConsumers and we got all the consumers from config")
		for _, consumer := range consumers {
			assert.True(t, consumer.Alive(), "The consumer ", consumer.Name(), "is not alive")
		}

		var consumer supervisor.Consumer
		consumer, err = factory.CreateConsumer("test1")
		require.NoError(t, err, "Failed to create all the consumers")
		require.NotNil(t, consumer)
		assert.True(t, consumer.Alive(), "The consumer ", consumer.Name(), "is not alive")
		ch, err := factory.conns["default"].Channel()
		require.NoError(t, err, "Error opening a channel")
		for _, cfg := range factory.config.Consumers {
			_, err := ch.QueueDelete(cfg.Queue.Name, false, false, false)
			require.NoError(t, err)
		}
		for name := range factory.config.Exchanges {
			err := ch.ExchangeDelete(name, false, false)
			require.NoError(t, err)
		}
	})
	t.Run("TestConsumerProcess", func(t *testing.T) {
		factory, err := NewFactory(config, zap.NewNop())
		require.NoError(t, err, "Failed to create the factory")
		cons, err := factory.CreateConsumer("test1")
		require.NoError(t, err, "Failed to create all the consumers")
		assert.NotNil(t, cons.(*consumer).runner, "Consumer runner must not be null")
		runner := &mockRunner{count: 0, exitStatus: 0}
		cons.(*consumer).runner = runner
		cons.Run()
		ch, err := factory.conns["default"].Channel()
		require.NoError(t, err, "Error opening a channel")
		for i := 0; i < 5; i++ {
			err = ch.Publish("upload-picture", "android.profile.upload", false, false, amqp.Publishing{
				Body: []byte(`{"fooo": "bazzz"}`),
			})
			require.NoError(t, err, "error publishing to rabbitMQ")
		}
		<-time.After(400 * time.Millisecond)
		assert.EqualValues(t, 5, runner.messagesProcessed())
		for _, cfg := range factory.config.Consumers {
			_, err := ch.QueueDelete(cfg.Queue.Name, false, false, false)
			require.NoError(t, err)
		}
		for name := range factory.config.Exchanges {
			err := ch.ExchangeDelete(name, false, false)
			require.NoError(t, err)
		}
	})
	t.Run("TestConsumerReconnect", func(t *testing.T) {
		stdStderr := os.Stderr
		r, w, err := os.Pipe()
		require.NoError(t, err, "Failed to open an file pipe")
		os.Stderr = w
		log, err := zap.NewDevelopment()
		require.NoError(t, err, "Failed to create the log")
		factory, err := NewFactory(config, log)
		require.NoError(t, err, "Failed to create the factory")

		// start the supervisor
		sup := supervisor.NewManager(10*time.Millisecond, log)
		err = sup.Start([]supervisor.Factory{factory})
		require.NoError(t, err, "Failed to start the supervisor")
		sendMessages(t, resource, "upload-picture", "android.profile.upload", 1, 3)
		time.Sleep(5 * time.Second)

		// get the http client and force to close all the connections
		client, err := rabbithole.NewClient(fmt.Sprintf("http://localhost:%s", resource.GetPort("15672/tcp")), "guest", "guest")
		require.NoError(t, err, "Fail to create the rabbithole client")
		conns, err := client.ListConnections()
		require.NoError(t, err, "fail to get all the connections")
		t.Logf("Found %d open connections", len(conns))
		for _, conn := range conns {
			_, err = client.CloseConnection(conn.Name)
			require.NoError(t, err, "fail to close the connection ", conn.Name)
		}
		time.Sleep(2 * time.Second)

		// Emulate added some load
		sendMessages(t, resource, "upload-picture", "android.profile.upload", 4, 6)
		// wait the supervisor restart the consumer
		time.Sleep(1 * time.Second)
		err = sup.Stop()
		require.NoError(t, err, "Failed to close the supervisor")
		// Verify the log output
		err = w.Close()
		require.NoError(t, err, "Failed closing the pipe")
		out, err := ioutil.ReadAll(r)
		require.NoError(t, err, "failed to get the log output")
		assert.Contains(t, string(out), `{"consumer": "test1", "output": "3"}`)
		assert.Contains(t, string(out), "Recreating the consumer")
		assert.Contains(t, string(out), `{"consumer": "test1", "output": "6"}`)

		_, err = client.DeleteQueue("/", "upload-picture")
		require.NoError(t, err, "failed to delete the queue")
		err = sup.Stop()
		require.NoError(t, err, "fail to close the supervisor and consumers")
		os.Stderr = stdStderr
	})
}

func sendMessages(t *testing.T, resource *dockertest.Resource, ex, key string, start, count int) {
	conn, err := amqp.Dial(fmt.Sprintf("amqp://localhost:%s", resource.GetPort("5672/tcp")))
	require.NoError(t, err, "failed to open a new connection for tests")
	ch, err := conn.Channel()
	require.NoError(t, err, "failed to open a channel for tests")

	for i := start; i <= count; i++ {
		err := ch.Publish(ex, key, false, false, amqp.Publishing{
			Body: []byte(fmt.Sprintf(`{"info": %d, "exitcode": 0, "sleep": 1000}`, i)),
		})
		require.NoError(t, err, "error publishing to rabbitMQ")
	}
}

func getConfig(t *testing.T, configFile string) Config {
	c := Config{}
	viper.Reset()
	viper.SetConfigType("yaml")
	viper.SetEnvPrefix("test")
	yaml, err := ioutil.ReadFile(filepath.Join("testdata", configFile))
	assert.NoError(t, err, "Failed to read the config file: ")
	err = viper.ReadConfig(bytes.NewBuffer(yaml))
	assert.NoError(t, err)
	viper.AutomaticEnv()

	err = viper.UnmarshalKey("rabbitmq", &c)
	assert.NoError(t, err, "Failed to marshal the config struct: ")
	return c
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
