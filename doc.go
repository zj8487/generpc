// Package generpc implements a (generalized) JSON-RPC 2.0 HTTP server handler.
//
// GeneRPC implements a generalized version of JSON-RPC 2.0: it decouples the
// wire data format from the RPC layer. This means it implements the JSON-RPC
// 2.0 specification but also provides an abstraction layer for decoding and
// encoding data on the wire, so any wire format can be used.
//
// Coders are allowed to deviate for things like object member names that are
// required by the JSON-RPC 2.0 specification. However it's important to
// document the implemented wire formats. Coders are required to provide valid
// Request structs by decoding wire data and accept Response structs (see coder
// package) for encoding into their wire data format. The specification should
// be closely followed wherever and whenever possible.
//
// This package provides also an implementation of the GeneRPC/JSON data format
// (JSON-RPC 2.0).
//
// The JSON-RPC 2.0 specification can be found at
// http://www.jsonrpc.org/specification.
package generpc
