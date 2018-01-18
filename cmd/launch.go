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
	"github.com/leandro-lugaresi/message-cannon/event"
	"github.com/leandro-lugaresi/message-cannon/rabbit"
	"github.com/leandro-lugaresi/message-cannon/supervisor"
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
		log := event.NewLogger(event.NewZeroLogHandler(
			os.Stdout,
			viper.GetBool("development")), viper.GetInt("event-buffer"))
		err := initConfig()
		if err != nil {
			return errors.Wrap(err, "failed initializing the config")
		}
		sup := supervisor.NewManager(viper.GetDuration("interval-checks"), log)
		var factories []supervisor.Factory
		if viper.InConfig("rabbitmq") {
			config := rabbit.Config{}
			err := viper.UnmarshalKey("rabbitmq", &config)
			defaults.SetDefaults(&config)
			if err != nil {
				return errors.Wrap(err, "problem unmarshaling your config into config struct")
			}
			var rFactory *rabbit.Factory
			rFactory, err = rabbit.NewFactory(config, log)
			if err != nil {
				return errors.Wrap(err, "error creating the rabbitMQ factory")
			}
			factories = append(factories, rFactory)
		}
		err = sup.Start(factories)
		if err != nil {
			return errors.Wrap(err, "error starting the supervisor")
		}
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

		// Block until a signal is received.
		s := <-sigs
		log.Info("signal received. shutting down...", event.KV("signal", s.String()))
		return errors.Wrap(sup.Stop(), "error stopping the supervisor")
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

	launchCmd.Flags().IntP("event-buffer", "b", 300, "this flag set the buffer size of the log and metrics systems")
	err = viper.BindPFlag("event-buffer", launchCmd.Flags().Lookup("event-buffer"))
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
