package amock

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"regexp"
	"runtime"
	"runtime/debug"
	"sync"
	"testing"
)

// MockRoundTripper implements http.RoundTripper for mocking/testing purposes
type MockRoundTripper struct {
	sync.Mutex
	Responses map[string][]*Response
}

// ResponsePayload is an interface that the Body object you pass in your expected responses can respect.
// It lets you customize the way your body is handled. If you pass an object that does NOT respect ResponsePayload,
// JSON is the default.
type ResponsePayload interface {
	Payload() ([]byte, error)
}

// Raw respects the ResponsePayload interface. It lets you define a Body object with raw bytes.
type Raw []byte

func (r Raw) Payload() ([]byte, error) {
	return []byte(r), nil
}

// JSON respects the ResponsePayload interface. It encloses another object and marshals it into json.
// This is used if your body object does not respect ResponsePayload.
type JSON struct {
	Obj interface{}
}

func (j JSON) Payload() ([]byte, error) {
	return json.Marshal(j.Obj)
}

// An expected mocked response. Defining traits are status and body.
// Optionally includes conditional filter function defined by one or several On(...) or OnIdentifier(...) calls.
type Response struct {
	Status  int
	headers http.Header
	Body    ResponsePayload
	Cond    func(*http.Request) bool
	*sync.Mutex
}

// NewMock creates a MockRoundTripper object
func NewMock() *MockRoundTripper {
	return &MockRoundTripper{
		Responses: map[string][]*Response{},
	}
}

// Headers adds http headers to the response
func (r *Response) Headers(h http.Header) *Response {
	r.Lock()
	defer r.Unlock()
	r.headers = h
	return r
}

// merges two conditional filter functions into a composite one (logical AND)
func condAND(fs ...func(*http.Request) bool) func(*http.Request) bool {
	return func(r *http.Request) bool {
		for _, f := range fs {
			if !f(r) {
				return false
			}
		}
		return true
	}
}

// OnIdentifier adds a conditional filter to the response.
// The response will be selected only if the HTTP path of the request contains
// "/.../IDENT(/...)": the identifier enclosed in a distinct path segment
func (r *Response) OnIdentifier(ident string) *Response {
	r.Lock()
	defer r.Unlock()
	ident = regexp.QuoteMeta(ident)
	matcher := regexp.MustCompile(`/[^/]+/` + ident + `(?:/.*|$)`)
	cond := func(req *http.Request) bool {
		return matcher.MatchString(req.URL.Path)
	}
	if r.Cond != nil {
		r.Cond = condAND(r.Cond, cond)
	} else {
		r.Cond = cond
	}
	return r
}

// On adds a conditional filter to the response.
func (r *Response) On(f func(*http.Request) bool) *Response {
	r.Lock()
	defer r.Unlock()
	if r.Cond != nil {
		r.Cond = condAND(r.Cond, f)
	} else {
		r.Cond = f
	}
	return r
}

// Expect adds a new expected response, specifying status and body. The other components (headers, conditional filters)
// can be further specified by chaining setter calls on the response object.
func (mc *MockRoundTripper) Expect(callerFunc interface{}, status int, body interface{}) *Response {
	mc.Lock()
	defer mc.Unlock()

	caller := getFunctionName(callerFunc)
	bodyPL, ok := body.(ResponsePayload)
	if !ok {
		bodyPL = JSON{body}
	}
	resp := &Response{Status: status, Body: bodyPL, Mutex: &mc.Mutex}
	mc.Responses[caller] = append(mc.Responses[caller], resp)
	return resp
}

func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

// RoundTrip respects http.RoundTripper. It finds the code path taken to get to here, and returns the first matching expected response.
func (mc *MockRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {

	mc.Lock()
	defer mc.Unlock()

	caller, err := mc.callerFunc()
	if err != nil {
		return nil, err
	}

	if len(mc.Responses[caller]) == 0 {
		return nil, fmt.Errorf("no more calls expected for '%s'", caller)
	}

	var resp *Response

	for i, rsp := range mc.Responses[caller] {
		if rsp.Cond == nil || rsp.Cond(r) {
			// Delete elem in place
			mc.Responses[caller] = append(mc.Responses[caller][:i], mc.Responses[caller][i+1:]...)
			resp = rsp
			break
		}
	}

	if resp == nil {
		return nil, fmt.Errorf("remaining responses for '%s' have unmet conditions", caller)
	}

	var respBody []byte

	if resp.Body != nil {
		respBody, err = resp.Body.Payload()
		if err != nil {
			return nil, err
		}
	}

	return &http.Response{
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Status:        http.StatusText(resp.Status),
		StatusCode:    resp.Status,
		Header:        resp.headers,
		Body:          ioutil.NopCloser(bytes.NewReader(respBody)),
		Request:       r,
		ContentLength: int64(len(respBody)),
	}, nil
}

// AssertEmpty ensures all expected responses have been consumed.
// It will call t.Error() detailing the remaining unconsumed responses.
func (mc *MockRoundTripper) AssertEmpty(t *testing.T) {
	mc.Lock()
	defer mc.Unlock()

	for f, resps := range mc.Responses {
		if len(resps) > 0 {
			t.Error(fmt.Sprintf("%s: %d expected responses remaining", f, len(resps)))
		}
	}
}

// Go up the stack to find which expected code path we went through
func (mc *MockRoundTripper) callerFunc() (string, error) {
	callers := make([]uintptr, 10)
	runtime.Callers(3, callers)
	for _, c := range callers {
		name := runtime.FuncForPC(c).Name()
		if len(mc.Responses[name]) != 0 {
			return name, nil
		}
	}
	return "", fmt.Errorf("unexpected call:\n%s", string(debug.Stack()))
}
