package cmd

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"encoding/json"
	"time"

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
		tmpDir, err := ioutil.TempDir("", "cannon")
		failIfErr(t, err, "Failed to create the temp directory ")
		b, err := json.Marshal(map[string]string{"file": filepath.Join(tmpDir, "fooo.txt")})
		failIfErr(t, err, "Failed to marshal ")
		ch.Publish(
			"upload-picture",
			"iphone.upload",
			true,
			false,
			amqp.Publishing{
				Headers:         amqp.Table{},
				ContentType:     "text/plain",
				ContentEncoding: "",
				Body:            b,
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
		time.Sleep(time.Duration(10 * time.Millisecond))
		result, err := ioutil.ReadFile(filepath.Join(tmpDir, "fooo.txt"))
		failIfErr(t, err, "Callback didn't create the file: ")

		assert.EqualValues(t, b, result, "Invalid message")
		conn.Close()
		<-done
		failIfErr(t, cerr, "Consumer exit with an error: ")
	})
	os.Remove(t)
	consumer.channel.QueueDelete(c.Consumers["test1"].Queue.Name, false, false, false)
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
