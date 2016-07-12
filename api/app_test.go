// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package api_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/topfreegames/santiago/api"
	. "github.com/topfreegames/santiago/testing"
	"github.com/uber-go/zap"
)

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

		Describe("App Initialization", func() {
		})
	})
})
