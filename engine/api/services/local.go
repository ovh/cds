package services

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/ovh/cds/sdk"
)

var (
	localHandlersMu sync.RWMutex
	localHandlers   = map[string]localHandler{} // serviceType -> handler
)

type localHandler struct {
	handler    http.Handler
	consumerID string
}

// RegisterLocalHandler registers an in-process HTTP handler for a service type.
// When the API needs to call a co-located service, it will use this handler
// directly instead of making network calls with RSA signatures.
func RegisterLocalHandler(serviceType string, handler http.Handler, consumerID string) {
	localHandlersMu.Lock()
	defer localHandlersMu.Unlock()
	localHandlers[serviceType] = localHandler{
		handler:    handler,
		consumerID: consumerID,
	}
}

// GetLocalHandler returns the in-process handler for a service type, if registered.
func GetLocalHandler(serviceType string) (http.Handler, string, bool) {
	localHandlersMu.RLock()
	defer localHandlersMu.RUnlock()
	lh, ok := localHandlers[serviceType]
	if !ok {
		return nil, "", false
	}
	return lh.handler, lh.consumerID, true
}

// doLocalRequest performs an in-process HTTP request to a co-located service handler.
// It injects the API's consumer identity into the context so the service's
// CheckRequestSignatureMiddleware can bypass RSA verification.
func doLocalRequest(ctx context.Context, handler http.Handler, apiConsumerID string, method, path string, body io.Reader, mods ...RequestModifierFunc) (io.Reader, http.Header, int, error) {
	// Build the request
	req, err := http.NewRequestWithContext(ctx, method, "http://local"+path, body)
	if err != nil {
		return nil, nil, 0, sdk.WithStack(err)
	}

	// Inject the API's local service identity so the service's auth middleware
	// recognizes this as a trusted in-process call
	ctx = sdk.ContextWithLocalService(req.Context(), "cds-api", sdk.TypeAPI)
	req = req.WithContext(ctx)

	req.Header.Set("Connection", "close")
	for _, mod := range mods {
		if mod != nil {
			mod(req)
		}
	}

	// Execute via the handler directly
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	respBody := rec.Body.Bytes()

	if resp.StatusCode < 400 {
		return bytes.NewReader(respBody), resp.Header, resp.StatusCode, nil
	}

	// Try to decode CDS error
	if cdserr := sdk.DecodeError(respBody); cdserr != nil {
		return nil, resp.Header, resp.StatusCode, cdserr
	}
	return nil, resp.Header, resp.StatusCode, sdk.Errorf("local request %s %s failed with status %d", method, path, resp.StatusCode)
}

// RequestModifierFunc is the same as cdsclient.RequestModifier
type RequestModifierFunc func(req *http.Request)
