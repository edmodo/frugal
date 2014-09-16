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
	"io"
	"net"
	"syscall"
	"time"
)

// Extension of Socket that supports resuming client connections when the pipe
// is broken.
type ResumeableSocket struct {
	*Socket

	// If false, the socket has been idle in a pool. It is not "verified" until a
	// successful call to Read(). It is true immediately after connection.
	verified bool

	// This is the "resend" buffer. If the connection appears to be dead, it is
	// used to resend any data. We use a list of bytes (1) so we don't have to
	// append slices together and (2) because multiple calls to Flush() aren't
	// guaranteed to fail. Only a Read() can tell us whether the pipe is truly
	// broken.
	resendBuffer [][]byte
}

// Creates a new resumeable socket with a given host/port and timeout.
func NewResumeableSocket(hostAndPort string, timeout time.Duration) (*ResumeableSocket, error) {
	socket, err := NewSocket(hostAndPort, timeout)
	if err != nil {
		return nil, err
	}

	return &ResumeableSocket{socket, true, nil}, nil
}

// Implements Transport.Reuse.
func (this *ResumeableSocket) Reuse() error {
	if err := this.Socket.Reuse(); err != nil {
		return err
	}
	this.verified = false
	this.resendBuffer = nil
	return nil
}

// Implements Transport.Read. If this is the first Read() call after a call to
// NewResumeableSocket() or Reuse(), and it fails, then the connection is
// re-established and any previously sent data is resent.
func (this *ResumeableSocket) Read(buf []byte) (int, error) {
	n, err := this.Socket.Read(buf)
	if err == nil {
		// We got a successful read, so verify the connection.
		this.verified = true
		return n, nil
	}

	// If we got no bytes and the connection died, restart and try again.
	if n == 0 {
		if err = this.tryRestart(err); err != nil {
			return n, err
		}
		return this.Socket.recv(buf)
	}

	// Return whatever we received.
	return n, err
}

// Only restart from broken pipes, which happen when either end of the socket
// closes.
func (this *ResumeableSocket) isRestartable(err error) bool {
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

func (this *ResumeableSocket) tryRestart(err error) error {
	// This socket was verified to be working, either via an initial call to Dial
	// or from a successful receive. Any failure now is a real failure.
	if this.verified {
		return err
	}

	if err = this.redial(); err != nil {
		return err
	}
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

// Flush the socket. If the operation fails due to a closed connection, the
// socket is redialed and all previous data re-written. This only happens
// if no calls to Read() have been made since either Reuse() or diailing.
func (this *ResumeableSocket) Flush() error {
	// Store the write buffer in case we need to resend.
	this.resendBuffer = append(this.resendBuffer, this.writeBuffer.Bytes())

	if err := this.Socket.Flush(); err != nil {
		// Try to redial. This automatically re-flushes the resend buffer.
		if err = this.tryRestart(err); err != nil {
			return err
		}
	}

	return nil
}
