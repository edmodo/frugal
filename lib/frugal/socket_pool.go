package frugal

import (
	"errors"
	"log"
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
	connections []*Connection
	lock        sync.Mutex
	closed      bool
}

// Create a new socket pool with a given maximum number of idle connections.
func NewSocketPool(factory ServiceFactory, maxIdle int) *SocketPool {
	return &SocketPool{
		factory: factory,
		maxIdle: maxIdle,
		closed:  false,
	}
}

// Get a transport and protocol from the cache if one is available.
func (this *SocketPool) getFree() (*Connection, error) {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.closed {
		return nil, ErrPoolClosed
	}

	if len(this.connections) == 0 {
		return nil, nil
	}

	tp := this.connections[len(this.connections)-1]
	this.connections = this.connections[:len(this.connections)-1]

	// Ask to re-use the connection. If that doesn't work, log the error and
	// just signal for a new connection.
	if err := tp.transport.Reuse(); err != nil {
		log.Printf("connection re-use error: %s\n", err.Error())
		return nil, nil
	}
	return tp, nil
}

// Returns a transport and factory. If any idle transports are available, one
// is returned, otherwise a new one is allocated.
//
// Callers may use Connection.Client to store per-connection data, for
// example, to cache thrift client objects so they don't have to be reallocated.
func (this *SocketPool) Get() (*Connection, error) {
	conn, err := this.getFree()
	if err != nil {
		return nil, err
	}
	if conn != nil {
		return conn, nil
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
func (this *SocketPool) Put(conn *Connection, err *error) {
	this.lock.Lock()
	defer this.lock.Unlock()

	if *err != nil || this.closed {
		conn.transport.Close()
		return
	}

	if len(this.connections) >= this.maxIdle {
		conn.transport.Close()
		return
	}
	this.connections = append(this.connections, conn)
}

// Close all pending connections, then mark the pool as closed so no further
// connections will be cached.
func (this *SocketPool) Close() {
	this.lock.Lock()
	defer this.lock.Unlock()

	for _, conn := range this.connections {
		conn.transport.Close()
	}
	this.connections = nil
	this.closed = true
}
