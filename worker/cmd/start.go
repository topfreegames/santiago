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

var redisHost string
var redisPort int
var redisPass string
var redisDB int
var sentryURL string
var backoff int64
var maxAttempts int
var debug bool

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "starts a worker",
	Long:  `starts a new worker listening for webhooks`,
	Run: func(cmd *cobra.Command, args []string) {
		level := zap.WarnLevel
		if debug {
			level = zap.DebugLevel
		}
		logger := zap.NewJSON(level)

		w := worker.New(
			"webhooks",
			redisHost,
			redisPort,
			redisPass,
			redisDB,
			maxAttempts,
			logger,
			debug,
			30*time.Second,
			sentryURL,
			backoff,
			&worker.RealClock{},
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
	startCmd.Flags().StringVarP(&redisHost, "redis-host", "r", "127.0.0.1", "Queue Redis Host")
	startCmd.Flags().IntVarP(&redisPort, "redis-port", "p", 6379, "Queue Redis Port")
	startCmd.Flags().StringVarP(&redisPass, "redis-pass", "s", "", "Queue Redis Password")
	startCmd.Flags().IntVarP(&redisDB, "redis-db", "b", 0, "Queue Redis DB")
	startCmd.Flags().StringVarP(&sentryURL, "sentry-url", "u", "", "Sentry URL to send errors to")
	startCmd.Flags().IntVarP(&maxAttempts, "max-attempts", "m", 15, "Max attempts before giving up on a hook")
	startCmd.Flags().Int64VarP(&backoff, "backoff-ms", "o", 5000, "Exponential backoff before retrying in ms")
	startCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Starts the worker in debug mode")
}
