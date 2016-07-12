// santiago - webhook dispatching service
// https://github.com/topfreegames/santiago
// Licensed under the MIT license:
// http://www.opensource.org/licenses/mit-license
// Copyright © 2016 Top Free Games <backend@tfgco.com>

package extensions

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kataras/fasthttp"
	"github.com/uber-go/zap"
)

//NSQLogger is a bridge to NSQ from Zap
type NSQLogger struct {
	Logger zap.Logger
}

//Output logs as Warnings
func (l *NSQLogger) Output(calldepth int, s string) error {
	l.Logger.Warn(s)
	return nil
}

//NSQd represents one NSQd node in our lookupd network
type NSQd struct {
	RemoteAddress    string
	HostName         string
	BroadcastAddress string
	TCPPort          int
	HTTPPort         int
	Version          string
	Topics           []string
}

//NSQLookup automatically queries the NSQLookup Server to retrieve available nodes
type NSQLookup struct {
	Host           string
	Port           int
	NSQdNodes      []NSQd
	UpdateInterval time.Duration
	Logger         zap.Logger
	Read           chan *getNodesOperation
}

//NewNSQLookup returns a new lookup updater
func NewNSQLookup(host string, port int, updateInterval time.Duration, logger zap.Logger) (*NSQLookup, error) {
	n := &NSQLookup{
		Host:           host,
		Port:           port,
		NSQdNodes:      []NSQd{},
		UpdateInterval: updateInterval,
		Logger:         logger,
	}

	n.start()

	return n, nil
}

func (n *NSQLookup) getNodes() []NSQd {
	return n.NSQdNodes
}

func (n *NSQLookup) updateNSQNodes() {
	client := fasthttp.Client{
		Name: "santiago",
	}

	url := fmt.Sprintf("http://%s:%d/nodes", n.Host, n.Port)
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod("GET")
	req.SetRequestURI(url)
	resp := fasthttp.AcquireResponse()

	timeout := time.Duration(5) * time.Second

	err := client.DoTimeout(req, resp, timeout)
	if err != nil {
		fmt.Printf("Could not update NSQLookup Nodes (%s): %s\n", url, err.Error())
		return
	}

	if resp.StatusCode() != 200 {
		fmt.Printf("Could not update NSQLookup Nodes (%s): %s\n", url, err.Error())
	}

	var obj map[string]interface{}
	err = json.Unmarshal(resp.Body(), &obj)
	if err != nil {
		fmt.Printf("Could not update NSQLookup Nodes (%s): %s\n", url, err.Error())
		return
	}

	data := obj["data"].(map[string]interface{})
	nodes := data["producers"].([]interface{})

	var nsqNodes []NSQd
	for _, nodeObj := range nodes {
		node := nodeObj.(map[string]interface{})
		n := NSQd{
			RemoteAddress:    node["remote_address"].(string),
			HostName:         node["hostname"].(string),
			BroadcastAddress: node["broadcast_address"].(string),
			TCPPort:          int(node["tcp_port"].(float64)),
			HTTPPort:         int(node["http_port"].(float64)),
			Version:          node["version"].(string),
			Topics:           []string{},
		}

		for _, topicObj := range node["topics"].([]interface{}) {
			topic := topicObj.(string)
			n.Topics = append(n.Topics, topic)
		}
		nsqNodes = append(nsqNodes, n)
	}
	n.NSQdNodes = nsqNodes
}

func (n *NSQLookup) start() {
	n.Read = make(chan *getNodesOperation)

	go func() {
		ticker := time.NewTicker(n.UpdateInterval)
		for {
			select {
			case <-ticker.C:
				n.updateNSQNodes()
			case read := <-n.Read:
				read.resp <- n.getNodes()
			}
		}
	}()
}

//Get returns the most current list of NSQd nodes
func (n *NSQLookup) Get() []NSQd {
	read := &getNodesOperation{
		resp: make(chan []NSQd),
	}
	n.Read <- read
	return <-read.resp
}

type getNodesOperation struct {
	resp chan []NSQd
}
