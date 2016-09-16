// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"time"

	"gopkg.in/redis.v4"

	"github.com/labstack/echo/engine/standard"
	. "github.com/onsi/gomega"
	"github.com/topfreegames/santiago/api"

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
	return api.New(options, logger, false)
}

//Get from server
func Get(app *api.App, url string) (int, string) {
	return doRequest(app, "GET", url, "")
}

//Post to server
func Post(app *api.App, url, body string) (int, string) {
	return doRequest(app, "POST", url, body)
}

//PostJSON to server
func PostJSON(app *api.App, url string, body interface{}) (int, string) {
	result, err := json.Marshal(body)
	if err != nil {
		return 510, "Failed to marshal specified body to JSON format"
	}
	return Post(app, url, string(result))
}

//Put to server
func Put(app *api.App, url, body string) (int, string) {
	return doRequest(app, "PUT", url, body)
}

//PutJSON to server
func PutJSON(app *api.App, url string, body interface{}) (int, string) {
	result, err := json.Marshal(body)
	if err != nil {
		return 510, "Failed to marshal specified body to JSON format"
	}
	return Put(app, url, string(result))
}

//Delete from server
func Delete(app *api.App, url string) (int, string) {
	return doRequest(app, "DELETE", url, "")
}

var client *http.Client
var transport *http.Transport

func initClient() {
	if client == nil {
		transport = &http.Transport{DisableKeepAlives: true}
		client = &http.Client{Transport: transport}
	}
}

func doRequest(app *api.App, method, url, body string) (int, string) {
	initClient()
	defer transport.CloseIdleConnections()
	app.Engine.SetHandler(app.WebApp)
	ts := httptest.NewServer(app.Engine.(*standard.Server))

	var bodyBuff io.Reader
	if body != "" {
		bodyBuff = bytes.NewBuffer([]byte(body))
	}
	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", ts.URL, url), bodyBuff)
	req.Header.Set("Connection", "close")
	req.Close = true
	Expect(err).NotTo(HaveOccurred())

	res, err := client.Do(req)
	ts.Close()
	//Wait for port of httptest to be reclaimed by OS
	time.Sleep(50 * time.Millisecond)
	Expect(err).NotTo(HaveOccurred())

	b, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	Expect(err).NotTo(HaveOccurred())

	return res.StatusCode, string(b)
}
