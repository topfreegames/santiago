// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package api

import (
	"net/http"
	"strings"

	"github.com/labstack/echo"
	"github.com/uber-go/zap"
)

// HealthCheckHandler is the handler responsible for validating that the app is still up
func HealthCheckHandler(app *App) func(c echo.Context) error {
	return func(c echo.Context) error {
		app.Logger.Debug("Starting healthcheck...")

		_, err := app.Client.Ping().Result()
		if err != nil {
			app.Logger.Error("Healthcheck failed", zap.Error(err))
			return c.String(http.StatusInternalServerError, "Healthcheck failed")
		}

		workingString := app.Config.GetString("api.workingText")
		workingString = strings.TrimSpace(workingString)
		app.Logger.Debug("Everything seems fine!")
		return c.String(http.StatusOK, workingString)
	}
}
