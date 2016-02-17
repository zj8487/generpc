package generpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/dwlnetnl/generpc/coder"
)

type subtractMethod struct{}

func (m subtractMethod) ParseNamedParams(p map[string]interface{}) ([]interface{}, error) {
	minuend, ok := p["minuend"]
	if !ok {
		return nil, errors.New("parameter minuend not provided")
	}

	subtrahend, ok := p["subtrahend"]
	if !ok {
		return nil, errors.New("parameter minuend not provided")
	}

	return []interface{}{minuend, subtrahend}, nil
}

func (m subtractMethod) Invoke(params []interface{}) interface{} {
	// This implementation is unsafe because it doesn't validate the input types.
	p0, _ := params[0].(coder.Number).CastInt()
	p1, _ := params[1].(coder.Number).CastInt()
	return p0 - p1
}

type errorMethod struct{}

func (m errorMethod) ParseNamedParams(p map[string]interface{}) ([]interface{}, error) {
	return []interface{}{}, nil
}

func (m errorMethod) Invoke(params []interface{}) interface{} {
	return coder.Error{Code: 1, Message: "Test error"}
}

type jsonCoderGeneralTestSuite struct {
	suite.Suite
	w *httptest.ResponseRecorder
}

func (s *jsonCoderGeneralTestSuite) SetupTest() {
	s.w = httptest.NewRecorder()
}

func (s *jsonCoderGeneralTestSuite) TestInvalidMethod() {
	r, err := http.NewRequest("GET", "/", nil)
	r.Header.Add("Content-Type", "application/json")
	s.Require().NoError(err)

	NewServer().ServeHTTP(s.w, r)

	s.Equal(http.StatusMethodNotAllowed, s.w.Code)
	s.Equal("POST", s.w.Header().Get("Allow"))
	s.Equal("application/json; charset=utf-8", s.w.Header().Get("Content-Type"))

	want := `{"jsonrpc":"2.0","error":{"code":-32700,"message":"Parse error","data":"invalid HTTP method"},"id":null}` + "\n"
	s.Equal(want, s.w.Body.String())
}

func (s *jsonCoderGeneralTestSuite) TestEmptyBody() {
	r, err := http.NewRequest("POST", "/", strings.NewReader(""))
	r.Header.Add("Content-Type", "application/json")
	s.Require().NoError(err)

	NewServer().ServeHTTP(s.w, r)

	s.Equal("application/json; charset=utf-8", s.w.Header().Get("Content-Type"))

	want := `{"jsonrpc":"2.0","error":{"code":-32700,"message":"Parse error","data":"empty POST body"},"id":null}` + "\n"
	s.Equal(want, s.w.Body.String())
}

func (s *jsonCoderGeneralTestSuite) TestInvalidBody() {
	r, err := http.NewRequest("POST", "/", strings.NewReader("invalid"))
	r.Header.Add("Content-Type", "application/json")
	s.Require().NoError(err)

	NewServer().ServeHTTP(s.w, r)

	s.Equal("application/json; charset=utf-8", s.w.Header().Get("Content-Type"))

	want := `{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request","data":"invalid character 'i' looking for beginning of value"},"id":null}` + "\n"
	s.Equal(want, s.w.Body.String())
}

func (s *jsonCoderGeneralTestSuite) TestInvalidJSON() {
	body := strings.NewReader(`{"jsonrpc": "2.0", "method": "foobar, "params": "bar", "baz]`)

	r, err := http.NewRequest("POST", "/", body)
	r.Header.Add("Content-Type", "application/json")
	s.Require().NoError(err)

	NewServer().ServeHTTP(s.w, r)

	s.Equal("application/json; charset=utf-8", s.w.Header().Get("Content-Type"))

	want := `{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request","data":"invalid character 'p' after object key:value pair"},"id":null}` + "\n"
	s.Equal(want, s.w.Body.String())
}

func Test_jsonCoderGeneralTestSuite(t *testing.T) {
	suite.Run(t, new(jsonCoderGeneralTestSuite))
}

type jsonCoderRequestTestSuite struct {
	suite.Suite
	w *httptest.ResponseRecorder
}

func (s *jsonCoderRequestTestSuite) SetupTest() {
	s.w = httptest.NewRecorder()
}

