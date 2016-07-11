// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package worker_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/nsqio/go-nsq"
	. "github.com/topfreegames/santiago/worker/handler"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func startRouteHandler(routes []string, port int) *[]map[string]interface{} {
	responses := []map[string]interface{}{}

	go func() {
		handleFunc := func(w http.ResponseWriter, r *http.Request) {
			bs, err := ioutil.ReadAll(r.Body)
			if err != nil {
				responses = append(responses, map[string]interface{}{"reason": err})
				return
			}

			var payload map[string]interface{}
			json.Unmarshal(bs, &payload)

			response := map[string]interface{}{
				"payload":  payload,
				"request":  r,
				"response": w,
			}

			responses = append(responses, response)
		}
		for _, route := range routes {
			http.HandleFunc(route, handleFunc)
		}

		http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), nil)
	}()

	return &responses
}

var _ = Describe("Santiago Worker", func() {

	Describe("Worker instance", func() {
		It("should create a new instance", func() {
			worker := NewDefault("127.0.0.1", 7778)
			Expect(worker).NotTo(BeNil())
			Expect(worker.LookupHost).To(Equal("127.0.0.1"))
			Expect(worker.LookupPort).To(Equal(7778))
		})
	})

	Describe("Message Handling", func() {
		It("should send webhook", func() {
			responses := startRouteHandler([]string{"/webhook-sent"}, 52525)

			payload := map[string]interface{}{
				"method":  "POST",
				"url":     "http://localhost:52525/webhook-sent",
				"payload": map[string]interface{}{"qwe": 123},
			}
			payloadJSON, _ := json.Marshal(payload)
			worker := NewDefault("127.0.0.1", 7778)
			msg := &nsq.Message{
				Body:        payloadJSON,
				Timestamp:   time.Now().UnixNano(),
				Attempts:    0,
				NSQDAddress: "127.0.0.1:7778",
			}

			err := worker.Handle(msg)
			Expect(err).NotTo(HaveOccurred())

			Expect(*responses).To(HaveLen(1))

			resp := (*responses)[0]["payload"].(map[string]interface{})
			Expect(int(resp["qwe"].(float64))).To(Equal(123))
		})
	})
})
