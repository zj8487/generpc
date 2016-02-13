package coder

// Error represents an error during handling the RPC request.
type Error struct {
	Code    int
	Message string
	Data    interface{}
}

// WithString returns an error with str in Data.
func (e Error) WithString(str string) *Error {
	v := e
	v.Data = str
	return &v
}

// WithError returns an error with err as string in Data.
func (e Error) WithError(err error) *Error {
	return e.WithString(err.Error())
}

// Response return a response based on the error. If r is not nil, the ID value
// of the request is used.
func (e Error) Response(r *Request) *Response {
	var id *RequestID

	if r != nil {
		id = r.ID
	}

	return &Response{Error: &e, ID: id}
}

var (
	// ParseError should be returned if the input data is invalid or an error has
	// occurred during decoding and parsing the data.
	//
	// JSON-RPC 2.0 specification:
	//   Invalid JSON was received by the server.
	//   An error occurred on the server while parsing the JSON text.
	ParseError = Error{Code: -32700, Message: "Parse error"}

	// InvalidRequest should be returned if the decoded and parsed input data
	// isn't a valid Request.
	//
	// JSON-RPC 2.0 specification:
	//   The JSON sent is not a valid Request object.
	InvalidRequest = Error{Code: -32600, Message: "Invalid Request"}
)

const (
	serverErrorCodeBegin         = -32000
	serverErrorCodeBeginReserved = -32090
	serverErrorCodeEnd           = -32099
)

func serverError(code int) Error {
	return Error{Code: code, Message: "Server error"}
}

// ServerError returns a "Server error" RPC error with a particular code. Codes
// between -32090 and -32099 are reserved and ServerError will panic.
func ServerError(code int) Error {
	if code < serverErrorCodeBegin || code > serverErrorCodeEnd {
		panic("coder: error code is not valid for use as server error")
	}

	if code >= serverErrorCodeBeginReserved && code <= serverErrorCodeEnd {
		panic("coder: use of reserved server error code")
	}

	return serverError(code)
}

// ExceptionErrorCode is a GeneRPC reserved server error code for exceptional
// situations where an error cannot be handled via a RPC response.
// See ExceptionError and Coder.WriteException.
const ExceptionErrorCode = -32090

// ExceptionError returns a RPC error with ExceptionErrorCode and the passed
// error as string in Data. The intention of this function is to make
// implementing Coder.WriteException easy. See ExceptionErrorCode and
// Coder.WriteException.
func ExceptionError(err error) *Error {
	return serverError(ExceptionErrorCode).WithError(err)
}
