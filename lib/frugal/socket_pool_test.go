package frugal

import (
	"git.apache.org/thrift.git/lib/go/thrift"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SocketPool", func() {
	It("Re-dials a dead connection", func() {
		server := NewTestServer()
		defer server.Stop()

		wait := make(chan bool)

		err := server.Start(func(transport thrift.TTransport) {
			// Immediately close the connection to simulate the server dying.
			transport.Close()
			wait <- true
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

		pool.Put(sap, &err)

		// Wait for the client goroutine to finish and then terminate the server.
		<-wait
		server.Stop()

		// Start the server again.
		err = server.Start(func(transport thrift.TTransport) {})
		Expect(err).To(BeNil())

		// Grab something out of the pool.
		sap, err = pool.Get()
		Expect(err).To(BeNil())
		Expect(sap).To(Equal(sap2))

		// Try to write some bizytes.
		n, err = sap.Transport().Write(bytes)
		Expect(err).To(BeNil())
		Expect(n).To(Equal(len(bytes)))

		err = sap.Transport().Flush()
		Expect(err).To(BeNil())
	})
})