func (s *jsonCoderRequestTestSuite) TestInvalidVersion() {
	r, err := http.NewRequest("POST", "/", strings.NewReader(`{"jsonrpc":""}`))
	r.Header.Add("Content-Type", "application/json")
	s.Require().NoError(err)

	NewServer().ServeHTTP(s.w, r)

	s.Equal("application/json; charset=utf-8", s.w.Header().Get("Content-Type"))

	want := `{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request","data":"invalid version"},"id":null}` + "\n"
	s.Equal(want, s.w.Body.String())
}

func (s *jsonCoderRequestTestSuite) TestInvalidID() {
	body := strings.NewReader(`{"jsonrpc":"2.0","method":"","id":[]}`)

	r, err := http.NewRequest("POST", "/", body)
	r.Header.Add("Content-Type", "application/json")
	s.Require().NoError(err)

	NewServer().ServeHTTP(s.w, r)

	s.Equal("application/json; charset=utf-8", s.w.Header().Get("Content-Type"))

	want := `{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request","data":"invalid id type"},"id":null}` + "\n"
	s.Equal(want, s.w.Body.String())
}

func (s *jsonCoderRequestTestSuite) TestInvalidRequest() {
	body := strings.NewReader(`{"jsonrpc":"2.0","method":1,"params":"bar"}`)

	r, err := http.NewRequest("POST", "/", body)
	r.Header.Add("Content-Type", "application/json")
	s.Require().NoError(err)

	NewServer().ServeHTTP(s.w, r)

	s.Equal("application/json; charset=utf-8", s.w.Header().Get("Content-Type"))

	want := `{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request","data":"json: cannot unmarshal number into Go value of type string"},"id":null}` + "\n"
	s.Equal(want, s.w.Body.String())
}

func (s *jsonCoderRequestTestSuite) TestIDString() {
	body := strings.NewReader(`{"jsonrpc":"2.0","method":"","id":"id"}`)

	r, err := http.NewRequest("POST", "/", body)
	r.Header.Add("Content-Type", "application/json")
	s.Require().NoError(err)

	NewServer().ServeHTTP(s.w, r)

	s.Equal("application/json; charset=utf-8", s.w.Header().Get("Content-Type"))

	want := `{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"},"id":"id"}` + "\n"
	s.Equal(want, s.w.Body.String())
}

func (s *jsonCoderRequestTestSuite) TestIDInt() {
	body := strings.NewReader(`{"jsonrpc":"2.0","method":"","id":1}`)

	r, err := http.NewRequest("POST", "/", body)
	r.Header.Add("Content-Type", "application/json")
	s.Require().NoError(err)

	NewServer().ServeHTTP(s.w, r)

	s.Equal("application/json; charset=utf-8", s.w.Header().Get("Content-Type"))

	want := `{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"},"id":1}` + "\n"
	s.Equal(want, s.w.Body.String())
}

func (s *jsonCoderRequestTestSuite) TestIDFloat() {
	body := strings.NewReader(`{"jsonrpc":"2.0","method":"","id":1.0}`)

	r, err := http.NewRequest("POST", "/", body)
	r.Header.Add("Content-Type", "application/json")
	s.Require().NoError(err)

	NewServer().ServeHTTP(s.w, r)

	s.Equal("application/json; charset=utf-8", s.w.Header().Get("Content-Type"))

	want := `{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"},"id":1.0}` + "\n"
	s.Equal(want, s.w.Body.String())
}

func (s *jsonCoderRequestTestSuite) TestIDNull() {
	body := strings.NewReader(`{"jsonrpc":"2.0","method":"","id":null}`)

	r, err := http.NewRequest("POST", "/", body)
	r.Header.Add("Content-Type", "application/json")
	s.Require().NoError(err)

	NewServer().ServeHTTP(s.w, r)

	s.Equal("application/json; charset=utf-8", s.w.Header().Get("Content-Type"))

	want := `{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"},"id":null}` + "\n"
	s.Equal(want, s.w.Body.String())
}

func (s *jsonCoderRequestTestSuite) TestInternalMethod() {
	body := strings.NewReader(`{"jsonrpc":"2.0","method":"rpc.method","id":1}`)

	r, err := http.NewRequest("POST", "/", body)
	r.Header.Add("Content-Type", "application/json")
	s.Require().NoError(err)

	NewServer().ServeHTTP(s.w, r)

	s.Equal("application/json; charset=utf-8", s.w.Header().Get("Content-Type"))

	want := `{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"},"id":1}` + "\n"
	s.Equal(want, s.w.Body.String())
}

