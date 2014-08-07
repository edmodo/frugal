package frugal

import (
	"git.apache.org/thrift.git/lib/go/thrift"
)

// A service factory is responsible for creating transports and protocols. It
// can do that by calling NewSocketAndProtocolFromFactory(). Currently, frugal
// requires the transport to be a frugal.Socket, although this may change in
// the future.
type ServiceFactory interface {
	Connect() (*SocketAndProtocol, error)
}

// A container for socket and protocol information required by thrift. It also
// has an arbitrary payload so consumers can cache and re-use data on a
// per-connection basis.
type SocketAndProtocol struct {
	socket *Socket
	iprot  thrift.TProtocol
	oprot  thrift.TProtocol

	// The client field may be used be consumers of the socket pool to store extra
	// data associated with the connection.
	Client interface{}
}

// Allocate a new SocketAndProtocol given a frugal.Socket and a TProtocolFactory.
func NewSocketAndProtocolFromFactory(socket *Socket, factory thrift.TProtocolFactory) *SocketAndProtocol {
	return &SocketAndProtocol{
		socket: socket,
		iprot:  factory.GetProtocol(socket),
		oprot:  factory.GetProtocol(socket),
	}
}

// Return the TTransport for Thrift.
func (this *SocketAndProtocol) Transport() thrift.TTransport {
	return this.socket
}

// Return the input TProtocol for Thrift.
func (this *SocketAndProtocol) Input() thrift.TProtocol {
	return this.iprot
}

// Return the output TProtocol for Thrift.
func (this *SocketAndProtocol) Output() thrift.TProtocol {
	return this.oprot
}
