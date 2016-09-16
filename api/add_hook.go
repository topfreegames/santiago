// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package api

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo"
	"github.com/topfreegames/santiago/log"
	"github.com/uber-go/zap"
)

// AddHookHandler sends new hooks
func AddHookHandler(app *App) func(c echo.Context) error {
	return func(c echo.Context) error {
		method := c.QueryParam("method")
		url := c.QueryParam("url")

		l := app.Logger.With(
			zap.String("source", "addHookHandler"),
			zap.String("method", method),
			zap.String("url", url),
			zap.String("queue", app.Queue),
		)

		if method == "" || url == "" {
			l.Warn("Request validation failed.")
			return FailWith(http.StatusBadRequest, "Both 'method' and 'url' must be provided as querystring parameters", c)
		}

		log.D(l, "Sending hook to queue...")
		payload, err := GetRequestBody(c)
		if err != nil {
			msg := "Failed to retrieve payload in request body."
			l.Error(msg, zap.Error(err))
			return FailWith(http.StatusBadRequest, msg, c)
		}

		err = app.PublishHook(method, url, payload)
		if err != nil {
			l.Error("Hook failed to be published.", zap.Error(err))
			return FailWith(500, fmt.Sprintf("Hook failed to be published (%s).", err.Error()), c)
		}

		log.D(l, "Hook sent to queue successfully...")
		return c.String(http.StatusOK, "OK")
	}
}
