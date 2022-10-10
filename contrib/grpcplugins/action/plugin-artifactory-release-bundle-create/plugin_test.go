package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
	"github.com/stretchr/testify/require"
)

func runHTTPServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/distribution/api/v1/release_bundle", r.RequestURI)
		content, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close() // nolint
		require.Equal(t, "{\"name\":\"the_name\",\"version\":\"1.0.0\",\"dry_run\":false,\"sign_immediately\":true,\"description\":\"the_description\",\"spec\":{\"queries\":[{\"aql\":\"items.find({\\\"path\\\":{\\\"$ne\\\":\\\".\\\"},\\\"$or\\\":[{\\\"$and\\\":[{\\\"repo\\\":\\\"sgu-cicd-cds-snapshot\\\",\\\"path\\\":\\\"FSAMIN/test/55\\\",\\\"name\\\":\\\"file.txt\\\"}]}]}).include(\\\"name\\\",\\\"repo\\\",\\\"path\\\",\\\"actual_md5\\\",\\\"actual_sha1\\\",\\\"sha256\\\",\\\"size\\\",\\\"type\\\",\\\"modified\\\",\\\"created\\\")\",\"mappings\":[{\"input\":\"^sgu-cicd-cds-snapshot/FSAMIN/test/55/file\\\\.txt$\",\"output\":\"sgu-cicd-cds-release/FSAMIN/test/55/file.txt\"}]}]}}", string(content))
		w.WriteHeader(http.StatusCreated)
	}))
}

func TestRun(t *testing.T) {
	ts := runHTTPServer(t)
	t.Cleanup(func() {
		ts.Close()
	})

	var p = artifactoryReleaseBundleCreatePlugin{}
	var q = actionplugin.ActionQuery{
		Options: map[string]string{
			"name":                                 "the_name",
			"version":                              "1.0.0",
			"description":                          "the_description",
			"release_notes":                        "",
			"cds.integration.artifact_manager.url": ts.URL + "/artifactory/",
			"cds.integration.artifact_manager.release.token": "123456",
			"specification": `
{
	"files": [
	  {
		"pattern": "sgu-cicd-cds-snapshot/FSAMIN/test/55/file.txt",
		"target": "sgu-cicd-cds-release/FSAMIN/test/55/file.txt"
	  }
	]
  }
`,
		},
	}
	res, err := p.Run(context.TODO(), &q)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, sdk.StatusSuccess, res.Status)
}
