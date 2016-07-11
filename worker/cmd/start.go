// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/topfreegames/santiago/worker/handler"
)

var nsqHost string
var nsqPort int

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "starts a worker",
	Long:  `starts a new worker listening for webhooks`,
	Run: func(cmd *cobra.Command, args []string) {
		w := &worker.Worker{
			NSQLookupDHost: nsqHost,
			NSQLookupDPort: nsqPort,
		}

		w.Start()
	},
}

func init() {
	RootCmd.AddCommand(startCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// startCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	startCmd.Flags().StringVarP(&nsqHost, "nsqhost", "n", "127.0.0.1", "NSQ Lookup host address")
	startCmd.Flags().IntVarP(&nsqPort, "nsqport", "p", 7778, "NSQ Lookup host port")
}
