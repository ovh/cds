package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime/mock_workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
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

	ctrl := gomock.NewController(t)

	mockHTTPClient := mock_cdsclient.NewMockHTTPClient(ctrl)
	mockWorker := mock_workerruntime.NewMockRuntime(ctrl)

	mockWorker.EXPECT().V2GetJobContext(gomock.Any()).Return(
		&sdk.WorkflowRunJobsContext{},
	)

	mockHTTPClient.EXPECT().Do(sdk.ReqMatcher{Method: "GET", URLPath: "/v2/context"}).DoAndReturn(
		func(req *http.Request) (*http.Response, error) {
			h := workerruntime.V2_contextHandler(context.TODO(), mockWorker)
			rec := httptest.NewRecorder()
			apiReq := http.Request{
				Method: "GET",
				URL:    &url.URL{},
			}
			h(rec, &apiReq)
			return rec.Result(), nil
		},
	)

	plugin := new(artifactoryReleaseBundleCreatePlugin)
	plugin.Common = actionplugin.Common{HTTPPort: 1, HTTPClient: mockHTTPClient}

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
}`,
		},
	}
	res, err := plugin.Run(context.TODO(), &q)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, sdk.StatusSuccess, res.Status, "failed with details: %s", res.Details)
}
