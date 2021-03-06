// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/topfreegames/santiago/api"
	"github.com/uber-go/zap"
)

var apiHost string
var apiPort int
var isDebug bool
var fast bool
var quiet bool

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "starts the API",
	Long:  `starts Santiago API.`,
	Run: func(cmd *cobra.Command, args []string) {
		ll := zap.InfoLevel
		if isDebug {
			ll = zap.DebugLevel
		}
		if quiet {
			ll = zap.ErrorLevel
		}
		logger := zap.New(
			zap.NewJSONEncoder(),
			ll,
		)

		options := api.NewOptions(
			apiHost,
			apiPort,
			isDebug,
			APIConfigurationFile,
		)
		app, err := api.New(options, logger, fast)
		if err != nil {
			log.Fatalf("Could not start server: %s", err)
			return
		}
		app.Start()
	},
}

func init() {
	RootCmd.AddCommand(startCmd)

	startCmd.Flags().StringVarP(&apiHost, "host", "b", "0.0.0.0", "Host to bind API to")
	startCmd.Flags().IntVarP(&apiPort, "port", "p", 3000, "Port to bind API to")
	startCmd.Flags().BoolVarP(&isDebug, "debug", "d", false, "Should Santiago run in debug mode?")
	startCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Should Santiago run in quiet mode? (LOGLEVEL=ERROR)")
	startCmd.Flags().BoolVarP(&fast, "fast", "f", false, "Run with FastHTTP")
}
