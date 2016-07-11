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
	NSQLookupDHost string
	NSQLookupDPort int
}

func (w *Worker) Handle(msg *nsq.Message) error {
	fmt.Println(string(msg.Body))
	return nil
}

func (w *Worker) Subscribe() error {
	nsqLookupPath := fmt.Sprintf("%s:%d", w.NSQLookupDHost, w.NSQLookupDPort)
	config := nsq.NewConfig()
	config.LookupdPollInterval = time.Duration(15) * time.Second
	config.MaxAttempts = 10
	config.MaxInFlight = 150
	config.DefaultRequeueDelay = time.Duration(15) * time.Second

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
