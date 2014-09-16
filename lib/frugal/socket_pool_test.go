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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SocketPool", func() {
	It("Re-dials a dead connection on writes", func() {
		server := NewTestServer()
		defer server.Stop()

		notifyClosed := make(chan bool)
		closeSignal := make(chan bool)

		err := server.Start(func(client thrift.TTransport) {
			// Wait for a signal to start sending.
			<-closeSignal

			// Immediately close the connection to simulate the server dying.
			client.Close()
			notifyClosed <- true
		})
		Expect(err).To(BeNil())

		// One idle connection.
		pool := NewSocketPool(NewTestClientFactory(), 1)
		sap, err := pool.Get()
		Expect(err).To(BeNil())

		// Immediately put the connection back into the pool.
		pool.Put(sap, &err)

		// We should get the same connection back.
		sap2, err := pool.Get()
		Expect(err).To(BeNil())
		Expect(sap2).To(Equal(sap))

		// Build a large buffer to send so it doesn't just get cached in the kernel.
		// We want to get the EPIPE.
		bytes := make([]byte, 131072*2)

		// Write some stuff.
		n, err := sap.Transport().Write(bytes)
		Expect(err).To(BeNil())
		Expect(n).To(Equal(len(bytes)))
		err = sap.Transport().Flush()
		Expect(err).To(BeNil())

		// Put the connection back into the pool.
		pool.Put(sap, &err)

		// Wake up the goroutine and then wait for it to exit.
		closeSignal <- true
		<-notifyClosed
		server.Stop()

		// Start the server again.
		err = server.Start(func(client thrift.TTransport) {})
		Expect(err).To(BeNil())

		// Grab the idle connection again.
		sap, err = pool.Get()
		Expect(err).To(BeNil())
		Expect(sap).To(Equal(sap2))

		// Try to write some bizytes.
		n, err = sap.Transport().Write(bytes)
		Expect(err).To(BeNil())
		Expect(n).To(Equal(len(bytes)))

		// Flush. This should redial the socket.
		err = sap.Transport().Flush()
		Expect(err).To(BeNil())
	})

	It("Re-dials a dead connection on reads", func() {
		server := NewTestServer()
		defer server.Stop()

		notifyClosed := make(chan bool)
		closeSignal := make(chan bool)

		err := server.Start(func(client thrift.TTransport) {
			// Wait for a signal to start sending.
			<-closeSignal

			// Immediately close the connection to simulate the server dying.
			client.Close()
			notifyClosed <- true
		})
		Expect(err).To(BeNil())

		// One idle connection.
		pool := NewSocketPool(NewTestClientFactory(), 1)
		sap, err := pool.Get()
		Expect(err).To(BeNil())

		// Immediately put the connection back into the pool.
		pool.Put(sap, &err)

		// Wake up the goroutine and then wait for it to exit.
		closeSignal <- true
		<-notifyClosed
		server.Stop()

		// Sample data we should receive.
		data := []byte("Wow!")

		// Start the server again.
		err = server.Start(func(client thrift.TTransport) {
			client.Write(data)

			// Wait for a signal to close.
			<-closeSignal
			client.Close()
			notifyClosed <- true
		})
		Expect(err).To(BeNil())

		// Grab the idle connection.
		sap, err = pool.Get()
		Expect(err).To(BeNil())

		// This sequence will usually just fit into the kernel's buffer, so the
		// initial Flush() will pass.
		bytes := []byte("Hello")
		n, err := sap.Transport().Write(bytes)
		Expect(err).To(BeNil())
		Expect(n).To(Equal(len(bytes)))

		// Flush. This should redial the socket.
		err = sap.Transport().Flush()
		Expect(err).To(BeNil())

		// Ask for some data.
		buffer := make([]byte, len(data))
		err = ReceiveAll(sap.Transport(), buffer)
		Expect(err).To(BeNil())
		Expect(buffer).To(Equal(data))
	})
})
