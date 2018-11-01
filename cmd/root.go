package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "message-cannon",
	Short: "Consume rabbitMQ messages and send to any cli program or http server",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	RootCmd.Version = fmt.Sprintf("%v, commit %v, built at %v", version, commit, date)
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", ".cannon.yaml", "config file (default is .cannon.yaml)")
}
