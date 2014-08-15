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

// Callbacks that must be implemented for NewThriftServer().
type ServerInterface interface {
	// Must return input and output protocols for a client.
	GetProtocolsForClient(client Transport) (thrift.TProtocol, thrift.TProtocol)

	// Process a client request.
	ProcessRequest(request *Request) error

	// Optionally log any errors encountered running the server.
	LogError(context string, err error)
}

// Options that can be passed to NewThriftServer().
type ServerOptions struct {
	// Host and port string for listening.
	ListenAddr string

	// Timeout for client operations.
	ClientTimeout time.Duration
}

// This is a reimplementation of thrift.TSimpleServer. Eventually, we would
// like to remove dependence on the unnecessary factory abstraction layers,
// but for now we wrap the Thrift API.
type ThriftServer struct {
	// Passed in via NewThriftServer().
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
func NewThriftServer(callbacks ServerInterface, options *ServerOptions) (*ThriftServer, error) {
	addr, err := net.ResolveTCPAddr("tcp", options.ListenAddr)
	if err != nil {
		return nil, err
	}
	return &ThriftServer{
		callbacks: callbacks,
		options:   options,
		addr:      addr,
		listener:  nil,
		stopped:   false,
		requestId: int64(0),
	}, nil
}

func (this *ThriftServer) Serve() error {
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

func (this *ThriftServer) processRequest(conn net.Conn) {
	socket := NewSocketFromConn(conn, this.options.ClientTimeout)
	defer socket.Close()

	iprot, oprot := this.callbacks.GetProtocolsForClient(socket)
	for {
		name, msgType, sequenceId, err := iprot.ReadMessageBegin()
		if err != nil {
			if err != io.EOF {
				// Log the error if it's not a clean exit.
				this.callbacks.LogError("read-message-begin", err)
			}
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
	}
}

func (this *ThriftServer) Stop() {
	if this.listener == nil || this.stopped {
		return
	}

	// Mark as stopped, then make the listener stop accepting conncetions.
	this.stopped = true
	this.listener.Close()
}
