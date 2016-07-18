// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package api

import (
	"fmt"

	"github.com/kataras/iris"
	"github.com/uber-go/zap"
)

// AddHookHandler sends new hooks
func AddHookHandler(app *App) func(c *iris.Context) {
	return func(c *iris.Context) {
		method := c.URLParam("method")
		url := c.URLParam("url")

		l := app.Logger.With(
			zap.String("source", "addHookHandler"),
			zap.String("method", method),
			zap.String("url", url),
			zap.String("queue", app.Queue),
		)

		if method == "" || url == "" {
			l.Warn("Request validation failed.")
			FailWith(400, "Both 'method' and 'url' must be provided as querystring parameters", c)
			return
		}

		l.Debug("Sending hook to queue...")
		payload := string(c.Request.Body())

		err := app.PublishHook(method, url, payload)
		if err != nil {
			l.Error("Hook failed to be published.", zap.Error(err))
			FailWith(500, fmt.Sprintf("Hook failed to be published (%s).", err.Error()), c)
			return
		}

		c.Write("OK")
		l.Debug("Hook sent to queue successfully...")
	}
}
