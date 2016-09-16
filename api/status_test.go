// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package api_test

import (
	"encoding/json"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/satori/go.uuid"
	. "github.com/topfreegames/santiago/testing"
)

var _ = Describe("Status Handler", func() {
	var logger *MockLogger

	BeforeEach(func() {
		logger = NewMockLogger()
	})

	It("Should respond with number of items in queue and errors", func() {
		a, err := GetDefaultTestApp(logger)
		a.Queue = uuid.NewV4().String()

		Expect(err).NotTo(HaveOccurred())
		status, body := Get(a, "/status")

		Expect(status).To(Equal(http.StatusOK))

		var obj map[string]interface{}
		err = json.Unmarshal([]byte(body), &obj)
		Expect(obj["errors"]).To(BeEquivalentTo(0.0))
		Expect(obj["messagesInQueue"]).To(BeEquivalentTo(0))
	})

	It("Should respond with number of items in queue when queue has items", func() {
		a, err := GetDefaultTestApp(logger)
		a.Queue = uuid.NewV4().String()

		for i := 0; i < 10; i++ {
			a.Client.LPush(a.Queue, "{\"x\":1}")
		}

		Expect(err).NotTo(HaveOccurred())
		status, body := Get(a, "/status")

		Expect(status).To(Equal(http.StatusOK))

		var obj map[string]interface{}
		err = json.Unmarshal([]byte(body), &obj)
		Expect(obj["errors"]).To(BeEquivalentTo(0.0))
		Expect(obj["messagesInQueue"]).To(BeEquivalentTo(10))
	})

	Measure("it should get status", func(b Benchmarker) {
		app, err := GetDefaultTestApp(logger)
		Expect(err).NotTo(HaveOccurred())

		runtime := b.Time("runtime", func() {
			status, body := Get(app, "/status")
			Expect(status).To(Equal(http.StatusOK))
			Expect(body).NotTo(BeEmpty())
		})

		Expect(runtime.Seconds()).Should(BeNumerically("<", 0.1), "Status shouldn't take too long.")
	}, 200)

})
