// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package worker

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"gopkg.in/redis.v4"

	"github.com/getsentry/raven-go"
	"github.com/topfreegames/santiago/log"
	"github.com/uber-go/zap"
	"github.com/valyala/fasthttp"
)

//Clock represents a clock to be used by the worker
type Clock interface {
	Now() int64
}

//RealClock uses the machine clock to return time
type RealClock struct{}

//Now returns time.Now()
func (r *RealClock) Now() int64 {
	return time.Now().UnixNano()
}

//Worker is a worker implementation that keeps processing webhooks
type Worker struct {
	Debug             bool
	Queue             string
	Logger            zap.Logger
	MaxAttempts       int
	Client            *redis.Client
	BlockTimeout      time.Duration
	SentryURL         string
	BackoffIntervalMs int64
	Clock             Clock
}

//NewDefault returns a new worker with default options
func NewDefault(redisHost string, redisPort int, redisPassword string, redisDB int, logger zap.Logger) *Worker {
	return New(
		"webhook",
		redisHost, redisPort, redisPassword, redisDB,
		15, logger, false, 5*time.Second, "", 5000,
		&RealClock{},
	)
}

//New creates a new worker instance
func New(
	queue string, redisHost string, redisPort int, redisPassword string, redisDB int,
	maxAttempts int, logger zap.Logger, debug bool, blockTimeout time.Duration,
	sentryURL string, backoffIntervalMs int64, clock Clock,
) *Worker {
	w := &Worker{
		Debug:             debug,
		Logger:            logger,
		Queue:             queue,
		MaxAttempts:       maxAttempts,
		BlockTimeout:      blockTimeout,
		SentryURL:         sentryURL,
		BackoffIntervalMs: backoffIntervalMs,
		Clock:             clock,
	}
	err := w.connectToRedis(redisHost, redisPort, redisPassword, redisDB)
	if err != nil {
		logger.Panic("Could not start worker due to error connecting to Redis...", zap.Error(err))
	}
	w.connectRaven()
	return w
}

func (w *Worker) connectRaven() {
	raven.SetDSN(w.SentryURL)
}

func (w *Worker) connectToRedis(redisHost string, redisPort int, redisPassword string, redisDB int) error {
	l := w.Logger.With(
		zap.String("operation", "connectToRedis"),
		zap.String("redisHost", redisHost),
		zap.Int("redisPort", redisPort),
		zap.Int("redisDB", redisDB),
		zap.Bool("hasPassword", redisPassword != ""),
	)

	log.D(l, "Connecting to Redis...")
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", redisHost, redisPort),
		Password: redisPassword,
		DB:       redisDB,
	})

	start := time.Now()
	_, err := client.Ping().Result()
	if err != nil {
		l.Error("Could not connect to redis.", zap.Error(err))
		return err
	}
	log.I(l, "Connected to Redis successfully.", func(cm log.CM) {
		cm.Write(zap.Duration("connection", time.Now().Sub(start)))
	})

	w.Client = client
	return nil
}

//DoRequest to some webhook endpoint
func (w *Worker) DoRequest(method, url, payload string) (int, string, error) {
	l := w.Logger.With(
		zap.String("operation", "DoRequest"),
		zap.String("method", method),
		zap.String("url", url),
		zap.String("payload", payload),
	)

	client := fasthttp.Client{
		Name: "santiago",
	}

	start := time.Now()
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod(method)
	req.SetRequestURI(url)
	if method != "GET" && payload != "" && payload != "NULL" {
		req.AppendBody([]byte(payload))
	}
	resp := fasthttp.AcquireResponse()

	timeout := time.Duration(5) * time.Second

	err := client.DoTimeout(req, resp, timeout)
	if err != nil {
		return 0, "", err
	}

	status := resp.StatusCode()
	body := string(resp.Body())
	log.I(l,
		"Request hook finished without error.",
		func(cm log.CM) {
			cm.Write(
				zap.Int("statusCode", status),
				zap.String("body", body),
				zap.Duration("requestDuration", time.Now().Sub(start)),
			)
		},
	)

	return status, body, nil
}

func (w *Worker) requeueMessage(method, url, payload string, attempts int, incrementAttempts bool) error {
	l := w.Logger.With(
		zap.String("operation", "requeueMessage"),
		zap.String("method", method),
		zap.String("url", url),
	)

	if attempts > w.MaxAttempts {
		msg := "Max attempts reached for message. Message will be discarded."
		l.Warn(msg)
		err := fmt.Errorf(msg)

		tags := map[string]string{
			"method":  method,
			"url":     url,
			"payload": payload,
		}
		raven.CaptureError(err, tags)

		return nil
	}

	if incrementAttempts {
		attempts++
	}

	millisecond := int64(1000000)
	power := int64(math.Pow(2, float64(attempts)))
	backoffTimestamp := w.Clock.Now() + (int64(w.BackoffIntervalMs) * power * millisecond)

	data := map[string]interface{}{
		"method":   method,
		"url":      url,
		"payload":  payload,
		"attempts": attempts,
		"backoff":  backoffTimestamp,
	}
	dataJSON, _ := json.Marshal(data)

	start := time.Now()

	if incrementAttempts {
		log.D(l, "Re-enqueueing hook...")
	} else {
		log.D(l, "Ignoring hook...")
	}
	_, err := w.Client.RPush(w.Queue, dataJSON).Result()
	if err != nil {
		if incrementAttempts {
			l.Error("Re-enqueueing hook failed.", zap.Error(err))
		} else {
			l.Error("Ignoring hook failed.", zap.Error(err))
		}
		return err
	}
	if incrementAttempts {
		log.I(l, "Hook re-enqueue succeeded.", func(cm log.CM) {
			cm.Write(zap.Duration("ReEnqueueDuration", time.Now().Sub(start)))
		})
	} else {
		log.D(l, "Hook ignore succeeded.", func(cm log.CM) {
			cm.Write(zap.Duration("IgnoreDuration", time.Now().Sub(start)))
		})
	}

	return nil
}

