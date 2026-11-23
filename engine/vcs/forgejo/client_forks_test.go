package forgejo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListForks(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/repos/ovh/cds/forks", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"id":2,"full_name":"khopa/cds","fork":true,"parent":{"id":1,"full_name":"ovh/cds"}}]`))
	})
	mux.HandleFunc("GET /api/v1/repos/ovh/broken/forks", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	client := &forgejoClient{client: newForgejoHTTPClient(srv.URL, "user", "token")}

	forks, err := client.ListForks(context.TODO(), "ovh/cds")
	require.NoError(t, err)
	require.Len(t, forks, 1)
	require.Equal(t, "khopa/cds", forks[0].Fullname)

	// With DISABLE_FORKS the route is not registered: the router answers a plain "404 page not found"
	forks, err = client.ListForks(context.TODO(), "ovh/noforks")
	require.NoError(t, err)
	require.Empty(t, forks)

	_, err = client.ListForks(context.TODO(), "ovh/broken")
	require.Error(t, err)
}
