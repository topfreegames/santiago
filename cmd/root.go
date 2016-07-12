// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

//APIConfigurationFile path
var APIConfigurationFile string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "snt",
	Short: "snt runs Santiago API",
	Long:  `snt runs Santiago API used to receive hooks from other apps`,
}

//Execute runs the specified command
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringVarP(&APIConfigurationFile, "config", "c", "./config/default.yaml", "config file")
}
