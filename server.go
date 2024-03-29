package generpc

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/dwlnetnl/generpc/coder"
)

// Method represents a RPC method.
//
// ParamNames contains a slice of parameter names so provided by-name
// parameters can be converted into their by-position representation.
//
// Func is the actual function that is called by the Server. It gets the
// parameters passed via the slice and should return the result. This may be a
// coder.Error. The passed parameters are in by-position representation.
type Method struct {
	ParamNames []string
	Func       func([]interface{}) interface{}
}

// Server implements a RPC HTTP handler.
type Server struct {
	m map[string]*Method
}

// NewServer returns an initialized handler.
func NewServer() *Server { return &Server{m: make(map[string]*Method)} }

// Register registers a RPC method for the given name. It panics if name is
// empty or Method.Func is nil or if there is already a method for the name
// registered. It's considered a programmer error to register a method after
// the HTTP server is serving requests.
func (s *Server) Register(name string, m Method) {
	if name == "" {
		panic("generpc: name is empty")
	}

	if m.Func == nil {
		panic("generpc: Method.Func is nil")
	}

	if _, ok := s.m[name]; ok {
		panic("generpc: method already exists: " + name)
	}

	s.m[name] = &m
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := coder.New(w, r)
	if c == nil {
		ct := r.Header.Get("Content-Type")
		msg := fmt.Sprintf("media type %q is not supported", ct)
		http.Error(w, msg, http.StatusUnsupportedMediaType)
		return
	}

	c.WriteContentType()

	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Header().Set("Allow", "POST")
		e := coder.ParseError.WithString("invalid HTTP method")
		c.WriteResponse(e.Response(nil))
		return
	}

	if r.ContentLength == 0 {
		e := coder.ParseError.WithString("empty POST body")
		c.WriteResponse(e.Response(nil))
		return
	}

	reqs, batch, e := c.ReadRequests()
	if e != nil {
		c.WriteResponse(e.Response(nil))
		return
	}

	var resps []*coder.Response
	for _, req := range reqs {
		if req == nil {
			resps = append(resps, coder.InvalidRequest.Response(nil))
			continue
		}

		resp := s.invokeRequest(req)
		if resp == nil {
			// Notifications should not return a response.
			continue
		}

		resps = append(resps, resp)
	}

	var err error
	if batch {
		err = c.WriteResponses(resps)
	} else {
		switch len(resps) {
		case 0:
			// Request was notification.
		case 1:
			err = c.WriteResponse(resps[0])
		default:
			const errorCode = -32091
			e := coder.ServerError(errorCode).WithString("multiple responses")
			err = c.WriteResponse(e.Response(nil))
		}
	}

	if err != nil {
		err := c.WriteException(nil, err)
		if err != nil {
			http.Error(w, "error: "+err.Error(), http.StatusInternalServerError)
		}
	}
}

// JSON-RPC 2.0 specification:
//   The method does not exist / is not available.
var methodNotFound = coder.Error{Code: -32601, Message: "Method not found"}

// JSON-RPC 2.0 specification:
//   Internal JSON-RPC error.
var internalError = coder.Error{Code: -32603, Message: "Internal error"}

// JSON-RPC 2.0 specification:
//   Invalid method parameter(s).
var invalidParams = coder.Error{Code: -32602, Message: "Invalid params"}

func (s *Server) invokeRequest(req *coder.Request) *coder.Response {
	if req.Method == "" || strings.HasPrefix(req.Method, "rpc.") {
		return methodNotFound.Response(req)
	}

	m, ok := s.m[req.Method]
	if !ok || m == nil {
		return methodNotFound.Response(req)
	}

	var params []interface{}
	switch v := req.Params.(type) {
	case []interface{}:
		params = v

	case map[string]interface{}:
		for _, name := range m.ParamNames {
			p, ok := v[name]
			if !ok {
				info := fmt.Sprintf("Parameter %q not provided", name)
				return invalidParams.WithString(info).Response(req)
			}

			params = append(params, p)
		}

	default:
		info := "params should be by-position (array) or by-name (object)"
		return invalidParams.WithString(info).Response(req)
	}

	result := m.Func(params)

	if *req.ID == nil {
		// Request is a notification.
		return nil
	}

	switch v := result.(type) {
	case coder.Error:
		return v.Response(req)

	default:
		return coder.NewResult(req, result)
	}
}
