package tests

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type testHTTPClient struct {
	t *testing.T
}

func (c *testHTTPClient) Do(r *http.Request) (*http.Response, error) {
	router := testsEngine[c.t]
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	res := &http.Response{}
	res.Body = nopCloser{w.Body}
	res.Header = w.Header()
	res.StatusCode = w.Code
	return res, nil
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

func getTestHTTPClient(t *testing.T) *testHTTPClient {
	router := testsEngine[t]
	if router == nil {
		t.Fail()
		return nil
	}
	return &testHTTPClient{t}
}
