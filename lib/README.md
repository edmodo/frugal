lib/frugal
==========

The frugal Go library provides additional helpers on top of the existing Thrift Go API, particularly for extending socket and multi-processing behavior.

The main API entrypoints are:
 - `ServiceFactory` - a replacement for the `TProtocolFactory` and `TTransportFactory` concepts. A service factory has one method, `Connect()`, which must return a socket and a protocol, or an error.
 - `ServiceAndProtocol` - a pair of network socket (conforming to a `TTransport`) and `TProtocol`s for input/output. It can also curry along arbitrary data.
 - `Socket` - a replacement for `TSocket` with more of the networking API exposed.
 - `SocketPool` - allows pooling and re-using of connections for Thrift clients.
