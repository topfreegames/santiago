// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package api

import (
	"strings"

	"github.com/kataras/iris"
	"github.com/uber-go/zap"
)

// HealthCheckHandler is the handler responsible for validating that the app is still up
func HealthCheckHandler(app *App) func(c *iris.Context) {
	return func(c *iris.Context) {
		app.Logger.Debug("Starting healthcheck...")

		_, err := app.Client.Ping().Result()
		if err != nil {
			c.SetStatusCode(500)
			app.Logger.Error("Healthcheck failed", zap.Error(err))
			c.Write("Healthcheck failed")
			return
		}

		workingString := app.Config.GetString("api.workingText")
		c.SetStatusCode(iris.StatusOK)
		workingString = strings.TrimSpace(workingString)
		c.Write(workingString)
		app.Logger.Debug("Everything seems fine!")
	}
}
