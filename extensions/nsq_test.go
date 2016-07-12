// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package extensions_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/topfreegames/santiago/extensions"
	. "github.com/topfreegames/santiago/testing"
)

var _ = Describe("NSQ Extension", func() {
	var logger *MockLogger

	BeforeEach(func() {
		logger = NewMockLogger()
	})

	Describe("NSQLookup", func() {
		It("should load NSQ Nodes", func() {
			lookup, err := extensions.NewNSQLookup("0.0.0.0", 7778, 10*time.Millisecond, logger)
			Expect(err).NotTo(HaveOccurred())

			time.Sleep(20 * time.Millisecond)

			nodes := lookup.Get()
			Expect(nodes).To(HaveLen(1))

			Expect(nodes[0].RemoteAddress).To(ContainSubstring("127.0.0.1"))
			Expect(nodes[0].BroadcastAddress).To(Equal("localhost"))
			Expect(nodes[0].TCPPort).To(Equal(7779))
			Expect(nodes[0].HTTPPort).To(Equal(7780))
		})
	})
})
