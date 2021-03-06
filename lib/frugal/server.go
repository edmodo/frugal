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

// The frugal Go library provides additional helpers on top of the existing Thrift Go API, particularly for extending socket and multi-processing behavior.
package frugal

import (
	"errors"
	"io"
	"net"
	"sync/atomic"
	"time"

	"git.apache.org/thrift.git/lib/go/thrift"
)

// Information about a thrift request.
type Request struct {
	RequestId   int64
	SequenceId  int32
	MessageType thrift.TMessageType
	MethodName  string
	Input       thrift.TProtocol
	Output      thrift.TProtocol
}

// Callbacks for a request processor.
type Processor interface {
	// Process a client request.
	ProcessRequest(request *Request) error
}

// Callbacks that must be implemented for NewServer().
type ServerInterface interface {
	Processor

	// Must return input and output protocols for a client.
	GetProtocolsForClient(client Transport) (thrift.TProtocol, thrift.TProtocol)

	// Optionally log any errors encountered running the server.
	LogError(context string, err error)
}

// Options that can be passed to NewServer().
type ServerOptions struct {
	// Host and port string for listening.
	ListenAddr string

	// Timeout for client operations.
	ClientTimeout time.Duration

	// Whether or not to use framing.
	Framed bool
}

// This is a reimplementation of thrift.TSimpleServer. Eventually, we would
// like to remove dependence on the unnecessary factory abstraction layers,
// but for now we wrap the Thrift API.
type Server struct {
	// Passed in via NewServer().
	callbacks ServerInterface
	options   *ServerOptions

	// Current server state.
	addr     net.Addr
	listener net.Listener
	stopped  bool

	// The next request id to use.
	requestId int64
}

// Allocates a new thrift server. If the given host+port cannot be resolved,
// an error is returned.
func NewServer(callbacks ServerInterface, options *ServerOptions) (*Server, error) {
	addr, err := net.ResolveTCPAddr("tcp", options.ListenAddr)
	if err != nil {
		return nil, err
	}
	return &Server{
		callbacks: callbacks,
		options:   options,
		addr:      addr,
		listener:  nil,
		stopped:   false,
		requestId: int64(0),
	}, nil
}

// Thrift's protocol is not framed by default, so to differentiate between
// an idle connection and one that times out while reading a component of
// a header, we use a small wrapper type.
type ServerClientSocket struct {
	*Socket
	firstRead  bool
	oldTimeout time.Duration
}

func NewServerClientSocket(conn net.Conn, timeout time.Duration) *ServerClientSocket {
	return &ServerClientSocket{
		NewSocketFromConn(conn, timeout),
		false,
		0,
	}
}

func (this *ServerClientSocket) Reuse() error {
	// Flag the socket so that the next Read() has no timeout.
	this.firstRead = true
	this.oldTimeout = this.SetTimeout(0)
	return this.Socket.Reuse()
}

func (this *ServerClientSocket) Read(buf []byte) (int, error) {
	n, err := this.Socket.Read(buf)
	if this.firstRead {
		this.firstRead = false
		this.SetTimeout(this.oldTimeout)
	}
	return n, err
}

// Returns the address the server is listening or will listen on.
func (this *Server) Addr() net.Addr {
	if this.listener != nil {
		return this.listener.Addr()
	}
	return this.addr
}

// Begins servicing requests. Blocks until Stop() is called.
func (this *Server) Serve() error {
	if this.listener != nil {
		return errors.New("server is already listening")
	}

	var err error
	this.listener, err = net.Listen(this.addr.Network(), this.addr.String())
	if err != nil {
		return err
	}

	// Close the error on exit.
	defer func() {
		this.listener = nil
		this.stopped = false
	}()

	for !this.stopped {
		conn, err := this.listener.Accept()
		if err != nil {
			// If we're supposed to stop, just exit out.
			if this.stopped {
				break
			}

			// Otherwise log an error and go back to accepting connections.
			this.callbacks.LogError("accept", err)
			continue
		}

		go this.processRequest(conn)
	}

	return nil
}

func (this *Server) processRequest(conn net.Conn) {
	socket := NewServerClientSocket(conn, this.options.ClientTimeout)
	defer socket.Close()

	// Number of requests serviced off this connection.
	serviced := 0

	iprot, oprot := this.callbacks.GetProtocolsForClient(socket)
	for {
		name, msgType, sequenceId, err := iprot.ReadMessageBegin()
		if err != nil {
			// NB: Thrift coughs up some non-standard EOF instance, so we have to compare
			// the string instead.
			if err.Error() == io.EOF.Error() {
				return
			}

			netErr, ok := err.(net.Error)
			if ok && netErr.Timeout() && serviced >= 1 {
				// We already got data from this connection, and now it's idle. Just keep
				// polling for more data. Currently, we always set an infinite timeout
				// when calling Reuse(), but we may want occasional polling later.
				continue
			}

			// Otherwise, the error is fatal.
			this.callbacks.LogError("read-message-begin", err)
			return
		}

		// Allocate a request id. Note we do this with sync/atomic since request
		// processing happens in goroutines.
		requestId := atomic.AddInt64(&this.requestId, int64(1))

		err = this.callbacks.ProcessRequest(&Request{
			RequestId:   requestId,
			SequenceId:  sequenceId,
			MessageType: msgType,
			MethodName:  name,
			Input:       iprot,
			Output:      oprot,
		})
		if err != nil {
			this.callbacks.LogError("process-request", err)
			break
		}

		serviced++

		if err := socket.Reuse(); err != nil {
			this.callbacks.LogError("reuse-socket", err)
			return
		}
	}
}

// Interrupts Serve() causing the server to stop servicing requests.
func (this *Server) Stop() {
	if this.listener == nil || this.stopped {
		return
	}

	// Mark as stopped, then make the listener stop accepting conncetions.
	this.stopped = true
	this.listener.Close()
}
