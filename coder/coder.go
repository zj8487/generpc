// Package coder provides a framework for implementing GeneRPC data coders.
package coder

import (
	"io"
	"net/http"
	"sync"
)

// A Coder decodes and encodes RPC message data.
type Coder interface {
	io.ReadWriter

	// ReadRequests should decode the request(s) into a slice and indicates if
	// the input is a batch or return an error. The returned slice may contain
	// nil values, this indicates that the request data was malformed.
	ReadRequests() (s []*Request, batch bool, e *Error)

	// WriteResponse is called when a single response should be encoded and
	// written to the client.
	WriteResponse(r *Response) error

	// WriteResponses is called when a batch response should be encoded and
	// written to the client.
	WriteResponses(s []*Response) error

	// WriteException is called in case of a Go error that cannot be handled with
	// a RPC error.
	WriteException(id *RequestID, err error) error
}

// RequestID represents an opaque RPC request ID. The coder is responsable for
// parsing and validating the data.
type RequestID []byte

// Request represents a RPC request.
type Request struct {
	Method string
	Params interface{} // []interface{} or map[string]interface{}
	ID     *RequestID
}

// NewResult returns a response object for the given request. It's
// allowed to pass nil as request.
func NewResult(r *Request, v interface{}) *Response {
	return &Response{Result: v, ID: r.ID}
}

// Response represents a RPC response.
type Response struct {
	Result interface{}
	Error  *Error
	ID     *RequestID
}

// Number represents a number value in a particular encoding.
type Number interface {
	CastFloat64() (float64, bool)
	CastInt() (int, bool)
	CastUint() (uint, bool)
}

// NewFn is called when a new coder is required.
type NewFn func(w http.ResponseWriter, r *http.Request) Coder

var fnMap struct {
	sync.Mutex
	m map[string]NewFn
}

func init() {
	fnMap.m = make(map[string]NewFn)
}

// New returns an appropiate coder for the given request. Nil is returned if no
// coder is suitable.
func New(w http.ResponseWriter, r *http.Request) Coder {
	fnMap.Lock()
	defer fnMap.Unlock()

	fn, ok := fnMap.m[r.Header.Get("Content-Type")]
	if !ok {
		return nil
	}

	return fn(w, r)
}

// Register registers a Coder for a particular Content-Type. If Register is
// called twice with the same name or if fn is nil, it panics.
func Register(typ string, fn NewFn) {
	if fn == nil {
		panic("coder: function is nil")
	}

	if _, dup := fnMap.m[typ]; dup {
		panic("coder: Register called twice for type " + typ)
	}

	register(typ, fn)
}

// ReplaceWith registers a Coder like Register except it replaces any existing
// coder. if fn is nil, it panics.
func ReplaceWith(typ string, fn NewFn) {
	if fn == nil {
		panic("coder: function is nil")
	}

	register(typ, fn)
}

func register(typ string, fn NewFn) {
	fnMap.Lock()
	defer fnMap.Unlock()

	fnMap.m[typ] = fn
}
