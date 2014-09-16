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

// Block until all bytes that can fit in buf are filled.
func ReceiveAll(transport Transport, buf []byte) error {
	received := 0
	for received < len(buf) {
		n, err := transport.Read(buf[received:])
		if err != nil {
			return err
		}
		received += n
	}
	return nil
}
