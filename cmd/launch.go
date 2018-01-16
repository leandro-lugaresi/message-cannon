package cmd

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/leandro-lugaresi/message-cannon/event"
	"github.com/leandro-lugaresi/message-cannon/rabbit"
	"github.com/leandro-lugaresi/message-cannon/supervisor"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/mcuadros/go-defaults.v1"
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
		sup := supervisor.NewManager(viper.GetDuration("interval-checks"), log)
		var factories []supervisor.Factory
		if viper.InConfig("rabbitmq") {
			config := rabbit.Config{}
			err := viper.UnmarshalKey("rabbitmq", &config)
			defaults.SetDefaults(&config)
			if err != nil {
				return errors.Wrap(err, "Problem unmarshaling your config into config struct")
			}
			var rFactory *rabbit.Factory
			rFactory, err = rabbit.NewFactory(config, log)
			if err != nil {
				return err
			}
			factories = append(factories, rFactory)
		}
		err := sup.Start(factories)
		if err != nil {
			return errors.Wrap(err, "failed on supervisor start")
		}
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

		// Block until a signal is received.
		s := <-sigs
		log.Info("Signal received. shutting down...", event.KV("signal", s.String()))
		err = sup.Stop()
		return err
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
