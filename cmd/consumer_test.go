package cmd

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/leandro-lugaresi/rabbit-cannon/config"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"github.com/streadway/amqp"
)

func Test_newConsumer(t *testing.T) {
	c := testGetConfig(t, "valid_queue_and_exchange_config.yml")
	conn, err := amqp.Dial(c.Connections["default"].DSN)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var consumer = new(consumer)
	t.Run("Should declare queue and exchanges", func(t *testing.T) {
		consumer, err = newConsumer("test1", c.Consumers["test1"], conn, zerolog.Nop())
		if err != nil {
			t.Fatal(err)
		}
	})
}

func testGetConfig(t *testing.T, configFile string) config.Config {
	c := config.Config{}
	viper.Reset()
	viper.SetConfigType("yaml")
	viper.SetEnvPrefix("test")
	yaml, err := ioutil.ReadFile(filepath.Join("testdata", configFile))
	if err != nil {
		t.Fatal("Failed to read teh config file: ", err)
	}
	viper.ReadConfig(bytes.NewBuffer(yaml))
	viper.AutomaticEnv()

	err = viper.Unmarshal(&c)
	if err != nil {
		t.Fatal("Failed to read the config file: ", err)
	}
	return c
}
