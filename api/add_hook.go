// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package api

import (
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

		payload := string(c.Request.Body())
		app.PublishHook(method, url, payload)
		c.Write("OK")
	}
}
