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
	"github.com/satori/go.uuid"
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

	Describe("Message subscription", func() {
		It("should subscribe to webhook", func() {
			queue := uuid.NewV4().String()
			responses := startRouteHandler([]string{"/webhook-subscribed"}, 52525)

			payload := map[string]interface{}{
				"method":  "POST",
				"url":     "http://localhost:52525/webhook-subscribed",
				"payload": map[string]interface{}{"qwe": 123},
			}
			payloadJSON, _ := json.Marshal(payload)

			worker := New(
				queue,
				"127.0.0.1", 7778, time.Duration(15)*time.Millisecond,
				10, 150, time.Duration(15)*time.Millisecond,
			)

			err := worker.Subscribe()
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(50 * time.Millisecond)

			status, _, err := worker.DoRequest("POST", fmt.Sprintf("http://127.0.0.1:7780/put?topic=%s", queue), string(payloadJSON))
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(Equal(200))
			time.Sleep(300 * time.Millisecond)

			Expect(*responses).To(HaveLen(1))

			resp := (*responses)[0]["payload"].(map[string]interface{})
			Expect(int(resp["qwe"].(float64))).To(Equal(123))
		})

	})
})
