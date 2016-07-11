// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package worker_test

import (
	. "github.com/topfreegames/santiago/worker/handler"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Santiago Worker", func() {

	Describe("Worker instance", func() {
		It("should create a new instance", func() {
			worker := NewDefault("127.0.0.1", 7778)
			Expect(worker).NotTo(BeNil())
			Expect(worker.LookupHost).To(Equal("127.0.0.1"))
			Expect(worker.LookupPort).To(Equal(7778))
		})
	})
})
