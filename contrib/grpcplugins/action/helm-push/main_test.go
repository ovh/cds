package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime/mock_workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
)

func Test_helmPushPlugin(t *testing.T) {
	artifactoryRepoPrefix := os.Getenv("ARTIFACTORY_REPO_PREFIX")
	artifactoryURL := os.Getenv("ARTIFACTORY_URL")
	artifactoryToken := os.Getenv("ARTIFACTORY_TOKEN")
	artifactoryUsername := os.Getenv("ARTIFACTORY_USERNAME")

	if artifactoryRepoPrefix == "" {
		artifactoryRepoPrefix = "fsamin-default"
	}

	log.Factory = log.NewTestingWrapper(t)

	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	mockWorker := mock_workerruntime.NewMockRuntime(ctrl)
	mockHTTPClient := mock_cdsclient.NewMockHTTPClient(ctrl)

	p := helmPushPlugin{
		Common: actionplugin.Common{
			HTTPClient: mockHTTPClient,
			HTTPPort:   1,
		},
	}

	mockHTTPClient.EXPECT().Do(sdk.ReqNotHostMatcher{NotHost: "127.0.0.1:1"}).DoAndReturn(
		func(req *http.Request) (*http.Response, error) {
			return http.DefaultClient.Do(req)
		},
	).AnyTimes()

	mockHTTPClient.EXPECT().Do(sdk.ReqMatcher{Method: "POST", URLPath: "/v2/result"}).DoAndReturn(
		func(req *http.Request) (*http.Response, error) {
			var rrRequest workerruntime.V2RunResultRequest
			btes, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			require.NoError(t, sdk.JSONUnmarshal(btes, &rrRequest))
			require.Equal(t, "buildachart", rrRequest.RunResult.Detail.Data.(*sdk.V2WorkflowRunResultHelmDetail).Name)

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

	mockHTTPClient.EXPECT().Do(sdk.ReqMatcher{Method: "PUT", URLPath: "/v2/result"}).DoAndReturn(
		func(req *http.Request) (*http.Response, error) {
			var rrRequest workerruntime.V2RunResultRequest
			btes, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			require.NoError(t, sdk.JSONUnmarshal(btes, &rrRequest))
			require.Equal(t, "buildachart", rrRequest.RunResult.Detail.Data.(*sdk.V2WorkflowRunResultHelmDetail).Name)

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

	mockHTTPClient.EXPECT().Do(sdk.ReqMatcher{Method: "GET", URLPath: "/v2/integrations/artifactory-integration"}).DoAndReturn(
		func(req *http.Request) (*http.Response, error) {
			h := workerruntime.V2_integrationsHandler(context.TODO(), mockWorker)
			rec := httptest.NewRecorder()
			apiReq := http.Request{
				Method: "POST",
				URL:    &url.URL{},
			}
			q := apiReq.URL.Query()
			q.Add("name", "artifactory-integration")
			apiReq.URL.RawQuery = q.Encode()
			h(rec, &apiReq)
			return rec.Result(), nil
		},
	)

	integ := sdk.ProjectIntegration{
		ID:                 1,
		ProjectID:          1,
		Name:               "artifactory-integration",
		IntegrationModelID: 1,
		Model:              sdk.ArtifactoryIntegration,
		Config:             sdk.ArtifactoryIntegration.DefaultConfig.Clone(),
	}

	integ.Config.SetValue(sdk.ArtifactoryConfigRepositoryPrefix, artifactoryRepoPrefix)
	integ.Config.SetValue(sdk.ArtifactoryConfigURL, artifactoryURL)
	integ.Config.SetValue(sdk.ArtifactoryConfigToken, artifactoryToken)
	integ.Config.SetValue(sdk.ArtifactoryConfigTokenName, artifactoryUsername)
	integ.Config.SetValue(sdk.ArtifactoryConfigPromotionLowMaturity, "snapshot")

	mockWorker.EXPECT().V2GetIntegrationByName(gomock.Any(), "artifactory-integration").Return(
		&integ, nil,
	)

	mockWorker.EXPECT().V2AddRunResult(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, req workerruntime.V2RunResultRequest) (*workerruntime.V2AddResultResponse, error) {
			require.Equal(t, "buildachart", req.RunResult.Detail.Data.(*sdk.V2WorkflowRunResultHelmDetail).Name)
			var s = "artifactory-integration"
			req.RunResult.ArtifactManagerIntegrationName = &s
			req.RunResult.ID = sdk.UUID()
			return &workerruntime.V2AddResultResponse{
				RunResult: req.RunResult,
			}, nil
		},
	)

	mockWorker.EXPECT().V2UpdateRunResult(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, req workerruntime.V2RunResultRequest) (*workerruntime.V2UpdateResultResponse, error) {
			require.Equal(t, "buildachart", req.RunResult.Detail.Data.(*sdk.V2WorkflowRunResultHelmDetail).Name)
			var s = "artifactory-integration"
			req.RunResult.ArtifactManagerIntegrationName = &s
			req.RunResult.ID = sdk.UUID()
			return &workerruntime.V2UpdateResultResponse{
				RunResult: req.RunResult,
			}, nil
		},
	)

	_, _, err := p.perform(context.TODO(), "fixtures/chart", "", "", false, chartMuseumOptions{})
	require.NoError(t, err)

}
