// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package api

import (
	"fmt"

	"github.com/kataras/iris"
)

type addHookPayload struct {
	HookURL     string
	HookPayload map[string]interface{}
}

// AddHookHandler sends new hooks
func AddHookHandler(app *App) func(c *iris.Context) {
	return func(c *iris.Context) {
		var payload addHookPayload
		if err := c.ReadJSON(&payload); err != nil {
			FailWith(400, err.Error(), c)
			return
		}
		if payload.HookURL == "" {
			FailWith(400, fmt.Errorf("Invalid request: URL can't be empty.").Error(), c)
			return
		}
		app.PublishHook(payload.HookURL, payload.HookPayload)
		c.Write("OK")
	}
}
