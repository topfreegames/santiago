// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package api

import (
	"encoding/json"
	"net/http"

	"github.com/labstack/echo"
	"github.com/topfreegames/santiago/log"
	"github.com/uber-go/zap"
)

// StatusHandler is the handler responsible for validating that the app is still up
func StatusHandler(app *App) func(c echo.Context) error {
	return func(c echo.Context) error {
		log.D(app.Logger, "Starting status...")

		messageCount, err := app.GetMessageCount()

		if err != nil {
			msg := "Status failed"
			log.E(app.Logger, msg, func(cm log.CM) {
				cm.Write(zap.Error(err))
			})
			return FailWith(500, msg, c)
		}

		items, err := json.Marshal(map[string]interface{}{
			"errors":          app.Errors.Rate(),
			"messagesInQueue": messageCount,
		})

		if err != nil {
			msg := "Status failed"
			log.E(app.Logger, msg, func(cm log.CM) {
				cm.Write(zap.Error(err))
			})
			return FailWith(500, msg, c)
		}

		log.D(app.Logger, "Status worked successfully.")
		return c.String(http.StatusOK, string(items))
	}
}
