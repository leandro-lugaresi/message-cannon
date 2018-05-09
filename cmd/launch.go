package cmd

import (
	"bytes"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/a8m/envsubst"
	"github.com/leandro-lugaresi/hub"
	"github.com/leandro-lugaresi/message-cannon/rabbit"
	"github.com/leandro-lugaresi/message-cannon/subscriber"
	"github.com/leandro-lugaresi/message-cannon/supervisor"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	defaults "gopkg.in/mcuadros/go-defaults.v1"
)

// launchCmd represents the launch command
var launchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Launch will start all the consumers from the config file",
	Long:  `Launch will start all the consumers from the config file `,
	RunE: func(cmd *cobra.Command, args []string) error {
		h := hub.New()
		err := initConfig()
		if err != nil {
			return errors.Wrap(err, "failed initializing the config")
		}

		factories, err := getFactories(h)
		log.Print(factories)
		if err != nil {
			return err
		}

		cancelchan := make(chan struct{})
		var g run.Group
		g.Add(func() error {
			return interrupt(cancelchan)
		}, func(error) {
			close(cancelchan)
		})

		logger := subscriber.NewLogSubscriber(
			os.Stdout,
			h.NonBlockingSubscribe(viper.GetInt("log-buffer"), "*.*.*"),
			viper.GetBool("development"),
		)
		g.Add(logger.Do, func(error) {
			h.Close()
			logger.Stop()
		})

		sup := supervisor.NewManager(viper.GetDuration("interval-checks"), h)
		g.Add(func() error {
			err := sup.Start(factories)
			if err != nil {
				return err
			}
			sup.CheckConsumers(cancelchan)
			return nil
		}, func(error) {
			err := sup.Stop()
			if err != nil {
				log.Printf("supervisor failed stopping: %v", err)
			}
			close(cancelchan)
		})

		log.Println("start actors...")
		return g.Run()
	},
}

func init() {
	launchCmd.Flags().DurationP("interval-checks", "c", 500*time.Millisecond, "this flag set the interval duration of supervisor operations")
	err := viper.BindPFlag("interval-checks", launchCmd.Flags().Lookup("interval-checks"))
	if err != nil {
		log.Fatal(err)
	}

	launchCmd.Flags().BoolP("development", "d", false, "this flag enable the development mode")
	err = viper.BindPFlag("development", launchCmd.Flags().Lookup("development"))
	if err != nil {
		log.Fatal(err)
	}

	launchCmd.Flags().IntP("log-buffer", "b", 300, "this flag set the buffer size of the log and metrics systems")
	err = viper.BindPFlag("log-buffer", launchCmd.Flags().Lookup("log-buffer"))
	if err != nil {
		log.Fatal(err)
	}
	RootCmd.AddCommand(launchCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() error {
	b, err := envsubst.ReadFileRestricted(cfgFile, true, false)
	if err != nil {
		return errors.Wrap(err, "failed to read the file")
	}
	viper.SetConfigType(strings.TrimPrefix(filepath.Ext(cfgFile), "."))
	err = viper.ReadConfig(bytes.NewBuffer(b))
	return errors.Wrap(err, "failed to unmarshal the initial map of configs")
}

func getFactories(h *hub.Hub) ([]supervisor.Factory, error) {
	var factories []supervisor.Factory
	if viper.InConfig("rabbitmq") {
		config := rabbit.Config{}
		err := viper.UnmarshalKey("rabbitmq", &config)
		defaults.SetDefaults(&config)
		if err != nil {
			return factories, errors.Wrap(err, "problem unmarshaling your config into config struct")
		}
		var rFactory *rabbit.Factory
		rFactory, err = rabbit.NewFactory(config, h)
		if err != nil {
			return factories, errors.Wrap(err, "error creating the rabbitMQ factory")
		}
		factories = append(factories, rFactory)
	}
	return factories, nil
}

func interrupt(cancel <-chan struct{}) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-c:
		return errors.Errorf("received signal %s. shutting down...", sig)
	case <-cancel:
		return errors.New("canceled")
	}
}