func (s *jsonCoderRequestTestSuite) TestUnregisteredMethod() {
	body := strings.NewReader(`{"jsonrpc":"2.0","method":"unregistered","id":1}`)

	r, err := http.NewRequest("POST", "/", body)
	r.Header.Add("Content-Type", "application/json")
	s.Require().NoError(err)

	NewServer().ServeHTTP(s.w, r)

	s.Equal("application/json; charset=utf-8", s.w.Header().Get("Content-Type"))

	want := `{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"},"id":1}` + "\n"
	s.Equal(want, s.w.Body.String())
}

func (s *jsonCoderRequestTestSuite) TestNilMethod() {
	body := strings.NewReader(`{"jsonrpc":"2.0","method":"nil","id":1}`)

	r, err := http.NewRequest("POST", "/", body)
	r.Header.Add("Content-Type", "application/json")
	s.Require().NoError(err)

	h := NewServer()
	h.Register("nil", (Method)(nil))
	h.ServeHTTP(s.w, r)

	s.Equal("application/json; charset=utf-8", s.w.Header().Get("Content-Type"))

	want := `{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"},"id":1}` + "\n"
	s.Equal(want, s.w.Body.String())
}

func (s *jsonCoderRequestTestSuite) TestByPosParams() {
	body := strings.NewReader(`{"jsonrpc":"2.0","method":"subtract","params":[42,23],"id":1}`)

	r, err := http.NewRequest("POST", "/", body)
	r.Header.Add("Content-Type", "application/json")
	s.Require().NoError(err)

	h := NewServer()
	h.Register("subtract", subtractMethod{})
	h.ServeHTTP(s.w, r)

	s.Equal("application/json; charset=utf-8", s.w.Header().Get("Content-Type"))

	want := `{"jsonrpc":"2.0","result":19,"id":1}` + "\n"
	s.Equal(want, s.w.Body.String())
}

func (s *jsonCoderRequestTestSuite) TestByNameParams() {
	body := strings.NewReader(`{"jsonrpc":"2.0","method":"subtract","params":{"subtrahend":23,"minuend":42},"id":1}`)

	r, err := http.NewRequest("POST", "/", body)
	r.Header.Add("Content-Type", "application/json")
	s.Require().NoError(err)

	h := NewServer()
	h.Register("subtract", subtractMethod{})
	h.ServeHTTP(s.w, r)

	s.Equal("application/json; charset=utf-8", s.w.Header().Get("Content-Type"))

	want := `{"jsonrpc":"2.0","result":19,"id":1}` + "\n"
	s.Equal(want, s.w.Body.String())
}

func (s *jsonCoderRequestTestSuite) TestByNameParams_error() {
	body := strings.NewReader(`{"jsonrpc":"2.0","method":"subtract","params":{"sub":23,"min":42},"id":1}`)

	r, err := http.NewRequest("POST", "/", body)
	r.Header.Add("Content-Type", "application/json")
	s.Require().NoError(err)

	h := NewServer()
	h.Register("subtract", subtractMethod{})
	h.ServeHTTP(s.w, r)

	s.Equal("application/json; charset=utf-8", s.w.Header().Get("Content-Type"))

	want := `{"jsonrpc":"2.0","error":{"code":-32602,"message":"Invalid params","data":"parameter minuend not provided"},"id":1}` + "\n"
	s.Equal(want, s.w.Body.String())
}

func (s *jsonCoderRequestTestSuite) TestInvalidParams() {
	body := strings.NewReader(`{"jsonrpc":"2.0","method":"subtract","params":null,"id":1}`)

	r, err := http.NewRequest("POST", "/", body)
	r.Header.Add("Content-Type", "application/json")
	s.Require().NoError(err)

	h := NewServer()
	h.Register("subtract", subtractMethod{})
	h.ServeHTTP(s.w, r)

	s.Equal("application/json; charset=utf-8", s.w.Header().Get("Content-Type"))

	want := `{"jsonrpc":"2.0","error":{"code":-32602,"message":"Invalid params","data":"params should be by-position (array) or by-name (object)"},"id":1}` + "\n"
	s.Equal(want, s.w.Body.String())
}

