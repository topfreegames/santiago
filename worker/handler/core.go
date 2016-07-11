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
	"github.com/valyala/fasthttp"
)

//Worker is a worker implementation that keeps processing webhooks
type Worker struct {
	LookupHost          string
	LookupPort          int
	LookupPollInterval  time.Duration
	MaxAttempts         int
	MaxMessagesInFlight int
	DefaultRequeueDelay time.Duration
}

//NewDefault returns a new worker with default options
func NewDefault(lookupHost string, lookupPort int) *Worker {
	return New(
		lookupHost, lookupPort, time.Duration(15)*time.Second,
		10, 150, time.Duration(15)*time.Second,
	)
}

//New creates a new worker instance
func New(
	lookupHost string, lookupPort int, lookupPollInterval time.Duration,
	maxAttempts int, maxMessagesInFlight int, defaultRequeueDelay time.Duration,
) *Worker {
	return &Worker{
		LookupHost:          lookupHost,
		LookupPort:          lookupPort,
		LookupPollInterval:  lookupPollInterval,
		MaxAttempts:         maxAttempts,
		MaxMessagesInFlight: maxMessagesInFlight,
		DefaultRequeueDelay: defaultRequeueDelay,
	}
}

//Handle a single message from NSQ
func (w *Worker) Handle(msg *nsq.Message) error {
	client := fasthttp.Client{
		Name: "santiago",
	}

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

	fmt.Println(result["url"])

	req := fasthttp.AcquireRequest()
	req.Header.SetMethod(result["method"].(string))
	req.SetRequestURI(result["url"].(string))
	req.AppendBody(payloadJSON)
	resp := fasthttp.AcquireResponse()

	timeout := time.Duration(5) * time.Second

	err = client.DoTimeout(req, resp, timeout)
	if err != nil {
		fmt.Printf("Could not request webhook %s: %s\n", result["url"], err.Error())
		return err
	}

	if resp.StatusCode() > 399 {
		fmt.Println("Error requesting webhook", resp.StatusCode())
		return err
	}

	fmt.Println(string(resp.Body()))
	return nil
}

//Subscribe to messages from NSQ
func (w *Worker) Subscribe() error {
	nsqLookupPath := fmt.Sprintf("%s:%d", w.LookupHost, w.LookupPort)
	config := nsq.NewConfig()
	config.LookupdPollInterval = w.LookupPollInterval
	config.MaxAttempts = uint16(w.MaxAttempts)
	config.MaxInFlight = w.MaxMessagesInFlight
	config.DefaultRequeueDelay = w.DefaultRequeueDelay

	q, err := nsq.NewConsumer("webhook", "main", config)
	if err != nil {
		log.Panic("Could not create consumer...")
		return err
	}

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
