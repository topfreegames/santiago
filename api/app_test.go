// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package api_test

import (
	"encoding/json"
	"time"

	"gopkg.in/redis.v4"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/satori/go.uuid"
	"github.com/topfreegames/santiago/api"
	. "github.com/topfreegames/santiago/testing"
	"github.com/uber-go/zap"
)

var _ = Describe("App", func() {
	var logger *MockLogger
	var testClient *redis.Client

	BeforeEach(func() {
		logger = NewMockLogger()
		cli, err := GetTestRedisConn()
		Expect(err).NotTo(HaveOccurred())
		testClient = cli
	})

	Describe("App", func() {
		Describe("App creation", func() {
			It("should create new app", func() {
				app, err := api.New(nil, logger, false)
				Expect(err).NotTo(HaveOccurred())
				Expect(app).NotTo(BeNil())
				Expect(app.ServerOptions).NotTo(BeNil())
				Expect(app.ServerOptions.Host).To(Equal("0.0.0.0"))
				Expect(app.ServerOptions.Port).To(Equal(3000))
				Expect(app.ServerOptions.Debug).To(BeTrue())

				Expect(app.Config).NotTo(BeNil())

				Expect(logger).To(HaveLogMessage(
					zap.DebugLevel, "Initializing app...",
					"source", "api",
					"host", "0.0.0.0",
					"port", 3000,
					"operation", "initialize",
				))

				Expect(logger).To(HaveLogMessage(
					zap.InfoLevel, "App initialized successfully.",
					"source", "api",
					"host", "0.0.0.0",
					"port", 3000,
					"operation", "initialize",
				))
			})
		})

		Describe("App Default Configurations", func() {
			It("Should set default configurations", func() {
				app, err := api.New(nil, logger, false)
				Expect(err).NotTo(HaveOccurred())
				Expect(app).NotTo(BeNil())

				Expect(app.Config.GetString("api.workingText")).To(Equal("WORKING"))
			})
		})

		Describe("App Load Configuration", func() {
			It("Should load configuration from file", func() {
				options := api.DefaultOptions()
				options.ConfigFile = "../config/default.yaml"

				app, err := api.New(options, logger, false)
				Expect(err).NotTo(HaveOccurred())
				Expect(app.Config).NotTo(BeNil())

				expected := app.Config.GetString("api.workingText")
				Expect(expected).To(Equal("WORKING"))
			})
		})

		Describe("App Submit Hook to Queue", func() {
			It("Should receive hook", func() {
				app, err := GetDefaultTestApp(logger)

				Expect(err).NotTo(HaveOccurred())
				Expect(app.Config).NotTo(BeNil())

				queueID := uuid.NewV4().String()
				app.Queue = queueID

				time.Sleep(50 * time.Millisecond)
				payloadJSON, _ := json.Marshal(map[string]interface{}{
					"x": 1,
				})

				err = app.PublishHook("POST", "http://test.url.com", string(payloadJSON))
				Expect(err).NotTo(HaveOccurred())

				res, err := testClient.BLPop(100*time.Millisecond, queueID).Result()
				Expect(err).NotTo(HaveOccurred())

				Expect(res).To(HaveLen(2))
				Expect(res[0]).To(Equal(queueID))

				var hook map[string]interface{}
				err = json.Unmarshal([]byte(res[1]), &hook)
				Expect(err).NotTo(HaveOccurred())

				Expect(hook["attempts"]).To(BeEquivalentTo(0))
				Expect(hook["method"]).To(BeEquivalentTo("POST"))
				Expect(hook["url"]).To(BeEquivalentTo("http://test.url.com"))

				var payload map[string]interface{}
				err = json.Unmarshal([]byte(hook["payload"].(string)), &payload)
				Expect(err).NotTo(HaveOccurred())

				Expect(payload["x"]).To(BeEquivalentTo(1))
			})
		})
	})
})