func (s *jsonCoderRequestTestSuite) TestNotification() {
	body := strings.NewReader(`{"jsonrpc":"2.0","method":"subtract","params":[42,23]}`)

	r, err := http.NewRequest("POST", "/", body)
	r.Header.Add("Content-Type", "application/json")
	s.Require().NoError(err)

	h := NewServer()
	h.Register("subtract", subtractMethod{})
	h.ServeHTTP(s.w, r)

	s.Equal("application/json; charset=utf-8", s.w.Header().Get("Content-Type"))

	s.Empty(s.w.Body.String())
}

func (s *jsonCoderRequestTestSuite) TestErrorMethod() {
	body := strings.NewReader(`{"jsonrpc":"2.0","method":"error","params":[],"id":1}`)

	r, err := http.NewRequest("POST", "/", body)
	r.Header.Add("Content-Type", "application/json")
	s.Require().NoError(err)

	h := NewServer()
	h.Register("error", errorMethod{})
	h.ServeHTTP(s.w, r)

	s.Equal("application/json; charset=utf-8", s.w.Header().Get("Content-Type"))

	want := `{"jsonrpc":"2.0","error":{"code":1,"message":"Test error"},"id":1}` + "\n"
	s.Equal(want, s.w.Body.String())
}

func Test_jsonCoderRequestTestSuite(t *testing.T) {
	suite.Run(t, new(jsonCoderRequestTestSuite))
}

type jsonCoderBatchTestSuite struct {
	suite.Suite
	w *httptest.ResponseRecorder
}

func (s *jsonCoderBatchTestSuite) SetupTest() {
	s.w = httptest.NewRecorder()
}

func (s *jsonCoderBatchTestSuite) TestParseError() {
	body := strings.NewReader(`[
		{"jsonrpc":"2.0","method":"sum","params":[1,2,4],"id":1},
		{"jsonrpc":"2.0","method"
	]`)

	r, err := http.NewRequest("POST", "/", body)
	r.Header.Add("Content-Type", "application/json")
	s.Require().NoError(err)

	NewServer().ServeHTTP(s.w, r)

	s.Equal("application/json; charset=utf-8", s.w.Header().Get("Content-Type"))

	want := `{"jsonrpc":"2.0","error":{"code":-32700,"message":"Parse error","data":"invalid character ']' after object key"},"id":null}` + "\n"
	s.Equal(want, s.w.Body.String())
}

func (s *jsonCoderBatchTestSuite) TestEmptyRequest() {
	r, err := http.NewRequest("POST", "/", strings.NewReader(`[]`))
	r.Header.Add("Content-Type", "application/json")
	s.Require().NoError(err)

	NewServer().ServeHTTP(s.w, r)

	s.Equal("application/json; charset=utf-8", s.w.Header().Get("Content-Type"))

	want := `{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request"},"id":null}` + "\n"
	s.Equal(want, s.w.Body.String())
}

func (s *jsonCoderBatchTestSuite) TestInvalidRequest() {
	r, err := http.NewRequest("POST", "/", strings.NewReader(`[1]`))
	r.Header.Add("Content-Type", "application/json")
	s.Require().NoError(err)

	NewServer().ServeHTTP(s.w, r)

	s.Equal("application/json; charset=utf-8", s.w.Header().Get("Content-Type"))

	want := `[{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request"},"id":null}]` + "\n"
	s.Equal(want, s.w.Body.String())
}

func (s *jsonCoderBatchTestSuite) TestInvalidBatch() {
	r, err := http.NewRequest("POST", "/", strings.NewReader(`[1,2,3]`))
	r.Header.Add("Content-Type", "application/json")
	s.Require().NoError(err)

	NewServer().ServeHTTP(s.w, r)

	s.Equal("application/json; charset=utf-8", s.w.Header().Get("Content-Type"))

	want := new(bytes.Buffer)
	err = json.Compact(want, []byte(`[
		{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request"},"id":null},
		{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request"},"id":null},
		{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request"},"id":null}
	]`))

	// Append new line, would be stripped away in json.Compact.
	want.WriteByte('\n')

	s.Require().NoError(err)
	s.Equal(want.String(), s.w.Body.String())
}

