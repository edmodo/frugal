package frugal

import (
	"errors"
	"net"
	"time"
)

var ErrSocketClosed = errors.New("socket was closed")

// A socket-based TTransport, as an alternative to TSocket. In particular it
// exposes deadline extension for SocketPool.
type Socket struct {
	cn      net.Conn
	timeout time.Duration
	closed  error
}

func NewSocket(hostAndPort string, timeout time.Duration) (*Socket, error) {
	addr, err := net.ResolveTCPAddr("tcp", hostAndPort)
	if err != nil {
		return nil, err
	}
	cn, err := net.DialTimeout(addr.Network(), addr.String(), timeout)
	if err != nil {
		return nil, err
	}
	return &Socket{
		cn: cn,
		timeout: timeout,
		closed: nil,
	}, nil
}

func (this *Socket) Open() error {
	// Sockets start open, and are only closed if explicitly closed.
	return this.closed
}

func (this *Socket) IsOpen() bool {
	return this.closed == nil
}

func (this *Socket) Close() error {
	this.closed = ErrSocketClosed
	return this.cn.Close()
}

// Returns true if there is more data to be read or the remote side is still open
func (this *Socket) Peek() bool {
	return this.IsOpen()
}

func (this *Socket) Flush() error {
	return nil
}

// Extend the timeout deadline.
func (this *Socket) ExtendDeadline() {
	this.cn.SetDeadline(this.extendedDeadline())
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
	return this.cn.Read(buf)
}

func (this *Socket) Write(bytes []byte) (int, error) {
	this.cn.SetWriteDeadline(this.extendedDeadline())
	return this.cn.Write(bytes)
}

// Returns the remote address as a string.
func (this *Socket) RemoteAddr() string {
	return this.cn.RemoteAddr().String()
}
