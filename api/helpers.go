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
	newrelic "github.com/newrelic/go-agent"
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

//GetTX returns new relic transaction
func GetTX(c echo.Context) newrelic.Transaction {
	tx := c.Get("txn")
	if tx == nil {
		return nil
	}

	return tx.(newrelic.Transaction)
}

//WithSegment adds a segment to new relic transaction
func WithSegment(name string, c echo.Context, f func() error) error {
	tx := GetTX(c)
	if tx == nil {
		return f()
	}
	segment := newrelic.StartSegment(tx, name)
	defer segment.End()
	return f()
}
