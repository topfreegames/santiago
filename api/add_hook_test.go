// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package api_test

import (
	"net/http"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/satori/go.uuid"
	"github.com/topfreegames/santiago/api"
	. "github.com/topfreegames/santiago/testing"
)

var _ = Describe("Add Hook Handler", func() {
	var logger *MockLogger

	BeforeEach(func() {
		logger = NewMockLogger()
	})
	It("should dispatch hook by adding it", func() {
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

		res := api.PostJSON(app, "/hooks", map[string]interface{}{
			"HookMethod": "POST",
			"HookURL":    "http://test.com",
			"HookPayload": map[string]interface{}{
				"test": "qwe",
			},
		})
		Expect(res.Raw().StatusCode).To(Equal(http.StatusOK))
		Expect(res.Body().Raw()).To(Equal("OK"))

		time.Sleep(50 * time.Millisecond)

		Expect(responses["http://test.com"]).NotTo(BeNil())
		payload := responses["http://test.com"].(map[string]interface{})
		Expect(payload["test"].(string)).To(Equal("qwe"))
	})
})
