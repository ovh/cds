package cdsclient

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/ovh/cds/sdk"
)

// LocalRoundTripper implements http.RoundTripper by calling an http.Handler
// directly in-process instead of making network calls. It injects the local
// service identity into the request context so that auth middleware
// can identify the caller without JWT tokens.
type LocalRoundTripper struct {
	handler     http.Handler
	serviceName string
	serviceType string
}

// NewLocalRoundTripper creates a RoundTripper that routes requests to the
// given handler in-process. The serviceName and serviceType identify the
// calling service for authentication bypass.
func NewLocalRoundTripper(handler http.Handler, serviceName, serviceType string) *LocalRoundTripper {
	return &LocalRoundTripper{
		handler:     handler,
		serviceName: serviceName,
		serviceType: serviceType,
	}
}

// RoundTrip executes the HTTP request by calling the handler directly.
func (t *LocalRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Inject local service identity into request context
	ctx := sdk.ContextWithLocalService(req.Context(), t.serviceName, t.serviceType)
	req = req.WithContext(ctx)

	// Ensure the request has a valid URL for the handler
	if req.URL.Scheme == "" {
		req.URL.Scheme = "http"
	}
	if req.URL.Host == "" {
		req.URL.Host = "local"
	}

	// Record the response via httptest.ResponseRecorder
	rec := httptest.NewRecorder()
	t.handler.ServeHTTP(rec, req)

	resp := rec.Result()
	// Replace the body with a re-readable buffer so callers can read it fully
	body := rec.Body.Bytes()
	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.ContentLength = int64(len(body))
	resp.Request = req

	return resp, nil
}
