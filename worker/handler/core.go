// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package worker

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gopkg.in/redis.v4"

	"github.com/uber-go/zap"
	"github.com/valyala/fasthttp"
)

//Worker is a worker implementation that keeps processing webhooks
type Worker struct {
	Debug        bool
	Queue        string
	Logger       zap.Logger
	MaxAttempts  int
	Client       *redis.Client
	BlockTimeout time.Duration
}

//NewDefault returns a new worker with default options
func NewDefault(redisHost string, redisPort int, redisPassword string, redisDB int, logger zap.Logger) *Worker {
	return New(
		"webhook",
		redisHost, redisPort, redisPassword, redisDB,
		10, logger, false, 5*time.Second,
	)
}

//New creates a new worker instance
func New(
	queue string, redisHost string, redisPort int, redisPassword string, redisDB int,
	maxAttempts int, logger zap.Logger, debug bool, blockTimeout time.Duration,
) *Worker {
	w := &Worker{
		Debug:        debug,
		Logger:       logger,
		Queue:        queue,
		MaxAttempts:  maxAttempts,
		BlockTimeout: blockTimeout,
	}
	err := w.connectToRedis(redisHost, redisPort, redisPassword, redisDB)
	if err != nil {
		logger.Panic("Could not start worker due to error connecting to Redis...", zap.Error(err))
	}
	return w
}

func (w *Worker) connectToRedis(redisHost string, redisPort int, redisPassword string, redisDB int) error {
	l := w.Logger.With(
		zap.String("operation", "connectToRedis"),
		zap.String("redisHost", redisHost),
		zap.Int("redisPort", redisPort),
		zap.Int("redisDB", redisDB),
	)

	l.Debug("Connecting to Redis...")
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
	l.Info("Connected to Redis successfully.", zap.Duration("connection", time.Now().Sub(start)))

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
	l.Info(
		"Request hook finished without error.",
		zap.Int("statusCode", status),
		zap.String("body", body),
		zap.Duration("requestDuration", time.Now().Sub(start)),
	)

	return status, body, nil
}

func (w *Worker) requeueMessage(method, url, payload string, attempts int) error {
	l := w.Logger.With(
		zap.String("operation", "requeueMessage"),
		zap.String("method", method),
		zap.String("url", url),
	)

	if attempts > w.MaxAttempts {
		l.Warn("Max attempts reached for message. Message will be discarded.")
		return nil
	}

	attempts++

	l.Debug("Requeueing message.", zap.Int("attempt", attempts), zap.String("payload", payload))

	data := map[string]interface{}{
		"method":   method,
		"url":      url,
		"payload":  payload,
		"attempts": attempts,
	}
	dataJSON, _ := json.Marshal(data)

	start := time.Now()

	l.Debug("Re-enqueueing hook...")
	_, err := w.Client.RPush(w.Queue, dataJSON).Result()
	if err != nil {
		l.Error("Re-enqueueing hook failed.", zap.Error(err))
		return err
	}
	l.Info("Hook re-enqueue succeeded.", zap.Duration("ReEnqueueDuration", time.Now().Sub(start)))

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

	l.Debug(
		"Performing request...",
		zap.String("payload", payload),
		zap.Int("attempts", attempts),
	)
	status, _, err := w.DoRequest(method, url, payload)
	if err != nil {
		l.Error("Could not process hook, trying again later.", zap.Error(err), zap.Int("attempts", attempts))
		err2 := w.requeueMessage(method, url, payload, attempts)
		if err2 != nil {
			l.Error("Could not re-enqueue hook.", zap.Error(err2))
		}
		return err
	}
	if status > 399 {
		err := fmt.Errorf("Error requesting webhook. Status code: %d", status)
		l.Error(
			"Could not process hook, trying again later.",
			zap.Int("statusCode", status),
			zap.Error(err),
			zap.Int("attempts", attempts),
		)
		err2 := w.requeueMessage(method, url, payload, attempts)
		if err2 != nil {
			l.Error("Could not re-enqueue hook.", zap.Error(err2))
		}
		return err
	}

	l.Info("Webhook processed successfully.")
	return nil
}

//ProcessSubscription to messages from Queue
func (w *Worker) ProcessSubscription() error {
	l := w.Logger.With(
		zap.String("operation", "Subscribe"),
		zap.String("queue", w.Queue),
		zap.Int("maxAttempts", w.MaxAttempts),
	)

	res, err := w.Client.BLPop(w.BlockTimeout, w.Queue).Result()
	if err != nil {
		if strings.HasSuffix("i/o timeout", err.Error()) {
			return nil
		}
		l.Error("Worker failed to consume message from queue.", zap.Error(err))
		return err
	}

	var msg map[string]interface{}
	err = json.Unmarshal([]byte(res[1]), &msg)
	if err != nil {
		l.Error("Worker failed to deserialize message from queue.", zap.Error(err))
		return err
	}

	err = w.Handle(msg)
	if err != nil {
		return err
	}

	l.Debug("Worker consumed message successfully.")
	return nil
}

//Start a new worker
func (w *Worker) Start() {
	l := w.Logger.With(
		zap.String("operation", "Subscribe"),
		zap.String("queue", w.Queue),
		zap.Int("maxAttempts", w.MaxAttempts),
	)

	for {
		l.Debug("Subscribing to next message...")
		err := w.ProcessSubscription()
		if err != nil {
			l.Warn("Failed to retrieve messages from queue.", zap.Error(err))
		}
	}
}
