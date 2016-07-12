// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package api

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/topfreegames/santiago/testing"
)

var _ = Describe("Healthcheck Handler", func() {
	var logger *MockLogger

	BeforeEach(func() {
		logger = NewMockLogger()
	})

	It("Should respond with default WORKING string", func() {
		a, err := GetDefaultTestApp(logger)
		Expect(err).NotTo(HaveOccurred())
		res := Get(a, "/healthcheck")

		Expect(res.Raw().StatusCode).To(Equal(http.StatusOK))
		Expect(res.Body().Raw()).To(Equal("WORKING"))
	})

	It("Should respond with customized WORKING string", func() {
		a, err := GetDefaultTestApp(logger)
		Expect(err).NotTo(HaveOccurred())

		a.Config.Set("api.workingText", "OTHERWORKING")
		res := Get(a, "/healthcheck")

		Expect(res.Raw().StatusCode).To(Equal(http.StatusOK))
		Expect(res.Body().Raw()).To(Equal("OTHERWORKING"))
	})
})
