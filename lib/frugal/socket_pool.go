package frugal

import (
	"sync"
)

type SocketPool struct {
	factory     ServiceFactory
	maxIdle     int
	connections []*SocketAndProtocol
	lock        sync.Mutex
}

// Create a new socket pool with a given maximum number of idle connections.
func NewSocketPool(factory ServiceFactory, maxIdle int) *SocketPool {
	return &SocketPool{
		factory: factory,
		maxIdle: maxIdle,
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

// Get a transport and protocol.
func (this *SocketPool) Get() (*SocketAndProtocol, error) {
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
	if *err != nil {
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
