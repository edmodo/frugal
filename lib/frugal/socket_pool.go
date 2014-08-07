package frugal

import (
	"errors"
	"sync"
)

var ErrPoolClosed = errors.New("pool is closed")

// A SocketPool is responsible for pooling re-using connections. It has three
// main entry points in its API:
//
//    Get - Returns a cached connection, or makes a new one.
//    Put - Returns a connection to the cache, or discards it if it errored.
//    Close - Closes all cached connections.
type SocketPool struct {
	factory     ServiceFactory
	maxIdle     int
	connections []*SocketAndProtocol
	lock        sync.Mutex
	closed      bool
}

// Create a new socket pool with a given maximum number of idle connections.
func NewSocketPool(factory ServiceFactory, maxIdle int) *SocketPool {
	return &SocketPool{
		factory: factory,
		maxIdle: maxIdle,
		closed: false,
	}
}

// Get a transport and protocol from the cache if one is available.
func (this *SocketPool) getFree() *SocketAndProtocol {
	this.lock.Lock()
	defer this.lock.Unlock()

	if len(this.connections) == 0 {
		return nil
	}

	tp := this.connections[len(this.connections)-1]
	this.connections = this.connections[:len(this.connections)-1]
	return tp
}

// Returns a transport and factory. If any idle transports are available, one
// is returned, otherwise a new one is allocated.
//
// Callers may use SocketAndProtocol.Client to store per-connection data, for
// example, to cache thrift client objects so they don't have to be reallocated.
func (this *SocketPool) Get() (*SocketAndProtocol, error) {
	if this.closed {
		return nil, ErrPoolClosed
	}

	sap := this.getFree()
	if sap != nil {
		sap.socket.ExtendDeadline()
		return sap, nil
	}

	return this.factory.Connect()
}

// Puts a socket and protocol back into the free pool. This is intended to be
// used with |defer|. For example:
//
//     cn, err := pool.Get()
//     if err != nil { ...
//     }
//     defer pool.Put(cn, &err)
func (this *SocketPool) Put(sap *SocketAndProtocol, err *error) {
	if *err != nil || this.closed {
		sap.socket.Close()
		return
	}

	this.lock.Lock()
	defer this.lock.Unlock()

	if len(this.connections) >= this.maxIdle {
		sap.socket.Close()
		return
	}
	this.connections = append(this.connections, sap)
}

// Close all pending connections, then mark the pool as closed so no further
// connections will be cached.
func (this *SocketPool) Close() {
	this.lock.Lock()
	defer this.lock.Unlock()

	for _, sap := range this.connections {
		sap.socket.Close()
	}
	this.connections = nil
	this.closed = true
}