//Handle a single message from Queue
func (w *Worker) Handle(msg map[string]interface{}) error {
	l := w.Logger.With(
		zap.String("operation", "Handle"),
	)

	if msg["method"] == nil || msg["url"] == nil {
		l.Warn("Web Hook must contain both method and URL to be processed.")
		return fmt.Errorf("Web Hook must contain both method(%s) and URL(%s) to be processed.", msg["url"], msg["method"])
	}

	l = l.With(
		zap.String("method", msg["method"].(string)),
		zap.String("url", msg["url"].(string)),
	)

	method := msg["method"].(string)
	url := msg["url"].(string)

	if att, ok := msg["expires"]; ok {
		dt := int64(att.(float64))
		expiration := time.Unix(dt, 0)

		l = l.With(zap.Time("expires", expiration))

		if expiration.Before(time.Now()) {
			l.Warn("Failed to send message since it's expired.")
			return nil
		}
	}

	attempts := 0
	if att, ok := msg["attempts"]; ok {
		switch att.(type) {
		case float64:
			attempts = int(att.(float64))
		case int:
			attempts = att.(int)
		default:
			attr, _ := strconv.ParseInt(fmt.Sprintf("%v", att), 10, 32)
			attempts = int(attr)
		}
	}
	payload := ""
	if msg["payload"] != nil {
		payload = msg["payload"].(string)
	}

	timestamp := w.Clock.Now()
	if msg["backoff"] != nil {
		if int64(msg["backoff"].(float64)) > timestamp {
			bkl := l.With(
				zap.Int("attempts", attempts),
				zap.Int64("backoff", int64(msg["backoff"].(float64))),
				zap.Int64("timestamp", timestamp),
			)
			log.D(bkl, "Re-enqueueing message with backoff.")
			err := w.requeueMessage(method, url, payload, attempts, false)
			if err != nil {
				bkl.Error("Could not re-enqueue hook with backoff.", zap.Error(err))
				return err
			}
			log.D(bkl, "Message re-enqueued successfully.")
			return nil
		}
	}

	log.D(l, "Performing request...", func(cm log.CM) {
		cm.Write(zap.String("payload", payload), zap.Int("attempts", attempts))
	})

	go func() {
		status, _, err := w.DoRequest(method, url, payload)
		if err != nil {
			l.Error("Could not process hook, trying again later.", zap.Error(err), zap.Int("attempts", attempts))
			err2 := w.requeueMessage(method, url, payload, attempts, true)
			if err2 != nil {
				l.Error("Could not re-enqueue hook.", zap.Error(err2))
			}
			return
		}
		if status > 399 {
			err := fmt.Errorf("Error requesting webhook. Status code: %d", status)
			l.Error(
				"Could not process hook, trying again later.",
				zap.Int("statusCode", status),
				zap.Error(err),
				zap.Int("attempts", attempts),
			)
			err2 := w.requeueMessage(method, url, payload, attempts, true)
			if err2 != nil {
				l.Error("Could not re-enqueue hook.", zap.Error(err2))
			}
			return
		}

		log.I(l, "Webhook processed successfully.")
	}()

	return nil
}

//ProcessSubscription to messages from Queue
func (w *Worker) ProcessSubscription() error {
	l := w.Logger.With(
		zap.String("operation", "Subscribe"),
		zap.String("queue", w.Queue),
		zap.Int("maxAttempts", w.MaxAttempts),
	)

	res, err := w.Client.LPop(w.Queue).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			log.D(l, "No hooks to be processed.")
			return nil
		}
		l.Error("Worker failed to consume message from queue.", zap.Error(err))
		return err
	}

	var msg map[string]interface{}
	err = json.Unmarshal([]byte(res), &msg)
	if err != nil {
		l.Error("Worker failed to deserialize message from queue.", zap.Error(err))
		return err
	}

	err = w.Handle(msg)
	if err != nil {
		return err
	}

	log.D(l, "Worker consumed message successfully.")
	return nil
}

//Start a new worker with the given params
func (w *Worker) Start() {
	l := w.Logger.With(
		zap.String("operation", "Subscribe"),
		zap.String("queue", w.Queue),
		zap.Int("maxAttempts", w.MaxAttempts),
	)

	for {
		log.D(l, "Subscribing to next message...")

		for i := 0; i < 50; i++ {
			raven.CapturePanic(func() {
				err := w.ProcessSubscription()
				if err != nil {
					l.Warn("Failed to retrieve messages from queue.", zap.Error(err))
					tags := map[string]string{
						"queue":       w.Queue,
						"maxAttempts": string(w.MaxAttempts),
					}
					raven.CaptureError(err, tags)
				}
				time.Sleep(50 * time.Millisecond)
			}, nil)
		}
	}
}
