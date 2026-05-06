package grpcplugins

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

func Test_checksums(t *testing.T) {
	c, err := checksums(context.TODO(), nil, os.DirFS("."), "grpcplugins.go")
	require.NoError(t, err)
	t.Log(c)
}

// TestArtifactoryItemUpload_RetryAfterTransientFailure verifies that the
// retry loop in ArtifactoryItemUpload can re-read the body after the first
// attempt failed. The previous implementation passed an io.ReadSeeker that
// was also an io.Closer (typically *os.File) directly to http.NewRequest,
// causing the HTTP client to close the file after the first request and
// the retry to fail with "file already closed".
func TestArtifactoryItemUpload_RetryAfterTransientFailure(t *testing.T) {
	const payload = "fake artifact content"
	var attempts int32
	var receivedBodies []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		receivedBodies = append(receivedBodies, string(body))
		n := atomic.AddInt32(&attempts, 1)
		if n == 1 {
			// Simulate the original Artifactory HTTP 400 from the bug report
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("HTTP Status 400 – Bad Request"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"repo":"r","path":"p","downloadUri":"d","mimeType":"m","size":"21","uri":"u"}`))
	}))
	defer srv.Close()

	common := &actionplugin.Common{HTTPClient: srv.Client()}
	integ := sdk.JobIntegrationsContext{Config: sdk.JobIntegrationsContextConfig{
		sdk.ArtifactoryConfigToken: "fake-token",
	}}

	// Simulate a real *os.File reader (which exposes Close()).
	tmp, err := os.CreateTemp(t.TempDir(), "upload-*.bin")
	require.NoError(t, err)
	_, err = tmp.WriteString(payload)
	require.NoError(t, err)
	_, err = tmp.Seek(0, io.SeekStart)
	require.NoError(t, err)

	res, _, err := ArtifactoryItemUpload(context.Background(), common, integ, tmp, map[string]string{}, srv.URL+"/repo/path/file.zip")
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, int32(2), atomic.LoadInt32(&attempts), "retry should have run a second attempt")
	require.Len(t, receivedBodies, 2, "server must receive the body twice")
	require.Equal(t, payload, receivedBodies[0], "first attempt body must match")
	require.Equal(t, payload, receivedBodies[1], "retry body must match — file must not be closed")
}

// TestArtifactoryItemUpload_URLPathPreservedWithForwardSlashes verifies that
// the upload URL reaches the server with forward slashes only, never with
// backslashes (which used to be produced by filepath.Join on Windows and
// were URL-encoded to %5C, causing Artifactory to return HTTP 400).
func TestArtifactoryItemUpload_URLPathPreservedWithForwardSlashes(t *testing.T) {
	var receivedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// r.URL.Path is already URL-decoded; %5C would surface as a literal '\'.
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"repo":"r","path":"p","downloadUri":"d","mimeType":"m","size":"0","uri":"u"}`))
	}))
	defer srv.Close()

	common := &actionplugin.Common{HTTPClient: srv.Client()}
	integ := sdk.JobIntegrationsContext{Config: sdk.JobIntegrationsContextConfig{
		sdk.ArtifactoryConfigToken: "fake-token",
	}}

	uploadURL := srv.URL + "/repo/my_vcs/foo/proj/wf/1.0.0/foo.zip"
	_, _, err := ArtifactoryItemUpload(context.Background(), common, integ, bytes.NewReader([]byte("x")), map[string]string{}, uploadURL)
	require.NoError(t, err)
	require.NotContains(t, receivedPath, `\`, "URL path must not contain backslashes")
	require.NotContains(t, receivedPath, `%5C`, "URL path must not contain URL-encoded backslashes")
	require.True(t, strings.HasSuffix(receivedPath, "/repo/my_vcs/foo/proj/wf/1.0.0/foo.zip"))
}

// TestNoCloseReadSeeker_HidesCloseMethod ensures the wrapper exposes
// io.ReadSeeker but is NOT an io.Closer, so http.Client cannot close the
// underlying resource.
func TestNoCloseReadSeeker_HidesCloseMethod(t *testing.T) {
	tmp, err := os.CreateTemp(t.TempDir(), "wrap-*.bin")
	require.NoError(t, err)
	_, _ = tmp.WriteString("data")
	_, _ = tmp.Seek(0, io.SeekStart)

	wrapped := noCloseReadSeeker{tmp}
	var asReader io.Reader = wrapped
	_, isCloser := asReader.(io.Closer)
	require.False(t, isCloser, "noCloseReadSeeker must not implement io.Closer")
	_, isSeeker := asReader.(io.Seeker)
	require.True(t, isSeeker, "noCloseReadSeeker must remain a Seeker")
}
