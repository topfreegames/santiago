// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package worker

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nsqio/go-nsq"
	"github.com/uber-go/zap"
	"github.com/valyala/fasthttp"
)

type NSQLogger struct {
	logger zap.Logger
}

func (l *NSQLogger) Output(calldepth int, s string) error {
	l.logger.Warn(s)
	return nil
}

//Worker is a worker implementation that keeps processing webhooks
type Worker struct {
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
		logger,
	)
}

//New creates a new worker instance
func New(
	topic string, lookupHost string, lookupPort int, lookupPollInterval time.Duration,
	maxAttempts int, maxMessagesInFlight int, defaultRequeueDelay time.Duration,
	logger zap.Logger,
) *Worker {
	return &Worker{
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

func (w *Worker) DoRequest(method, url, payload string) (int, string, error) {
	client := fasthttp.Client{
		Name: "santiago",
	}

	req := fasthttp.AcquireRequest()
	req.Header.SetMethod(method)
	req.SetRequestURI(url)
	req.AppendBody([]byte(payload))
	resp := fasthttp.AcquireResponse()

	timeout := time.Duration(5) * time.Second

	err := client.DoTimeout(req, resp, timeout)
	if err != nil {
		fmt.Printf("Could not request webhook %s: %s\n", url, err.Error())
		return 0, "", err
	}

	return resp.StatusCode(), string(resp.Body()), nil
}

//Handle a single message from NSQ
func (w *Worker) Handle(msg *nsq.Message) error {
	var result map[string]interface{}
	err := json.Unmarshal(msg.Body, &result)
	if err != nil {
		fmt.Println("Could not process body", err)
		return err
	}

	payloadJSON, err := json.Marshal(result["payload"])
	if err != nil {
		fmt.Println("Could not process payload", err)
		return err
	}

	status, _, err := w.DoRequest(result["method"].(string), result["url"].(string), string(payloadJSON))
	if status > 399 {
		fmt.Println("Error requesting webhook", status)
		return err
	}

	return nil
}

//Subscribe to messages from NSQ
func (w *Worker) Subscribe() error {
	l := w.Logger.With(
		zap.String("operation", "Subscribe"),
	)
	nsqLookupPath := fmt.Sprintf("%s:%d", w.LookupHost, w.LookupPort)
	config := nsq.NewConfig()
	config.LookupdPollInterval = w.LookupPollInterval
	config.MaxAttempts = uint16(w.MaxAttempts)
	config.MaxInFlight = w.MaxMessagesInFlight
	config.DefaultRequeueDelay = w.DefaultRequeueDelay

	q, err := nsq.NewConsumer(w.Topic, "main", config)
	if err != nil {
		log.Panic("Could not create consumer...")
		return err
	}
	q.SetLogger(&NSQLogger{logger: l}, nsq.LogLevelWarning)

	q.AddHandler(nsq.HandlerFunc(w.Handle))

	err = q.ConnectToNSQLookupd(nsqLookupPath)
	if err != nil {
		log.Panic("Could not connect.")
		return err
	}

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
