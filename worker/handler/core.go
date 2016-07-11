// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package worker

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nsqio/go-nsq"
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
	fmt.Println(string(msg.Body))
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
