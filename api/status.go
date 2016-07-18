// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package api

import (
	"encoding/json"

	"github.com/kataras/iris"
	"github.com/uber-go/zap"
)

// StatusHandler is the handler responsible for validating that the app is still up
func StatusHandler(app *App) func(c *iris.Context) {
	return func(c *iris.Context) {
		app.Logger.Debug("Starting status...")

		messageCount, err := app.GetMessageCount()

		if err != nil {
			app.Logger.Error("Status failed!", zap.Error(err))
			c.Write("Status failed")
			c.SetStatusCode(500)
			return
		}

		items, err := json.Marshal(map[string]interface{}{
			"errors":          app.Errors.Rate(),
			"messagesInQueue": messageCount,
		})

		if err != nil {
			app.Logger.Error("Status failed!", zap.Error(err))
			c.Write("Status failed")
			c.SetStatusCode(500)
			return
		}

		c.SetStatusCode(iris.StatusOK)
		c.Write(string(items))
		app.Logger.Debug("Status worked successfully.")
	}
}
