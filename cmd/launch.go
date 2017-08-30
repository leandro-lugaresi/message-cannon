package cmd

import (
	"github.com/spf13/cobra"
)

// launchCmd represents the launch command
var launchCmd = &cobra.Command{
	Use:   "launch",
	Short: "Launch will start all the consumers from the config file",
	Long:  `Launch will start all the consumers from the config file `,
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func init() {
	RootCmd.AddCommand(launchCmd)
}
