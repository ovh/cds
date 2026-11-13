package forgejo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

func TestToVCSRepoParent(t *testing.T) {
	var fork, origin Repository
	require.NoError(t, json.Unmarshal([]byte(`{"id":2,"full_name":"khopa/cds","fork":true,"parent":{"id":1,"full_name":"ovh/cds"}}`), &fork))
	require.NoError(t, json.Unmarshal([]byte(`{"id":1,"full_name":"ovh/cds"}`), &origin))

	client := &forgejoClient{}
	require.Equal(t, "ovh/cds", client.ToVCSRepo(&fork).Parent)
	require.Empty(t, client.ToVCSRepo(&origin).Parent)
}

func TestRepoByFullname(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/repos/ovh/cds", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"id":1,"full_name":"ovh/cds"}`))
	})
	mux.HandleFunc("GET /api/v1/repos/ovh/unknown", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"The target couldn't be found."}`))
	})
	mux.HandleFunc("GET /api/v1/repos/ovh/broken", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	client := &forgejoClient{client: newForgejoHTTPClient(srv.URL, "user", "token")}

	repo, err := client.RepoByFullname(context.TODO(), "ovh/cds")
	require.NoError(t, err)
	require.Equal(t, "ovh/cds", repo.Fullname)

	_, err = client.RepoByFullname(context.TODO(), "ovh/unknown")
	require.True(t, sdk.ErrorIs(err, sdk.ErrRepoNotFound), "%v", err)

	_, err = client.RepoByFullname(context.TODO(), "ovh/broken")
	require.Error(t, err)
	require.False(t, sdk.ErrorIs(err, sdk.ErrRepoNotFound))
}
