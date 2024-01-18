package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime/mock_workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

func Test_checksums(t *testing.T) {
	c, err := checksums(context.TODO(), os.DirFS("."), "main.go")
	require.NoError(t, err)
	t.Log(c)
}

func Test_perform(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	mockHTTPClient := mock_cdsclient.NewMockHTTPClient(ctrl)
	mockWorker := mock_workerruntime.NewMockRuntime(ctrl)

	mockHTTPClient.EXPECT().Do(reqMatcher{method: "POST", urlPath: "/v2/result"}).DoAndReturn(
		func(req *http.Request) (*http.Response, error) {
			var rrRequest workerruntime.V2RunResultRequest
			btes, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			require.NoError(t, sdk.JSONUnmarshal(btes, &rrRequest))
			require.Equal(t, "main.go", rrRequest.RunResult.Detail.Data.(*sdk.V2WorkflowRunResultGenericDetail).Name)

			h := workerruntime.V2_runResultHandler(context.TODO(), mockWorker)

			rec := httptest.NewRecorder()
			apiReq := http.Request{
				Method: "POST",
				URL:    &url.URL{},
			}
			apiReq.Body = io.NopCloser(bytes.NewBuffer(btes))
			h(rec, &apiReq)
			return rec.Result(), nil
		},
	)

	mockHTTPClient.EXPECT().Do(reqMatcher{method: "PUT", urlPath: "/v2/result"}).DoAndReturn(
		func(req *http.Request) (*http.Response, error) {
			var rrRequest workerruntime.V2RunResultRequest
			btes, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			require.NoError(t, sdk.JSONUnmarshal(btes, &rrRequest))
			require.Equal(t, "main.go", rrRequest.RunResult.Detail.Data.(*sdk.V2WorkflowRunResultGenericDetail).Name)

			h := workerruntime.V2_runResultHandler(context.TODO(), mockWorker)

			rec := httptest.NewRecorder()
			apiReq := http.Request{
				Method: "PUT",
				URL:    &url.URL{},
			}
			apiReq.Body = io.NopCloser(bytes.NewBuffer(btes))
			h(rec, &apiReq)
			return rec.Result(), nil
		},
	)

	mockHTTPClient.EXPECT().Do(reqMatcher{method: "POST", urlPath: "cdn-address/item/upload"}).DoAndReturn(
		func(req *http.Request) (*http.Response, error) {
			btes, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			require.True(t, strings.HasPrefix(string(btes), "package main"))
			item := sdk.CDNItem{ID: sdk.UUID(), APIRefHash: sdk.RandomString(10), Type: sdk.CDNTypeItemRunResultV2}
			btes, _ = json.Marshal(item)
			return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(bytes.NewBuffer(btes))}, nil
		},
	)

	mockWorker.EXPECT().V2AddRunResult(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, req workerruntime.V2RunResultRequest) (*workerruntime.V2AddResultResponse, error) {
			require.Equal(t, "main.go", req.RunResult.Detail.Data.(*sdk.V2WorkflowRunResultGenericDetail).Name)
			return &workerruntime.V2AddResultResponse{
				RunResult:  req.RunResult,
				CDNAddress: "cdn-address",
			}, nil
		},
	)

	mockWorker.EXPECT().V2UpdateRunResult(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, req workerruntime.V2RunResultRequest) (*workerruntime.V2UpdateResultResponse, error) {
			require.Equal(t, "main.go", req.RunResult.Detail.Data.(*sdk.V2WorkflowRunResultGenericDetail).Name)
			return &workerruntime.V2UpdateResultResponse{
				RunResult: req.RunResult,
			}, nil
		},
	)

	plugin := new(runActionUploadArtifactPlugin)
	plugin.Common = actionplugin.Common{HTTPPort: 1, HTTPClient: mockHTTPClient}

	err := plugin.perform(context.TODO(), os.DirFS("."), "*.go !*_test.go", "warn")
	require.NoError(t, err)
}

type reqMatcher struct {
	method  string
	urlPath string
}

func (m reqMatcher) Matches(x interface{}) bool {
	switch i := x.(type) {
	case *http.Request:
		return i.URL.Path == m.urlPath && m.method == i.Method
	default:
		return false
	}
}

func (m reqMatcher) String() string {
	return fmt.Sprintf("Method is %q, URL Path is %q", m.method, m.urlPath)
}
