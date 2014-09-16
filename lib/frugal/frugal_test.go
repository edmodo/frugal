// vim: set ts=4 sw=4 tw=99 noet: 
//
// Copyright 2014, Edmodo, Inc. 
// 
// Licensed under the Apache License, Version 2.0 (the "License"); you may not use this work except in compliance with the License.
// You may obtain a copy of the License in the LICENSE file, or at:
// 
// http://www.apache.org/licenses/LICENSE-2.0
// 
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" 
// BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language 
// governing permissions and limitations under the License. 

package frugal

import (
	"fmt"
	"log"

	"git.apache.org/thrift.git/lib/go/thrift"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestInit(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Frugal library testing")
}

type TestServer struct {
	Port    string
	Socket  *thrift.TServerSocket
	notify  chan bool
	stopped bool
}

type TestHandler func(transport thrift.TTransport)

// A simple server that just replies with whatever data it receives.
func NewTestServer() *TestServer {
	return &TestServer{
		Port:    "45321",
		Socket:  nil,
		notify:  make(chan bool),
		stopped: false,
	}
}

func (this *TestServer) Start(callback TestHandler) error {
	if this.Socket != nil {
		return fmt.Errorf("server already started")
	}

	socket, err := thrift.NewTServerSocket(fmt.Sprintf("127.0.0.1:%s", this.Port))
	if err != nil {
		return err
	}

	if err := socket.Open(); err != nil {
		socket.Close()
		return err
	}

	this.Socket = socket
	this.stopped = false

	go (func() {
		for !this.stopped {
			client, err := this.Socket.Accept()
			if err != nil && !this.stopped {
				log.Println("Accept err: ", err)
			}
			if client != nil {
				go callback(client)
			}
		}
		this.notify <- true
	})()

	return nil
}

func (this *TestServer) Stop() {
	if this.Socket == nil {
		return
	}

	this.stopped = true
	this.Socket.Interrupt()
	this.Socket.Close()

	// Wait for the goroutine to end.
	<-this.notify

	this.Socket = nil
}

type TestClientFactory struct {
}

func NewTestClientFactory() *TestClientFactory {
	return &TestClientFactory{}
}

func (this *TestClientFactory) Connect() (*Connection, error) {
	transport, err := NewResumeableSocket("127.0.0.1:45321", 0)
	if err != nil {
		return nil, err
	}
	return NewConnectionFromFactory(transport, thrift.NewTBinaryProtocolFactoryDefault()), nil
}