func (s *jsonCoderBatchTestSuite) TestInvalidJSON() {
	body := strings.NewReader(`[
	  {"jsonrpc":"2.0","method":"sum","params":[1,2,4],"id":"1"},
	  {"jsonrpc":"2.0","method"
	]`)

	r, err := http.NewRequest("POST", "/", body)
	r.Header.Add("Content-Type", "application/json")
	s.Require().NoError(err)

	NewServer().ServeHTTP(s.w, r)

	s.Equal("application/json; charset=utf-8", s.w.Header().Get("Content-Type"))

	want := `{"jsonrpc":"2.0","error":{"code":-32700,"message":"Parse error","data":"invalid character ']' after object key"},"id":null}` + "\n"
	s.Equal(want, s.w.Body.String())
}

func (s *jsonCoderBatchTestSuite) TestRequests() {
	body := strings.NewReader(`[
		{"jsonrpc":"2.0","method":"subtract","params":[42,23],"id":1},
		{"jsonrpc":"2.0","method":"subtract","params":[42,23],"id":2},
		{"foo":"bar"},
		{"jsonrpc":"2.0","method":"subtract","params":[42,23],"id":3}
	]`)

	r, err := http.NewRequest("POST", "/", body)
	r.Header.Add("Content-Type", "application/json")
	s.Require().NoError(err)

	h := NewServer()
	h.Register("subtract", subtractMethod{})
	h.ServeHTTP(s.w, r)

	s.Equal("application/json; charset=utf-8", s.w.Header().Get("Content-Type"))

	want := new(bytes.Buffer)
	err = json.Compact(want, []byte(`[
		{"jsonrpc":"2.0","result":19,"id":1},
		{"jsonrpc":"2.0","result":19,"id":2},
		{"jsonrpc":"2.0","result":19,"id":3}
	]`))

	// Append new line, would be stripped away in json.Compact.
	want.WriteByte('\n')

	s.Require().NoError(err)
	s.Equal(want.String(), s.w.Body.String())
}

func Test_jsonCoderBatchTestSuite(t *testing.T) {
	suite.Run(t, new(jsonCoderBatchTestSuite))
}

func Test_jsonCoder_WriteContentType(t *testing.T) {
	r := httptest.NewRecorder()
	c := &jsonCoder{ResponseWriter: r}

	c.WriteContentType()
	assert.Equal(t, "application/json; charset=utf-8", r.Header().Get("Content-Type"))
}

func Test_jsonCoder_WriteException(t *testing.T) {
	r := httptest.NewRecorder()
	c := &jsonCoder{ResponseWriter: r}

	err := c.WriteException(nil, errors.New("error"))
	assert.NoError(t, err)

	want := `{"jsonrpc":"2.0","error":{"code":-32090,"message":"Server error","data":"error"},"id":null}` + "\n"
	assert.Equal(t, want, r.Body.String())
}

func Test_jsonNumber_CastFloat64(t *testing.T) {
	cases := []struct {
		in json.Number
		v  float64
		ok bool
	}{
		{"2.0201", 2.0201, true},
		{"2", 2, true},
		{"-2", -2, true},
		{"2..", 0, false},
		{"--2.0201", 0, false},
	}

	for _, c := range cases {
		got, ok := jsonNumber{c.in}.CastFloat64()
		assert.Equal(t, c.v, got)
		assert.Equal(t, c.ok, ok)
	}
}

func Test_jsonNumber_CastInt(t *testing.T) {
	cases := []struct {
		in json.Number
		v  int
		ok bool
	}{
		{"2", 2, true},
		{"-2", -2, true},
		{"2.0", 0, false},
		{"2.0201", 0, false},
		{"18446744073709551615", 9223372036854775807, false},
	}

	for _, c := range cases {
		got, ok := jsonNumber{c.in}.CastInt()
		assert.Equal(t, c.v, got)
		assert.Equal(t, c.ok, ok)
	}
}

func Test_jsonNumber_CastUint(t *testing.T) {
	cases := []struct {
		in json.Number
		v  uint
		ok bool
	}{
		{"2", 2, true},
		{"2.0", 0, false},
		{"-2", 0, false},
		{"-2.0", 0, false},
		{"18446744073709551615", 9223372036854775807, false},
	}

	for _, c := range cases {
		got, ok := jsonNumber{c.in}.CastUint()
		assert.Equal(t, c.v, got)
		assert.Equal(t, c.ok, ok)
	}
}
