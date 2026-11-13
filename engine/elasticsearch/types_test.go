package elasticsearch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/opensearch-project/opensearch-go/v4"
	"github.com/opensearch-project/opensearch-go/v4/opensearchapi"
	"github.com/stretchr/testify/require"
)

func newTestESClient(t *testing.T, status int) (*esClient, *http.Request) {
	var received http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = *r
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"_index":"jobs","_id":"42","result":"created"}`))
	}))
	t.Cleanup(srv.Close)

	client, err := opensearchapi.NewClient(opensearchapi.Config{
		Client: opensearch.Config{Addresses: []string{srv.URL}},
	})
	require.NoError(t, err)
	return &esClient{client: client}, &received
}

func Test_esClient_IndexDocWithExternalVersion(t *testing.T) {
	c, received := newTestESClient(t, http.StatusCreated)

	_, err := c.IndexDocWithExternalVersion(context.TODO(), "jobs", "42", 1000002, map[string]string{"job": "build"})
	require.NoError(t, err)
	require.Equal(t, http.MethodPut, received.Method)
	require.Equal(t, "/jobs/_doc/42", received.URL.Path)
	require.Equal(t, url.Values{"version": {"1000002"}, "version_type": {"external_gte"}}, received.URL.Query())
}

func Test_esClient_IndexDocWithExternalVersion_conflict(t *testing.T) {
	c, _ := newTestESClient(t, http.StatusConflict)

	_, err := c.IndexDocWithExternalVersion(context.TODO(), "jobs", "42", 1000001, map[string]string{"job": "build"})
	require.ErrorIs(t, err, errVersionConflict)
}

func Test_esClient_IndexDocWithoutType(t *testing.T) {
	c, received := newTestESClient(t, http.StatusCreated)

	_, err := c.IndexDocWithoutType(context.TODO(), "jobs", "42", map[string]string{"job": "build"})
	require.NoError(t, err)
	require.Equal(t, "/jobs/_doc/42", received.URL.Path)
	require.Empty(t, received.URL.RawQuery)
}
