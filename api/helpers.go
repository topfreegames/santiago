// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package api

import (
	"encoding/json"
	"io/ioutil"

	"github.com/labstack/echo"
)

// FailWith fails with the specified message
func FailWith(status int, message string, c echo.Context) error {
	result, _ := json.Marshal(map[string]interface{}{
		"success": false,
		"reason":  message,
	})
	return c.String(status, string(result))
}

//GetRequestBody from echo context
func GetRequestBody(c echo.Context) (string, error) {
	bodyCache := c.Get("requestBody")
	if bodyCache != nil {
		return bodyCache.(string), nil
	}
	body := c.Request().Body()
	b, err := ioutil.ReadAll(body)
	if err != nil {
		return "", err
	}
	c.Set("requestBody", string(b))
	return string(b), nil
}

//GetRequestJSON as the specified interface from echo context
func GetRequestJSON(payloadStruct interface{}, c echo.Context) error {
	body, err := GetRequestBody(c)
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(body), payloadStruct)
	if err != nil {
		return err
	}

	return nil
}
