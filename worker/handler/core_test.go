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
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"gopkg.in/redis.v4"

	"github.com/satori/go.uuid"
	"github.com/topfreegames/santiago/testing"
	. "github.com/topfreegames/santiago/worker/handler"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type mockClock struct {
	currentTime int64
}

func (m *mockClock) Now() int64 {
	return m.currentTime
}

//getTestRedisConn returns a connection to the test redis server
func getTestRedisConn() (*redis.Client, error) {
	redisPort := 57575
	redisPortEnv := os.Getenv("REDIS_PORT")
	if redisPortEnv != "" {
		res, err := strconv.ParseInt(redisPortEnv, 10, 32)
		if err != nil {
			return nil, err
		}
		redisPort = int(res)
	}
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("localhost:%d", redisPort),
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	return client, nil
}

func pushHook(client *redis.Client, queue, method, url string, payload map[string]interface{}, expiration ...time.Time) error {
	payloadJSON, _ := json.Marshal(payload)

	data := map[string]interface{}{
		"method":   method,
		"url":      url,
		"payload":  string(payloadJSON),
		"attempts": 0,
	}
	if len(expiration) > 0 {
		data["expires"] = expiration[0].Unix()
	}
	dataJSON, _ := json.Marshal(data)
	count, err := client.RPush(queue, dataJSON).Result()
	if err != nil {
		return err
	}
	Expect(count).To(BeEquivalentTo(1))
	return nil
}

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
	var logger *testing.MockLogger
	var testClient *redis.Client

	BeforeEach(func() {
		logger = testing.NewMockLogger()
		cli, err := getTestRedisConn()
		Expect(err).NotTo(HaveOccurred())
		testClient = cli
	})

	Describe("Worker instance", func() {
		It("should create a new instance", func() {
			worker := NewDefault("127.0.0.1", 57575, "", 0, logger)
			Expect(worker).NotTo(BeNil())
		})
	})

	Describe("Message Handling", func() {
		It("should send webhook", func() {
			responses := startRouteHandler([]string{"/webhook-sent"}, 52525)

			worker := NewDefault("127.0.0.1", 57575, "", 0, logger)
			msg := map[string]interface{}{
				"method":   "POST",
				"url":      "http://localhost:52525/webhook-sent",
				"payload":  "{\"qwe\":123}",
				"attempts": 0,
			}

			err := worker.Handle(msg)
			Expect(err).NotTo(HaveOccurred())

			Expect(*responses).To(HaveLen(1))

			resp := (*responses)[0]["payload"].(map[string]interface{})
			Expect(resp["qwe"]).To(BeEquivalentTo(123))
		})
	})

	Describe("Message subscription", func() {
		It("should subscribe to webhook", func() {
			queue := uuid.NewV4().String()
			responses := startRouteHandler([]string{"/webhook-subscribed"}, 52525)

			worker := New(
				queue,
				"127.0.0.1", 57575, "", 0,
				10, logger, true, 10*time.Millisecond,
				"", 10, &RealClock{},
			)

			err := pushHook(
				testClient, queue, "POST",
				"http://localhost:52525/webhook-subscribed",
				map[string]interface{}{
					"qwe": 123,
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = worker.ProcessSubscription()
			Expect(err).NotTo(HaveOccurred())

			time.Sleep(10 * time.Millisecond)

			Expect(*responses).To(HaveLen(1))
			resp := (*responses)[0]["payload"].(map[string]interface{})
			Expect(int(resp["qwe"].(float64))).To(Equal(123))
		})

		It("should requeue and process later if webhook down", func() {
			hookURL := "/webhook-retry"
			queue := uuid.NewV4().String()

			worker := New(
				queue,
				"127.0.0.1", 57575, "", 0,
				10, logger, true, time.Millisecond, "", 10, &RealClock{},
			)

			err := pushHook(
				testClient, queue, "POST",
				fmt.Sprintf("http://localhost:52525%s", hookURL),
				map[string]interface{}{
					"qwe": 123,
				},
			)
			Expect(err).NotTo(HaveOccurred())

			before := time.Now().UnixNano()
			err = worker.ProcessSubscription()
			Expect(err).To(HaveOccurred())

			res, err := testClient.LRange(queue, 0, 1).Result()
			Expect(err).NotTo(HaveOccurred())

			Expect(res).To(HaveLen(1))

			var hook map[string]interface{}
			err = json.Unmarshal([]byte(res[0]), &hook)
			Expect(err).NotTo(HaveOccurred())

			ms := 1000000
			Expect(hook["attempts"]).To(BeEquivalentTo(1))
			Expect(hook["backoff"]).To(BeNumerically(">", before+int64(9*ms)))
			Expect(hook["method"]).To(BeEquivalentTo("POST"))
			Expect(hook["url"]).To(BeEquivalentTo("http://localhost:52525/webhook-retry"))
			Expect(hook["payload"]).To(BeEquivalentTo("{\"qwe\":123}"))

			responses := startRouteHandler([]string{hookURL}, 52525)

			time.Sleep(50 * time.Millisecond)

			err = worker.ProcessSubscription()
			Expect(err).NotTo(HaveOccurred())

			Expect(*responses).To(HaveLen(1))

			resp := (*responses)[0]["payload"].(map[string]interface{})
			Expect(int(resp["qwe"].(float64))).To(Equal(123))
		})

		It("should requeue with exponential backoff", func() {
			hookURL := "/webhook-backoff"
			queue := uuid.NewV4().String()
			clock := &mockClock{}

			worker := New(
				queue,
				"127.0.0.1", 57575, "", 0,
				15, logger, true, time.Millisecond, "", 10, clock,
			)

			err := pushHook(
				testClient, queue, "POST",
				fmt.Sprintf("http://localhost:52525%s", hookURL),
				map[string]interface{}{
					"qwe": 123,
				},
			)
			Expect(err).NotTo(HaveOccurred())

			err = worker.ProcessSubscription()
			Expect(err).To(HaveOccurred())

			res, err := testClient.LRange(queue, 0, 1).Result()
			Expect(err).NotTo(HaveOccurred())

			var hook map[string]interface{}
			err = json.Unmarshal([]byte(res[0]), &hook)
			Expect(err).NotTo(HaveOccurred())

			Expect(hook["attempts"]).To(BeEquivalentTo(1))

			ms := int64(1000000)
			backoff := int64(hook["backoff"].(float64))
			Expect(backoff).To(BeNumerically(">", 10*ms))

			msg := fmt.Sprintf(
				"Expected time to be lesser than %v, but got %v",
				21*ms, backoff,
			)
			Expect(backoff).To(BeNumerically("<", int64(21*ms)), msg)

			for i := 2; i < 5; i++ {
				By(fmt.Sprintf("Backoff %d", i))
				power := int64(math.Pow(2, float64(i)))
				clock.currentTime = 10 * int64(power) * ms
				err = worker.ProcessSubscription()
				Expect(err).To(HaveOccurred())

				res, err = testClient.LRange(queue, 0, 1).Result()
				Expect(err).NotTo(HaveOccurred())

				err = json.Unmarshal([]byte(res[0]), &hook)
				Expect(err).NotTo(HaveOccurred())

				Expect(hook["backoff"]).To(BeNumerically(">", 10*ms*power))
			}
		})

		It("should subscribe to webhook if message has expiration but not expired", func() {
			queue := uuid.NewV4().String()
			responses := startRouteHandler([]string{"/webhook-subscribed-not-expired"}, 52525)

			worker := New(
				queue,
				"127.0.0.1", 57575, "", 0,
				10, logger, true, 10*time.Millisecond,
				"", 10, &RealClock{},
			)

			err := pushHook(
				testClient, queue, "POST",
				"http://localhost:52525/webhook-subscribed-not-expired",
				map[string]interface{}{
					"qwe": 123,
				},
				time.Now().Add(1*time.Hour),
			)
			Expect(err).NotTo(HaveOccurred())

			err = worker.ProcessSubscription()
			Expect(err).NotTo(HaveOccurred())

			time.Sleep(10 * time.Millisecond)

			Expect(*responses).To(HaveLen(1))
			resp := (*responses)[0]["payload"].(map[string]interface{})
			Expect(int(resp["qwe"].(float64))).To(Equal(123))
		})

		It("should not send webhook if message is expired", func() {
			queue := uuid.NewV4().String()
			responses := startRouteHandler([]string{"/webhook-expired"}, 52525)

			worker := New(
				queue,
				"127.0.0.1", 57575, "", 0,
				10, logger, true, 10*time.Millisecond,
				"", 10, &RealClock{},
			)

			err := pushHook(
				testClient, queue, "POST",
				"http://localhost:52525/webhook-expired",
				map[string]interface{}{
					"qwe": 123,
				},
				time.Now().Add(-1*time.Hour),
			)
			Expect(err).NotTo(HaveOccurred())

			err = worker.ProcessSubscription()
			Expect(err).NotTo(HaveOccurred())

			time.Sleep(10 * time.Millisecond)

			Expect(*responses).To(HaveLen(0))
		})
	})
})
