package frugal

import (
	"bytes"
	"errors"
	"io"
	"net"
	"syscall"
	"time"
)

var ErrSocketClosed = errors.New("socket was closed")
var ErrPendingReads = errors.New("reads are pending; socket was not flushed")
var ErrPendingWrites = errors.New("writes are pending; socket was not flushed")

const kReadBufferSize int = 4096

// Helper interface around a socket.
type Socket struct {
	hostAndPort string
	cn          net.Conn
	timeout     time.Duration
	closed      error

	// Network data is received into a fixed-size read buffer, and calls to Read()
	// access this buffer. If the buffer is depleted, the network is read again.
	readBuffer []byte
	readPos    int
	readLimit  int

	// If false, the socket has been idle in a pool. It is not "verified" until a
	// successful call to Read(). It is true immediately after connection.
	verified   bool

	// All writes accumulate into this buffer.
	writeBuffer bytes.Buffer

	// This is the "resend" buffer. If the connection appears to be dead, it is
	// used to resend any data. We use a list of bytes (1) so we don't have to
	// append slices together and (2) because multiple calls to Flush() aren't
	// guaranteed to fail. Only a Read() can tell us whether the pipe is truly
	// broken.
	resendBuffer [][]byte
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
		hostAndPort: hostAndPort,
		cn:          cn,
		timeout:     timeout,
		readBuffer:  make([]byte, kReadBufferSize),
		verified:    true,
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
	if this.closed != nil {
		return this.closed
	}
	if this.readLimit != 0 {
		return ErrPendingReads
	}
	if this.writeBuffer.Len() > 0 {
		return ErrPendingWrites
	}

	// Reset everything.
	this.cn.SetDeadline(this.extendedDeadline())
	this.verified = false
	this.resendBuffer = nil
	return nil
}

func (this *Socket) extendedDeadline() time.Time {
	var t time.Time
	if this.timeout > 0 {
		t = time.Now().Add(this.timeout)
	}
	return t
}

func (this *Socket) recv(buf []byte) (int, error) {
	// Check if we need to refill the read buffer.
	if this.readPos == this.readLimit {
		this.readPos = 0
		this.readLimit = 0
		n, err := this.cn.Read(this.readBuffer)
		if err != nil {
			return n, err
		}
		this.readLimit = n

		// We got a successful read, so verify the connection.
		this.verified = true
	}

	n := copy(buf, this.readBuffer[this.readPos:this.readLimit])
	this.readPos += n
	return n, nil
}

func (this *Socket) Read(buf []byte) (int, error) {
	if this.closed != nil {
		return 0, this.closed
	}

	// It is illegal to write data without flushing and then Read(), since the
	// state of what the remote side receives is undefined.
	if this.writeBuffer.Len() > 0 {
		return 0, ErrPendingWrites
	}

	this.cn.SetReadDeadline(this.extendedDeadline())

	n, err := this.recv(buf)

	// If we got no bytes and the connection died, restart and try again.
	if n == 0 {
		if err := this.tryRestart(err); err != nil {
			return n, err
		}
		return this.recv(buf)
	}

	return n, err
}

func (this *Socket) Write(bytes []byte) (int, error) {
	if this.closed != nil {
		return 0, this.closed
	}

	this.cn.SetWriteDeadline(this.extendedDeadline())
	this.writeBuffer.Write(bytes)
	return len(bytes), nil
}

// Returns the remote address as a string.
func (this *Socket) RemoteAddr() string {
	return this.cn.RemoteAddr().String()
}

// Only restart from broken pipes, which happen when either end of the socket
// closes.
func (this *Socket) isRestartable(err error) bool {
	if err == io.EOF {
		return true
	}

	if opError, ok := err.(*net.OpError); ok {
		if opError.Err == syscall.EPIPE && opError.Err == syscall.ECONNRESET {
			return true
		}
	}

	return false
}

func (this *Socket) tryRestart(err error) error {
	// This socket was verified to be working, either via an initial call to Dial
	// or from a successful receive. Any failure now is a real failure.
	if this.verified {
		return err
	}

	// Close the old socket before we continue.
	this.Close()

	// Try reconnecting.
	this.cn, err = dialHostAndPort(this.hostAndPort, this.timeout)
	if err != nil {
		return err
	}

	// Reopen for business.
	this.closed = nil
	this.verified = true

	// Attempt to resend everything that was sent via Flush(). We cannot get here
	// if we've already had a successful call to Read(), so we expect that it's
	// safe to resend everything from the current thrift request.
	resend := this.resendBuffer
	this.resendBuffer = nil
	for _, bytes := range resend {
		if err := this.send(bytes); err != nil {
			return err
		}
	}

	return nil
}

func (this *Socket) send(bytes []byte) error {
	// Write everything until either an error or there's nothing left.
	for written := 0; written < len(bytes); {
		n, err := this.cn.Write(bytes[written:])
		if err != nil {
			return err
		}
		written += n
	}

	return nil
}

func (this *Socket) Flush() error {
	if this.closed != nil {
		return this.closed
	}

	// Steal the write buffer's bytes.
	bytes := this.writeBuffer.Bytes()
	this.writeBuffer.Reset()

	// Add this to the resend buffer, in case we need to reconnect.
	this.resendBuffer = append(this.resendBuffer, bytes)

	if err := this.send(bytes); err != nil {
		if err = this.tryRestart(err); err != nil {
			return err
		}
	}

	return nil
}


func (this *Socket) ReadAll(buf []byte) error {
	received := 0
	for received < len(buf) {
		n, err := this.Read(buf[received:])
		if err != nil {
			return err
		}
		received += n
	}
	return nil
}
