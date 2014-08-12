package frugal

import (
	"git.apache.org/thrift.git/lib/go/thrift"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SocketPool", func() {
	It("Re-dials a dead connection", func() {
		server := NewTestServer()
		//defer server.Stop()

		err := server.Start(func(transport thrift.TTransport) {
			// Immediately close the connection to simulate the server dying.
			transport.Close()
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

		pool.Put(sap, &err)

		// Terminate the server.
		server.Stop()

		// Start it again.
		err = server.Start(func(transport thrift.TTransport) {
			// Just leave the connection idle.
		})
		Expect(err).To(BeNil())

		// Grab something out of the pool.
		sap, err = pool.Get()
		Expect(err).To(BeNil())

		// Try to write some bizytes.
		bytes := []byte("Hello!")
		n, err := sap.Transport().Write(bytes)
		Expect(err).To(BeNil())
		Expect(n).To(Equal(len(bytes)))
	})
})
