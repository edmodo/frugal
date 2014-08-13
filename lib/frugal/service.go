package frugal

import (
	"git.apache.org/thrift.git/lib/go/thrift"
)

// A service factory is responsible for creating transports and protocols. It
// can do that by calling NewConnectionFromFactory(). Currently, frugal
// requires the transport to be a frugal.Socket, although this may change in
// the future.
type ServiceFactory interface {
	Connect() (*Connection, error)
}

type Transport interface {
	thrift.TTransport

	// Indicate that the transport is about to be re-used. This is an opportunity
	// to send keepalive messages or indicate if the transport cannot be reused.
	// Errors are logged but are non-fatal; a new connection will be made.
	Reuse() error
}

// A container for socket and protocol information required by thrift. It also
// has an arbitrary payload so consumers can cache and re-use data on a
// per-connection basis.
type Connection struct {
	transport Transport
	iprot     thrift.TProtocol
	oprot     thrift.TProtocol

	// The client field may be used be consumers of the socket pool to store extra
	// data associated with the connection.
	Client interface{}
}

// Allocate a new Connection given a frugal.Socket and a TProtocolFactory.
func NewConnectionFromFactory(transport Transport, factory thrift.TProtocolFactory) *Connection {
	return &Connection{
		transport: transport,
		iprot:     factory.GetProtocol(transport),
		oprot:     factory.GetProtocol(transport),
	}
}

// Return the TTransport for Thrift.
func (this *Connection) Transport() Transport {
	return this.transport
}

// Return the input TProtocol for Thrift.
func (this *Connection) Input() thrift.TProtocol {
	return this.iprot
}

// Return the output TProtocol for Thrift.
func (this *Connection) Output() thrift.TProtocol {
	return this.oprot
}
