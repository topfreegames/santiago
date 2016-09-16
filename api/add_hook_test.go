// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package api_test

import (
	"encoding/json"
	"net/http"
	"time"

	"gopkg.in/redis.v4"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/satori/go.uuid"
	. "github.com/topfreegames/santiago/testing"
)

var _ = Describe("Add Hook Handler", func() {
	var logger *MockLogger
	var testClient *redis.Client

	BeforeEach(func() {
		logger = NewMockLogger()
		cli, err := GetTestRedisConn()
		Expect(err).NotTo(HaveOccurred())
		testClient = cli
	})

	It("should dispatch hook by adding it", func() {
		app, err := GetDefaultTestApp(logger)
		Expect(err).NotTo(HaveOccurred())
		Expect(app.Config).NotTo(BeNil())

		queueID := uuid.NewV4().String()
		app.Queue = queueID

		Expect(err).NotTo(HaveOccurred())
		time.Sleep(50 * time.Millisecond)

		status, body := PostJSON(app, "/hooks?method=POST&url=http://test.com", map[string]interface{}{
			"test": "qwe",
		})
		Expect(status).To(Equal(http.StatusOK))
		Expect(body).To(Equal("OK"))

		time.Sleep(50 * time.Millisecond)

		results, err := testClient.BLPop(20*time.Millisecond, queueID).Result()
		Expect(results).To(HaveLen(2))
		Expect(results[0]).To(Equal(queueID))

		var hook map[string]interface{}
		err = json.Unmarshal([]byte(results[1]), &hook)
		Expect(err).NotTo(HaveOccurred())

		Expect(hook["attempts"]).To(BeEquivalentTo(0))
		Expect(hook["method"]).To(BeEquivalentTo("POST"))
		Expect(hook["url"]).To(BeEquivalentTo("http://test.com"))

		var payload map[string]interface{}
		err = json.Unmarshal([]byte(hook["payload"].(string)), &payload)
		Expect(err).NotTo(HaveOccurred())

		Expect(payload["test"]).To(BeEquivalentTo("qwe"))
	})

	Measure("it should add hooks", func(b Benchmarker) {
		app, err := GetDefaultTestApp(logger)
		Expect(err).NotTo(HaveOccurred())

		runtime := b.Time("runtime", func() {
			status, body := PostJSON(app, "/hooks?method=POST&url=http://test.com", map[string]interface{}{
				"test": "qwe",
			})
			Expect(status).To(Equal(http.StatusOK))
			Expect(body).To(Equal("OK"))
		})

		Expect(runtime.Seconds()).Should(BeNumerically("<", 0.1), "Add Hook shouldn't take too long.")
	}, 200)
})
