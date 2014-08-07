package frugal

import (
	"git.apache.org/thrift.git/lib/go/thrift"
)

type ServiceFactory interface {
	Connect() (*SocketAndProtocol, error)
}

// A transport and protocol
type SocketAndProtocol struct {
	socket *Socket
	iprot  thrift.TProtocol
	oprot  thrift.TProtocol

	// The client field may be used be consumers of the socket pool to store extra
	// data associated with the connection.
	Client interface{}
}

func NewSocketAndProtocolFromFactory(socket *Socket, factory thrift.TProtocolFactory) *SocketAndProtocol {
	return &SocketAndProtocol{
		socket: socket,
		iprot:  factory.GetProtocol(socket),
		oprot:  factory.GetProtocol(socket),
	}
}

func (this *SocketAndProtocol) Transport() thrift.TTransport {
	return this.socket
}

func (this *SocketAndProtocol) Input() thrift.TProtocol {
	return this.iprot
}

func (this *SocketAndProtocol) Output() thrift.TProtocol {
	return this.oprot
}
