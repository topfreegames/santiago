// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package api_test

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"gopkg.in/redis.v4"

	. "github.com/onsi/gomega"
	"github.com/topfreegames/santiago/api"
	"github.com/valyala/fasthttp"

	"github.com/gavv/httpexpect"
	"github.com/uber-go/zap"
)

//GetTestRedisConn returns a connection to the test redis server
func GetTestRedisConn() (*redis.Client, error) {
	redisPort := 57575
	redisPortEnv := os.Getenv("REDIS_PORT")
	if redisPortEnv != "" {
		res, err := strconv.ParseInt(redisPortEnv, 10, 32)
		if err != nil {
			return nil, err
		}
		redisPort = int(res)
	}
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("localhost:%d", redisPort),
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	return client, nil
}

// GetDefaultTestApp returns a new Santiago API Application bound to 0.0.0.0:8888 for test
func GetDefaultTestApp(logger zap.Logger) (*api.App, error) {
	options := api.DefaultOptions()
	options.ConfigFile = "../config/test.yaml"
	return api.New(options, logger)
}

// Get returns a test request against specified URL
func Get(app *api.App, url string) *httpexpect.Response {
	req := sendRequest(app, "GET", url)
	return req.Expect()
}

// PostBody returns a test request against specified URL
func PostBody(app *api.App, url string, payload string) *httpexpect.Response {
	return sendBody(app, "POST", url, payload)
}

func sendBody(app *api.App, method string, url string, payload string) *httpexpect.Response {
	req := sendRequest(app, method, url)
	return req.WithBytes([]byte(payload)).Expect()
}

// PostJSON returns a test request against specified URL
func PostJSON(app *api.App, url string, payload map[string]interface{}, querystring ...map[string]string) *httpexpect.Response {
	return sendJSON(app, "POST", url, payload, querystring...)
}

func sendJSON(app *api.App, method, url string, payload map[string]interface{}, querystring ...map[string]string) *httpexpect.Response {
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

func sendRequest(app *api.App, method, url string) *httpexpect.Request {
	api := app.WebApp
	srv := api.Servers.Main()

	if srv == nil { // maybe the user called this after .Listen/ListenTLS/ListenUNIX, the tester can be used as standalone (with no running iris instance) or inside a running instance/app
		srv = api.ListenVirtual(api.Config.Tester.ListeningAddr)
	}

	opened := api.Servers.GetAllOpened()
	h := srv.Handler
	baseURL := srv.FullHost()
	if len(opened) > 1 {
		baseURL = ""
		//we have more than one server, so we will create a handler here and redirect by registered listening addresses
		h = func(reqCtx *fasthttp.RequestCtx) {
			for _, s := range opened {
				if strings.HasPrefix(reqCtx.URI().String(), s.FullHost()) { // yes on :80 should be passed :80 also, this is inneed for multiserver testing
					s.Handler(reqCtx)
					break
				}
			}
		}
	}

	if api.Config.Tester.ExplicitURL {
		baseURL = ""
	}

	testConfiguration := httpexpect.Config{
		BaseURL: baseURL,
		Client: &http.Client{
			Transport: httpexpect.NewFastBinder(h),
			Jar:       httpexpect.NewJar(),
		},
		Reporter: &GinkgoReporter{},
	}
	if api.Config.Tester.Debug {
		testConfiguration.Printers = []httpexpect.Printer{
			httpexpect.NewDebugPrinter(&GinkgoPrinter{}, true),
		}
	}

	return httpexpect.WithConfig(testConfiguration).Request(method, url)
}
