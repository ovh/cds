package forgejo

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

func TestPullRequestComment(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		bts, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(bts, &gotBody)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id": 1, "body": "run ended"}`))
	}))
	defer srv.Close()

	client := &forgejoClient{client: newForgejoHTTPClient(srv.URL, "user", "token")}
	err := client.PullRequestComment(context.TODO(), "ovh/cds", sdk.VCSPullRequestCommentRequest{ID: 7, Message: "run ended"})
	require.NoError(t, err)

	// a plain comment is visible to everyone, unlike a review created without event
	require.Equal(t, http.MethodPost, gotMethod)
	require.Equal(t, "/api/v1/repos/ovh/cds/issues/7/comments", gotPath)
	require.Equal(t, map[string]interface{}{"body": "run ended"}, gotBody)
}

func TestPullRequest(t *testing.T) {
	prs := map[string]string{
		"/api/v1/repos/ovh/cds/pulls/1": `{"id": 101, "number": 1, "state": "open", "merged": false}`,
		"/api/v1/repos/ovh/cds/pulls/2": `{"id": 102, "number": 2, "state": "closed", "merged": true}`,
		"/api/v1/repos/ovh/cds/pulls/3": `{"id": 103, "number": 3, "state": "closed", "merged": false}`,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(prs[r.URL.Path]))
	}))
	defer srv.Close()

	client := &forgejoClient{client: newForgejoHTTPClient(srv.URL, "user", "token")}

	open, err := client.PullRequest(context.TODO(), "ovh/cds", "1")
	require.NoError(t, err)
	require.Equal(t, 1, open.ID)
	require.False(t, open.Merged)
	require.False(t, open.Closed)

	merged, err := client.PullRequest(context.TODO(), "ovh/cds", "2")
	require.NoError(t, err)
	require.True(t, merged.Merged)
	require.True(t, merged.Closed)

	declined, err := client.PullRequest(context.TODO(), "ovh/cds", "3")
	require.NoError(t, err)
	require.False(t, declined.Merged)
	require.True(t, declined.Closed)
}
