package cmd

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/leandro-lugaresi/message-cannon/rabbit"
	"github.com/leandro-lugaresi/message-cannon/supervisor"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gopkg.in/mcuadros/go-defaults.v1"
)

// launchCmd represents the launch command
var launchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Launch will start all the consumers from the config file",
	Long:  `Launch will start all the consumers from the config file `,
	RunE: func(cmd *cobra.Command, args []string) error {
		log, err := zap.NewProduction()
		if viper.GetBool("development") {
			log, err = zap.NewDevelopment()
		}
		if err != nil {
			return err
		}
		sup := supervisor.NewManager(viper.GetDuration("interval-checks"), log)
		var factories []supervisor.Factory
		if viper.InConfig("rabbitmq") {
			config := rabbit.Config{}
			err = viper.UnmarshalKey("rabbitmq", &config)
			defaults.SetDefaults(&config)
			if err != nil {
				log.Error("Error unmarshaling the config", zap.Error(err))
				return errors.Wrap(err, "Problem unmarshaling your config into config struct")
			}
			rabbitF, err := rabbit.NewFactory(config, log)
			if err != nil {
				log.Error("Error creating the rabbitMQ factory", zap.Error(err))
				return err
			}
			factories = append(factories, rabbitF)
		}
		err = sup.Start(factories)
		if err != nil {
			log.Error("Error starting the supervisor", zap.Error(err))
			return errors.Wrap(err, "failed on supervisor start")
		}
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

		// Block until a signal is received.
		s := <-sigs
		log.Info("Signal received. shutting down...", zap.String("signal", s.String()))
		err = sup.Stop()
		if err != nil {
			log.Error("Error stoping the supervisor", zap.Error(err))
			return err
		}
		return nil
	},
}

func init() {
	launchCmd.Flags().DurationP("interval-checks", "c", 500*time.Millisecond, "this flag set the interval duration of supervisor operations")
	launchCmd.Flags().BoolP("development", "d", false, "this flag enable the development mode")
	err := viper.BindPFlag("interval-checks", launchCmd.Flags().Lookup("interval-checks"))
	if err != nil {
		log.Fatal(err)
	}
	err = viper.BindPFlag("development", launchCmd.Flags().Lookup("development"))
	if err != nil {
		log.Fatal(err)
	}
	RootCmd.AddCommand(launchCmd)
}
