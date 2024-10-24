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
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime/mock_workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

func Test_perform(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	log.UnregisterField(log.FieldCaller, log.FieldSourceFile, log.FieldSourceLine, log.FieldStackTrace)
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
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

	mockHTTPClient.EXPECT().Do(sdk.ReqMatcher{Method: "GET", URLPath: "/v2/workerConfig"}).DoAndReturn(
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

	mockHTTPClient.EXPECT().Do(reqMatcher{method: "POST", urlPath: "/v2/result"}).DoAndReturn(
		func(req *http.Request) (*http.Response, error) {
			var rrRequest workerruntime.V2RunResultRequest
			btes, err := io.ReadAll(req.Body)
			assert.NoError(t, err)
			log.Debug(context.TODO(), "modk read content: %s", string(btes))
			err = sdk.JSONUnmarshal(btes, &rrRequest)
			if err != nil {
				log.Debug(context.TODO(), "sdk.JSONUnmarshal error: %s", err.Error())
			}

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
			log.Debug(ctx, "V2AddRunResult")
			require.Equal(t, "main.go", req.RunResult.Detail.Data.(*sdk.V2WorkflowRunResultGenericDetail).Name)
			return &workerruntime.V2AddResultResponse{
				RunResult: req.RunResult,
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

	err := plugin.perform(context.TODO(), ".", "*.go !*_test.go", "warn", sdk.V2WorkflowRunResultTypeGeneric)
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

func TestUnMarshalRunResulRequest(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	log.UnregisterField(log.FieldCaller, log.FieldSourceFile, log.FieldSourceLine, log.FieldStackTrace)

	btes := []byte(`{"RunResult":{"id":"","workflow_run_id":"","workflow_run_job_id":"","run_attempt":0,"issued_at":"2024-09-24T07:13:30.606970365Z","type":"generic","artifact_manager_integration_name":null,"artifact_manager_metadata":null,"detail":{"data":{"name":"main.go","size":3868,"mode":420,"md5":"dcdcb00178f065cd3d728091578b92db","sha1":"cf2c4760aae1fc78f5fc16f301389e3966c85403","sha256":"79d1855a45e7dd816b46364fd2da931af33ebcca756d5c145db89af80d47121b"},"type":"V2WorkflowRunResultGenericDetail"},"sync":null,"status":"PENDING"},"CDNItemLink":{"item":{"id":"","created":"0001-01-01T00:00:00Z","last_modified":"0001-01-01T00:00:00Z","hash":"","api_ref":null,"api_ref_hash":"","status":"","type":"","size":0,"md5":"","to_delete":false}}}`)
	var rrRequest workerruntime.V2RunResultRequest
	require.NoError(t, sdk.JSONUnmarshal(btes, &rrRequest))

	t.Logf("--> %T", rrRequest.RunResult.Detail.Data)

	var err error
	btes, err = json.Marshal(rrRequest)
	require.NoError(t, err)
	t.Log(string(btes))

	require.Equal(t, "main.go", rrRequest.RunResult.Detail.Data.(*sdk.V2WorkflowRunResultGenericDetail).Name)
}
