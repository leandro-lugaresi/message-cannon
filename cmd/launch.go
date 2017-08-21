package cmd

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// launchCmd represents the launch command
var launchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Launch will start all the consumers from the config file",
	Long:  `Launch will start all the consumers from the config file `,
	Run: func(cmd *cobra.Command, args []string) {
		var config Config
		err := viper.Unmarshal(config)
		if err != nil {
			log.Fatal().Msgf("Failed to parse the config file. Err: %s", err)
		}
	},
}

func init() {
	RootCmd.AddCommand(launchCmd)
}
