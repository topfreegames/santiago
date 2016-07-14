// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package api

import (
	"fmt"
	"net/http"

	. "github.com/onsi/gomega"

	"github.com/gavv/httpexpect"
	"github.com/uber-go/zap"
)

// GetDefaultTestApp returns a new Santiago API Application bound to 0.0.0.0:8888 for test
func GetDefaultTestApp(logger zap.Logger) (*App, error) {
	options := DefaultOptions()
	return New(options, logger)
}

// Get returns a test request against specified URL
func Get(app *App, url string) *httpexpect.Response {
	req := sendRequest(app, "GET", url)
	return req.Expect()
}

// PostBody returns a test request against specified URL
func PostBody(app *App, url string, payload string) *httpexpect.Response {
	return sendBody(app, "POST", url, payload)
}

func sendBody(app *App, method string, url string, payload string) *httpexpect.Response {
	req := sendRequest(app, method, url)
	return req.WithBytes([]byte(payload)).Expect()
}

// PostJSON returns a test request against specified URL
func PostJSON(app *App, url string, payload map[string]interface{}, querystring ...map[string]string) *httpexpect.Response {
	return sendJSON(app, "POST", url, payload, querystring...)
}

func sendJSON(app *App, method, url string, payload map[string]interface{}, querystring ...map[string]string) *httpexpect.Response {
	req := sendRequest(app, method, url)
	if len(querystring) > 0 {
		qs := querystring[0]
		for q, v := range qs {
			req.WithQuery(q, v)
		}
	}
	return req.WithJSON(payload).Expect()
}

//GinkgoReporter implements tests for httpexpect
type GinkgoReporter struct {
}

// Errorf implements Reporter.Errorf.
func (g *GinkgoReporter) Errorf(message string, args ...interface{}) {
	Expect(false).To(BeTrue(), fmt.Sprintf(message, args...))
}

//GinkgoPrinter reports errors to stdout
type GinkgoPrinter struct{}

//Logf reports to stdout
func (g *GinkgoPrinter) Logf(source string, args ...interface{}) {
	fmt.Printf(source, args...)
}

func sendRequest(app *App, method, url string) *httpexpect.Request {
	handler := app.WebApp.NoListen().Handler

	e := httpexpect.WithConfig(httpexpect.Config{
		BaseURL: "http://example.com",
		Client: &http.Client{
			Transport: httpexpect.NewFastBinder(handler),
			Jar:       httpexpect.NewJar(),
		},
		Reporter: &GinkgoReporter{},
		Printers: []httpexpect.Printer{
		//httpexpect.NewDebugPrinter(&GinkgoPrinter{}, true),
		},
	})

	return e.Request(method, url)
}
