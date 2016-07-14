// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package api_test

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nsqio/go-nsq"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/satori/go.uuid"
	"github.com/topfreegames/santiago/api"
	"github.com/topfreegames/santiago/extensions"
	. "github.com/topfreegames/santiago/testing"
	"github.com/uber-go/zap"
)

func startListeningNSQ(host string, port int, queue string, logger zap.Logger) (map[string]interface{}, error) {
	responses := map[string]interface{}{
		"errors": []error{},
	}

	nsqLookupPath := fmt.Sprintf("%s:%d", host, port)
	config := nsq.NewConfig()
	config.LookupdPollInterval = 10 * time.Millisecond

	q, err := nsq.NewConsumer(queue, "main", config)
	if err != nil {
		log.Panic("Could not create consumer...")
		return nil, err
	}
	q.SetLogger(&extensions.NSQLogger{Logger: logger}, nsq.LogLevelWarning)

	q.AddHandler(nsq.HandlerFunc(func(msg *nsq.Message) error {
		var obj map[string]interface{}
		err := json.Unmarshal(msg.Body, &obj)
		if err != nil {
			responses["errors"] = append(responses["errors"].([]error), err)
			return err
		}

		responses[obj["url"].(string)] = obj["payload"]
		return nil
	}))

	err = q.ConnectToNSQLookupd(nsqLookupPath)
	if err != nil {
		return nil, err
	}

	return responses, nil
}

var _ = Describe("App", func() {
	var logger *MockLogger

	BeforeEach(func() {
		logger = NewMockLogger()
	})

	Describe("App", func() {
		Describe("App creation", func() {
			It("should create new app", func() {
				app, err := api.New(nil, logger)
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
				app, err := api.New(nil, logger)
				Expect(err).NotTo(HaveOccurred())
				Expect(app).NotTo(BeNil())

				Expect(app.Config.GetString("api.workingText")).To(Equal("WORKING"))
			})
		})

		Describe("App Load Configuration", func() {
			It("Should load configuration from file", func() {
				options := api.DefaultOptions()
				options.ConfigFile = "../config/default.yaml"

				app, err := api.New(options, logger)
				Expect(err).NotTo(HaveOccurred())
				Expect(app.Config).NotTo(BeNil())

				expected := app.Config.GetString("api.workingText")
				Expect(expected).To(Equal("WORKING"))
			})
		})

		Describe("App Submit Hook to NSQ", func() {
			It("Should receive hook", func() {
				options := api.DefaultOptions()
				options.ConfigFile = "../config/default.yaml"

				app, err := api.New(options, logger)
				Expect(err).NotTo(HaveOccurred())
				Expect(app.Config).NotTo(BeNil())

				queueID := uuid.NewV4().String()
				app.Queue = queueID

				responses, err := startListeningNSQ(
					"127.0.0.1",
					7778,
					queueID,
					logger,
				)
				Expect(err).NotTo(HaveOccurred())
				time.Sleep(50 * time.Millisecond)
				payload := map[string]interface{}{
					"x": 1,
				}
				payloadJSON, _ := json.Marshal(payload)

				err = app.PublishHook("POST", "http://test.url.com", string(payloadJSON))
				Expect(err).NotTo(HaveOccurred())

				time.Sleep(50 * time.Millisecond)

				Expect(responses["http://test.url.com"]).NotTo(BeNil())
			})
		})
	})
})
