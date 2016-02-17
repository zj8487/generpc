package generpc

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/dwlnetnl/generpc/coder"
)

func init() {
	coder.Register("application/json", jsonCoderFor)
}

type jsonCoder struct {
	http.ResponseWriter
	*bufio.Reader
}

func jsonCoderFor(w http.ResponseWriter, r *http.Request) coder.Coder {
	return &jsonCoder{w, bufio.NewReader(r.Body)}
}

func (c *jsonCoder) ReadRequests() (reqs []*coder.Request, batch bool, e *coder.Error) {
	data, err := c.Peek(1)
	if err != nil {
		e = coder.ParseError.WithError(err)
		return
	}

	if data[0] == '[' {
		batch = true
		reqs, e = c.jsonReadBatch()
	} else {
		reqs, e = c.jsonReadRequest()
	}

	return
}

func (c *jsonCoder) jsonReadRequest() ([]*coder.Request, *coder.Error) {
	var jr jsonRequest

	d := json.NewDecoder(c)
	d.UseNumber()

	err := d.Decode(&jr)
	if err != nil {
		return nil, coder.InvalidRequest.WithError(err)
	}

	r, e := jr.Request()
	if e != nil {
		return nil, e
	}

	return []*coder.Request{r}, nil
}

func (c *jsonCoder) jsonReadBatch() (reqs []*coder.Request, e *coder.Error) {
	var s []json.RawMessage

	err := json.NewDecoder(c).Decode(&s)
	if err != nil {
		e = coder.ParseError.WithError(err)
		return
	}

	if len(s) == 0 {
		e = &coder.InvalidRequest
		return
	}

	for _, raw := range s {
		var jr jsonRequest

		d := json.NewDecoder(bytes.NewReader(raw))
		d.UseNumber()

		err := d.Decode(&jr)
		if err != nil {
			// Error during parsing request, nil requests will be ignored.
			reqs = append(reqs, nil)
			continue
		}

		r, e := jr.Request()
		if e != nil {
			// Ignore malformed objects in batch.
			continue
		}

		reqs = append(reqs, r)
	}

	return reqs, nil
}

func (c *jsonCoder) WriteContentType() {
	c.Header().Set("Content-Type", "application/json; charset=utf-8")
}

func (c *jsonCoder) WriteResponse(r *coder.Response) error {
	jr := jsonResponseFor(*r)
	return json.NewEncoder(c).Encode(jr)
}

func (c *jsonCoder) WriteResponses(s []*coder.Response) error {
	js := make([]jsonResponse, len(s))

	for i, r := range s {
		js[i] = jsonResponseFor(*r)
	}

	return json.NewEncoder(c).Encode(js)
}

func (c *jsonCoder) WriteException(id *coder.RequestID, err error) error {
	r := coder.Response{
		Error: coder.ExceptionError(err),
		ID:    id,
	}

	return json.NewEncoder(c).Encode(jsonResponseFor(r))
}

type jsonRequest struct {
	V string          `json:"jsonrpc"`
	M string          `json:"method"`
	P interface{}     `json:"params,omitempty"`
	I json.RawMessage `json:"id,omitempty"`
}

const jsonrpcVersion = "2.0"

func (jr jsonRequest) Request() (*coder.Request, *coder.Error) {
	var id coder.RequestID

	if jr.V != jsonrpcVersion {
		return nil, coder.InvalidRequest.WithString("invalid version")
	}

	if jr.I != nil {
		var v interface{}

		err := json.Unmarshal(jr.I, &v)
		if err != nil {
			return nil, coder.ParseError.WithError(err)
		}

		switch v.(type) {
		case string, float64, nil: // json numbers are unmarshaled as float64
		default:
			return nil, coder.InvalidRequest.WithString("invalid id type")
		}

		id = coder.RequestID(jr.I)
	}

	switch p := jr.P.(type) {
	case []interface{}:
		for i, v := range p {
			if v, ok := v.(json.Number); ok {
				p[i] = jsonNumber{v}
			}
		}

	case map[string]interface{}:
		for k, v := range p {
			if v, ok := v.(json.Number); ok {
				p[k] = jsonNumber{v}
			}
		}
	}

	return &coder.Request{Method: jr.M, Params: jr.P, ID: &id}, nil
}

type jsonResponse struct {
	V string           `json:"jsonrpc"`
	R *interface{}     `json:"result,omitempty"`
	E *jsonError       `json:"error,omitempty"`
	I *json.RawMessage `json:"id,omitempty"`
}

var jsonNull = json.RawMessage([]byte("null"))

func jsonResponseFor(r coder.Response) jsonResponse {
	jr := jsonResponse{V: jsonrpcVersion, I: &jsonNull}

	if r.ID != nil {
		rm := json.RawMessage(*r.ID)
		jr.I = &rm
	}

	if r.Error != nil {
		jr.E = jsonErrorFor(r.Error)
	} else {
		jr.R = &r.Result
	}

	return jr
}

type jsonError struct {
	C int         `json:"code"`
	M string      `json:"message"`
	D interface{} `json:"data,omitempty"`
}

func jsonErrorFor(e *coder.Error) *jsonError {
	return &jsonError{e.Code, e.Message, e.Data}
}

type jsonNumber struct {
	json.Number
}

func (n jsonNumber) CastFloat64() (float64, bool) {
	v, err := n.Float64()
	return v, err == nil
}

func (n jsonNumber) CastInt() (int, bool) {
	v, err := n.Int64()
	return int(v), err == nil
}

func (n jsonNumber) CastUint() (uint, bool) {
	v, ok := n.CastInt()

	if v < 0 {
		return 0, false
	}

	return uint(v), ok
}
