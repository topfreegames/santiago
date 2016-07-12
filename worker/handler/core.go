// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package worker

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nsqio/go-nsq"
	"github.com/topfreegames/santiago/extensions"
	"github.com/uber-go/zap"
	"github.com/valyala/fasthttp"
)

//Worker is a worker implementation that keeps processing webhooks
type Worker struct {
	Debug               bool
	Topic               string
	Logger              zap.Logger
	LookupHost          string
	LookupPort          int
	LookupPollInterval  time.Duration
	MaxAttempts         int
	MaxMessagesInFlight int
	DefaultRequeueDelay time.Duration
}

//NewDefault returns a new worker with default options
func NewDefault(lookupHost string, lookupPort int, logger zap.Logger) *Worker {
	return New(
		"webhook",
		lookupHost, lookupPort, time.Duration(15)*time.Second,
		10, 150, time.Duration(15)*time.Second,
		logger, false,
	)
}

//New creates a new worker instance
func New(
	topic string, lookupHost string, lookupPort int, lookupPollInterval time.Duration,
	maxAttempts int, maxMessagesInFlight int, defaultRequeueDelay time.Duration,
	logger zap.Logger, debug bool,
) *Worker {
	return &Worker{
		Debug:               debug,
		Logger:              logger,
		Topic:               topic,
		LookupHost:          lookupHost,
		LookupPort:          lookupPort,
		LookupPollInterval:  lookupPollInterval,
		MaxAttempts:         maxAttempts,
		MaxMessagesInFlight: maxMessagesInFlight,
		DefaultRequeueDelay: defaultRequeueDelay,
	}
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

func (w *Worker) requeueMessage(method, url string, msg *nsq.Message) {
	l := w.Logger.With(
		zap.String("operation", "requeueMessage"),
		zap.String("method", method),
		zap.String("url", url),
	)

	if int(msg.Attempts) > w.MaxAttempts {
		l.Warn("Max attempts reached for message. Message will be discarded.")
		return
	}

	l.Debug("Requeueing message.", zap.Int("attempt", int(msg.Attempts)))
	msg.RequeueWithoutBackoff(time.Duration(-1))
}

//Handle a single message from NSQ
func (w *Worker) Handle(msg *nsq.Message) error {
	l := w.Logger.With(
		zap.String("operation", "Handle"),
		zap.String("message", string(msg.Body)),
	)

	l.Debug("Unmarshaling message body...")
	var result map[string]interface{}
	err := json.Unmarshal(msg.Body, &result)
	if err != nil {
		l.Error("Message body could not be processed.", zap.Error(err))
		return err
	}
	l.Debug("Message body unmarshaled successfully.")

	l.Debug("Unmarshaling payload...")
	payloadJSON, err := json.Marshal(result["payload"])
	if err != nil {
		l.Error("Message payload could not be processed.", zap.Object("payload", result["payload"]), zap.Error(err))
		fmt.Println("Could not process payload", err)
		return nil
	}
	l.Debug("Payload unmarshaled successfully.")

	method := result["method"].(string)
	url := result["url"].(string)
	l.Debug(
		"Performing request...",
		zap.String("method", method),
		zap.String("url", url),
		zap.String("payload", string(payloadJSON)),
	)
	status, _, err := w.DoRequest(method, url, string(payloadJSON))
	if err != nil {
		l.Error("Could not process hook, trying again later.", zap.Error(err), zap.Int("attempts", int(msg.Attempts)))
		w.requeueMessage(method, url, msg)
		return err
	}
	if status > 399 {
		err := fmt.Errorf("Error requesting webhook. Status code: %d", status)
		l.Error(
			"Could not process hook, trying again later.",
			zap.Int("statusCode", status),
			zap.Error(err),
			zap.Int("attempts", int(msg.Attempts)),
		)
		w.requeueMessage(method, url, msg)
		return err
	}

	l.Info("Webhook processed successfully.")
	return nil
}

//Subscribe to messages from NSQ
func (w *Worker) Subscribe() error {
	nsqLookupPath := fmt.Sprintf("%s:%d", w.LookupHost, w.LookupPort)

	l := w.Logger.With(
		zap.String("operation", "Subscribe"),
		zap.String("nsqLookup", nsqLookupPath),
		zap.String("topic", w.Topic),
		zap.String("channel", "main"),
		zap.Duration("lookupPollInterval", w.LookupPollInterval),
		zap.Int("maxAttempts", w.MaxAttempts),
		zap.Int("maxInFlight", w.MaxMessagesInFlight),
		zap.Duration("defaultRequeueDelay", w.DefaultRequeueDelay),
	)

	config := nsq.NewConfig()
	config.LookupdPollInterval = w.LookupPollInterval
	config.MaxAttempts = uint16(w.MaxAttempts)
	config.MaxInFlight = w.MaxMessagesInFlight
	config.DefaultRequeueDelay = w.DefaultRequeueDelay

	l.Debug("Starting consumer...")
	q, err := nsq.NewConsumer(w.Topic, "main", config)
	if err != nil {
		l.Error("Consumer failed to start.", zap.Error(err))
		return err
	}

	logLevel := nsq.LogLevelWarning
	//if w.Debug {
	//logLevel = nsq.LogLevelDebug
	//}

	q.SetLogger(&extensions.NSQLogger{Logger: l}, logLevel)

	q.AddHandler(nsq.HandlerFunc(w.Handle))

	err = q.ConnectToNSQLookupd(nsqLookupPath)
	if err != nil {
		l.Error("Consumer failed to connect to NSQLookupD.", zap.Error(err))
		return err
	}

	l.Info("Consumer started successfully.")
	return nil
}

//Start a new worker
func (w *Worker) Start() error {
	if err := w.Subscribe(); err != nil {
		return err
	}
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		done <- true
	}()
	<-done

	return nil
}
