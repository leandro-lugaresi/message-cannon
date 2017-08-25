package cmd

import (
	"bytes"
	"context"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/leandro-lugaresi/rabbit-cannon/config"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
)

func Test_consumer(t *testing.T) {
	c := testGetConfig(t, "valid_queue_and_exchange_config.yml")
	conn, err := amqp.Dial(c.Connections["default"].DSN)
	failIfErr(t, err, "Failed to connect to rabbitMQ using "+c.Connections["default"].DSN)

	defer conn.Close()
	var consumer = new(consumer)
	t.Run("Should declare queue and exchanges", func(t *testing.T) {
		consumer, err = newConsumer("test1", c.Consumers["test1"], conn, zerolog.Nop())
		failIfErr(t, err, "Failed to create a new consumer: ")
	})
	t.Run("Should get messages from queue ", func(t *testing.T) {
		ch, err := conn.Channel()
		failIfErr(t, err, "Failed to open another channel ")
		ch.Publish(
			"upload-picture",
			"iphone.upload",
			true,
			false,
			amqp.Publishing{
				Headers:         amqp.Table{},
				ContentType:     "text/plain",
				ContentEncoding: "",
				Body:            []byte(`{"foo": "baz"}`),
				DeliveryMode:    amqp.Transient, // 1=non-persistent, 2=persistent
				Priority:        9,
			},
		)
		var cerr error
		done := make(chan bool)
		go func() {
			cerr = consumer.consume(context.Background())
			done <- true
		}()
		_, err = ch.QueueDeclare("reply", false, false, false, false, amqp.Table{})
		failIfErr(t, err, "Failed to open another channel ")
		delivery, err := ch.Consume("reply", "test", true, true, false, false, amqp.Table{})
		failIfErr(t, err, "Failed to get the reply consumer ")
		msg := <-delivery
		assert.EqualValues(t, []byte(`{"foo": "baz"}`), msg.Body, "Invalid message")
		msg.Ack(false)
		conn.Close()
		<-done
		failIfErr(t, cerr, "Consumer exit with an error: ")
	})
	consumer.channel.QueueDelete(c.Consumers["test1"].Queue.Name, false, false, false)
	consumer.channel.QueueDelete("reply", false, false, false)
	consumer.channel.ExchangeDelete(c.Consumers["test1"].Exchange.Name, false, false)
}

func testGetConfig(t *testing.T, configFile string) config.Config {
	c := config.Config{}
	viper.Reset()
	viper.SetConfigType("yaml")
	viper.SetEnvPrefix("test")
	yaml, err := ioutil.ReadFile(filepath.Join("testdata", configFile))
	failIfErr(t, err, "Failed to read the config file: ")
	viper.ReadConfig(bytes.NewBuffer(yaml))
	viper.AutomaticEnv()

	err = viper.Unmarshal(&c)
	failIfErr(t, err, "Failed to marshal the config struct: ")
	return c
}

func failIfErr(t *testing.T, err error, msg string) {
	if err != nil {
		t.Fatal(msg, err)
	}
}
