package rabbit

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/leandro-lugaresi/hub"
	"github.com/leandro-lugaresi/message-cannon/runner"
	"github.com/leandro-lugaresi/message-cannon/supervisor"
	rabbithole "github.com/michaelklishin/rabbit-hole"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/ory-am/dockertest.v3"
)

func TestIntegrationSuite(t *testing.T) {
	tests := []struct {
		scenario string
		function func(*testing.T, *dockertest.Resource)
	}{
		{
			scenario: "validate the behavior when we have connection trouble",
			function: testFactoryShouldReturnConnectionErrors,
		},
		{
			scenario: "validate a factory with two connections",
			function: testFactoryWithTwoConnections,
		},
		{
			scenario: "validate the behavior of one healthy consumer",
			function: testConsumerProcess,
		},
		{
			scenario: "validate that all the consumers will restart without problems",
			function: testConsumerReconnect,
		},
	}
	// -> Setup
	dockerPool, err := dockertest.NewPool("")
	require.NoError(t, err, "Coud not connect to docker")
	resource, err := dockerPool.Run("rabbitmq", "3.6.12-management", []string{})
	require.NoError(t, err, "Could not start resource")
	// -> TearDown
	defer func() {
		if err := dockerPool.Purge(resource); err != nil {
			t.Errorf("Could not purge resource: %s", err)
		}
	}()
	// -> Run!
	for _, test := range tests {
		t.Run(test.scenario, func(st *testing.T) {
			test.function(st, resource)
		})
	}
}

func testFactoryShouldReturnConnectionErrors(t *testing.T, resource *dockertest.Resource) {
	c := getConfig(t, "valid_queue_and_exchange_config.yml")
	t.Run("when we pass an invalid port", func(t *testing.T) {
		conn := c.Connections["default"]
		conn.DSN = "amqp://guest:guest@localhost:80/"
		c.Connections["default"] = conn
		_, err := NewFactory(c, hub.New())
		require.NotNil(t, err)
		assert.Contains(t, err.Error(), "error opening the connection \"default\": ")
	})
	t.Run("when we pass an invalid host", func(t *testing.T) {
		conn := c.Connections["default"]
		conn.DSN = "amqp://guest:guest@10.255.255.1:5672/"
		c.Connections["default"] = conn
		_, err := NewFactory(c, hub.New())
		assert.EqualError(t, err, "error opening the connection \"default\": dial tcp 10.255.255.1:5672: i/o timeout")
	})
}

func testFactoryWithTwoConnections(t *testing.T, resource *dockertest.Resource) {
	c := getConfig(t, "valid_two_connections_config.yml")
	c.Connections["default"] = setDSN(resource, c.Connections["default"])
	c.Connections["test1"] = setDSN(resource, c.Connections["test1"])
	factory, err := NewFactory(c, hub.New())
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
}

func testConsumerProcess(t *testing.T, resource *dockertest.Resource) {
	config := getConfig(t, "valid_queue_and_exchange_config.yml")
	config.Connections["default"] = setDSN(resource, config.Connections["default"])
	factory, err := NewFactory(config, hub.New())
	require.NoError(t, err, "Failed to create the factory")
	cons, err := factory.CreateConsumer("test1")
	require.NoError(t, err, "Failed to create all the consumers")
	assert.NotNil(t, cons.(*consumer).runner, "Consumer runner must not be null")
	mock := &mockRunner{count: 0, exitStatus: 0}
	cons.(*consumer).runner = mock
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
	assert.EqualValues(t, 5, mock.messagesProcessed())
	for _, cfg := range factory.config.Consumers {
		_, err := ch.QueueDelete(cfg.Queue.Name, false, false, false)
		require.NoError(t, err)
	}
	for name := range factory.config.Exchanges {
		err := ch.ExchangeDelete(name, false, false)
		require.NoError(t, err)
	}
}

func testConsumerReconnect(t *testing.T, resource *dockertest.Resource) {
	config := getConfig(t, "valid_queue_and_exchange_config.yml")
	config.Connections["default"] = setDSN(resource, config.Connections["default"])
	h := hub.New()
	reconectionSubscriber := h.Subscribe(10, "supervisor.recreating_consumer.*")
	processSubscriber := h.Subscribe(10, "rabbit.process.sucess")
	factory, err := NewFactory(config, h)
	require.NoError(t, err, "Failed to create the factory")

	// start the supervisor
	sup := supervisor.NewManager(10*time.Millisecond, h)
	err = sup.Start([]supervisor.Factory{factory})
	require.NoError(t, err, "Failed to start the supervisor")
	sendMessages(t, resource, "upload-picture", "android.profile.upload", 1, 3)
	time.Sleep(5 * time.Second)

	// get the http client and force to close all the connections
	go closeRabbitMQConnections(t, resource)
	//receive the message of consumer reconnect
	<-reconectionSubscriber.Receiver

	// send new messages
	sendMessages(t, resource, "upload-picture", "android.profile.upload", 4, 6)
	time.Sleep(1 * time.Second)
	err = sup.Stop()
	require.NoError(t, err, "Failed to close the supervisor")
	// Verify if process all 6 messages
	require.Len(t, processSubscriber.Receiver, 6, "processSubscriber should have 6 messages")
	err = sup.Stop()
	require.NoError(t, err, "fail to close the supervisor and consumers")
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

func setDSN(resource *dockertest.Resource, conn Connection) Connection {
	conn.DSN = fmt.Sprintf("amqp://localhost:%s", resource.GetPort("5672/tcp"))
	return conn
}

func closeRabbitMQConnections(t *testing.T, resource *dockertest.Resource) {
	client, err := rabbithole.NewClient(fmt.Sprintf("http://localhost:%s", resource.GetPort("15672/tcp")), "guest", "guest")
	require.NoError(t, err, "Fail to create the rabbithole client")
	conns, err := client.ListConnections()
	require.NoError(t, err, "fail to get all the connections")
	t.Logf("Found %d open connections", len(conns))
	for _, conn := range conns {
		_, err = client.CloseConnection(conn.Name)
		require.NoError(t, err, "fail to close the connection ", conn.Name)
	}
}

type mockRunner struct {
	count      int64
	exitStatus int
}

func (m *mockRunner) Process(ctx context.Context, msg runner.Message) (int, error) {
	atomic.AddInt64(&m.count, 1)
	return m.exitStatus, nil
}

func (m *mockRunner) messagesProcessed() int64 {
	return atomic.LoadInt64(&m.count)
}
