package cdsclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
)

// TestRequestCarriesARequestID pins that a call leaves with an id of its own. The API mints one on
// arrival, so a call that reaches a handler is traceable either way; a call that never reaches one
// leaves nothing behind on the server, and this header is then the only thing that can tie the
// failure the caller logged to what did or did not happen on the other side.
func TestRequestCarriesARequestID(t *testing.T) {
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get(cdslog.HeaderRequestID))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := New(Config{Host: srv.URL}).(*serviceClient)

	_, _, _, err := c.Request(context.TODO(), http.MethodGet, "/mon/version", nil)
	require.NoError(t, err)

	require.Len(t, seen, 1)
	require.True(t, sdk.IsValidUUID(seen[0]),
		"the api only reuses an incoming id when it is a valid uuid, otherwise it mints its own and the two sides no longer join")
}

// TestRequestReportsTheRequestIDOnFailure pins the half that matters during an incident: when the
// call fails, the id has to be in the error, because that error is all the caller logs.
func TestRequestReportsTheRequestIDOnFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	srv.Close() // nothing is listening: the call cannot reach a handler at all

	c := New(Config{Host: srv.URL}).(*serviceClient)

	_, _, _, err := c.Request(context.TODO(), http.MethodGet, "/mon/version", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "request_id: ")

	id := strings.TrimSpace(strings.SplitN(strings.SplitN(err.Error(), "request_id: ", 2)[1], ")", 2)[0])
	require.True(t, sdk.IsValidUUID(id), "the reported id must be the one that was sent, got %q", id)
}
