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
	socket := NewSocketFromConn(conn, this.options.ClientTimeout)
	defer socket.Close()

	// Number of requests serviced off this connection.
	serviced := 0

	iprot, oprot := this.callbacks.GetProtocolsForClient(socket)
	for {
		name, msgType, sequenceId, err := iprot.ReadMessageBegin()
		if err != nil {
			if err.Error() == io.EOF.Error() {
				return
			}

			netErr, ok := err.(net.Error)
			if ok && netErr.Timeout() && serviced >= 1 {
				// We already got data from this connection, and now it's idle. Just keep
				// polling for more data.
				if err := socket.Reuse(); err != nil {
					this.callbacks.LogError("reuse-socket", err)
					return
				}
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
