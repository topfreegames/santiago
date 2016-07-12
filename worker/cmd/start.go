// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package cmd

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/topfreegames/santiago/worker/handler"
	"github.com/uber-go/zap"
)

var nsqHost string
var nsqPort int
var nsqLookupPollInterval int
var nsqMaxAttempts int
var nsqMaxInFlight int
var nsqDefaultRequeueDelay int

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "starts a worker",
	Long:  `starts a new worker listening for webhooks`,
	Run: func(cmd *cobra.Command, args []string) {
		logger := zap.NewJSON()
		w := worker.New(
			"webhook",
			nsqHost,
			nsqPort,
			time.Duration(nsqLookupPollInterval)*time.Second,
			nsqMaxAttempts,
			nsqMaxInFlight,
			time.Duration(nsqDefaultRequeueDelay)*time.Second,
			logger,
		)

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
	startCmd.Flags().StringVarP(&nsqHost, "host", "n", "127.0.0.1", "NSQ Lookup host address")
	startCmd.Flags().IntVarP(&nsqPort, "port", "p", 7778, "NSQ Lookup host port")
	startCmd.Flags().IntVarP(&nsqLookupPollInterval, "interval", "i", 15, "NSQ Lookup update interval in seconds")
	startCmd.Flags().IntVarP(&nsqMaxAttempts, "max-attempts", "m", 10, "NSQ max attempts before error")
	startCmd.Flags().IntVarP(&nsqMaxInFlight, "max-inflight", "f", 150, "NSQ max messages in flight")
	startCmd.Flags().IntVarP(&nsqMaxInFlight, "rq-delay", "r", 15, "NSQ default requeue delay in seconds")
}
