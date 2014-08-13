package frugal

import (
	"bytes"
	"errors"
	"net"
	"time"
)

var ErrSocketClosed = errors.New("socket was closed")

const kReadBufferSize int = 4096

// Helper interface around a socket.
type Socket struct {
	cn      net.Conn
	timeout time.Duration
	closed  error

	// Network data is received into a fixed-size read buffer, and calls to Read()
	// access this buffer. If the buffer is depleted, the network is read again.
	readBuffer     []byte
	readPos        int
	readLimit      int

	// If true, the socket has just been selected for re-use. This is cleared after
	// the first successful call to Read().
	reusing        bool

	// All writes accumulate into this buffer.
	writeBuffer    bytes.Buffer

	// This is the "resend" buffer. If the connection appears to be dead, it is
	// used to resend any data.
	resendBuffer   []byte
}

func dialHostAndPort(hostAndPort string, timeout time.Duration) (net.Conn, error) {
	addr, err := net.ResolveTCPAddr("tcp", hostAndPort)
	if err != nil {
		return nil, err
	}
	cn, err := net.DialTimeout(addr.Network(), addr.String(), timeout)
	if err != nil {
		return nil, err
	}
	return cn, nil
}

// Allocate a new socket using the given host:port string and timeout duration.
func NewSocket(hostAndPort string, timeout time.Duration) (*Socket, error) {
	cn, err := dialHostAndPort(hostAndPort, timeout)
	if err != nil {
		return nil, err
	}

	return &Socket{
		cn:      cn,
		timeout: timeout,
		readBuffer: make([]byte, kReadBufferSize),
	}, nil
}

// Provided for TTransport compatibility; the socket is always open unless it
// is explicitly closed.
func (this *Socket) Open() error {
	return this.closed
}

// Provided for TTransport compatibility; the socket is always open unless it
// is explicitly closed.
func (this *Socket) IsOpen() bool {
	return this.closed == nil
}

// Close the socket.
func (this *Socket) Close() error {
	this.closed = ErrSocketClosed
	return this.cn.Close()
}

// Returns true if there is more data to be read or the remote side is still open
func (this *Socket) Peek() bool {
	return this.IsOpen()
}

// Extend the timeout deadline based on the current time and the socket's
// allowable timeout.
func (this *Socket) Reuse() error {
	this.cn.SetDeadline(this.extendedDeadline())
	this.reusing = true
	return nil
}

func (this *Socket) extendedDeadline() time.Time {
	var t time.Time
	if this.timeout > 0 {
		t = time.Now().Add(this.timeout)
	}
	return t
}

func (this *Socket) Read(buf []byte) (int, error) {
	this.cn.SetReadDeadline(this.extendedDeadline())

	// Check if we need to refill the read buffer.
	if this.readPos == this.readLimit {
		this.readPos = 0
		this.readLimit = 0
		n, err := this.cn.Read(this.readBuffer)
		if err != nil {
			return n, err
		}
		this.readLimit = n
	}

	n := copy(buf, this.readBuffer[this.readPos:this.readLimit])
	this.readPos += n
	return n, nil
}

func (this *Socket) Write(bytes []byte) (int, error) {
	this.cn.SetWriteDeadline(this.extendedDeadline())
	this.writeBuffer.Write(bytes)
	return len(bytes), nil
}

// Returns the remote address as a string.
func (this *Socket) RemoteAddr() string {
	return this.cn.RemoteAddr().String()
}

func (this *Socket) Flush() error {
	if this.writeBuffer.Len() == 0 {
		return nil
	}

	bytes := this.writeBuffer.Bytes()
	this.writeBuffer.Reset()

	for written := 0; written < len(bytes); {
		n, err := this.cn.Write(bytes[written:])
		if err != nil {
			return err
		}
		written += n
	}

	return nil
}
