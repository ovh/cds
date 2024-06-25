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
	"github.com/moby/moby/client"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime/mock_workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
)

func Test_dockerPushPlugin_perform(t *testing.T) {
	// If we don't have docker client, skip this test
	if _, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation()); err != nil {
		t.Logf("unable to get instantiate docker client: %v", err)
		t.SkipNow()
	}

	artifactoryRepoPrefix := os.Getenv("ARTIFACTORY_REPO_PREFIX")
	artifactoryURL := os.Getenv("ARTIFACTORY_URL")
	artifactoryToken := os.Getenv("ARTIFACTORY_TOKEN")
	artifactoryUsername := os.Getenv("ARTIFACTORY_USERNAME")

	if artifactoryRepoPrefix == "" {
		artifactoryRepoPrefix = "fsamin-default"
	}
	if artifactoryURL == "" {
		artifactoryURL = "https://artifactory.localhost.local/artifactory"
	}
	if artifactoryToken == "" {
		artifactoryToken = "xxxx"
	}
	if artifactoryUsername == "" {
		artifactoryUsername = "workflow_v2_test_it"
	}

	rtURL, err := url.Parse(artifactoryURL)
	require.NoError(t, err)

	rtHost := rtURL.Host

	log.Factory = log.NewTestingWrapper(t)
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)
	mockHTTPClient := mock_cdsclient.NewMockHTTPClient(ctrl)
	mockWorker := mock_workerruntime.NewMockRuntime(ctrl)

	type args struct {
		image    string
		tags     []string
		registry string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test 1",
			args: args{
				image:    "alpine:latest",
				tags:     []string{"test-1"},
				registry: "fsamin-default-docker" + rtHost,
			},
		},
	}

	mockHTTPClient.EXPECT().Do(sdk.ReqMatcher{Method: "POST", URLPath: "/v2/result"}).DoAndReturn(
		func(req *http.Request) (*http.Response, error) {
			var rrRequest workerruntime.V2RunResultRequest
			btes, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			require.NoError(t, sdk.JSONUnmarshal(btes, &rrRequest))
			require.Equal(t, "alpine:latest", rrRequest.RunResult.Detail.Data.(*sdk.V2WorkflowRunResultDockerDetail).Name)

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

	mockHTTPClient.EXPECT().Do(sdk.ReqHostMatcher{Host: rtHost}).DoAndReturn(
		func(req *http.Request) (*http.Response, error) {
			return http.DefaultClient.Do(req)
		},
	).AnyTimes()

	mockWorker.EXPECT().V2AddRunResult(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, req workerruntime.V2RunResultRequest) (*workerruntime.V2AddResultResponse, error) {
			require.Equal(t, "alpine:latest", req.RunResult.Detail.Data.(*sdk.V2WorkflowRunResultDockerDetail).Name)
			var s = "artifactory-integration"
			req.RunResult.ArtifactManagerIntegrationName = &s
			req.RunResult.ID = sdk.UUID()
			return &workerruntime.V2AddResultResponse{
				RunResult: req.RunResult,
			}, nil
		},
	)

	jobCtx := sdk.WorkflowRunJobsContext{
		Integrations: &sdk.JobIntegrationsContexts{
			ArtifactManager: sdk.JobIntegrationsContext{
				Name:   "artifactory-integration",
				Config: map[string]interface{}{},
			},
		},
	}
	mprefix := map[string]interface{}{
		"prefix": artifactoryRepoPrefix,
	}
	jobCtx.Integrations.ArtifactManager.Config["repo"] = mprefix
	jobCtx.Integrations.ArtifactManager.Config[sdk.ArtifactoryConfigURL] = artifactoryURL
	jobCtx.Integrations.ArtifactManager.Config[sdk.ArtifactoryConfigToken] = artifactoryToken
	jobCtx.Integrations.ArtifactManager.Config[sdk.ArtifactoryConfigTokenName] = artifactoryUsername

	mockWorker.EXPECT().V2GetJobContext(gomock.Any()).Return(
		&jobCtx,
	)
	mockHTTPClient.EXPECT().Do(sdk.ReqMatcher{Method: "GET", URLPath: "/v2/context"}).DoAndReturn(
		func(req *http.Request) (*http.Response, error) {
			h := workerruntime.V2_contextHandler(context.TODO(), mockWorker)
			rec := httptest.NewRecorder()
			apiReq := http.Request{
				Method: "GET",
				URL:    &url.URL{},
			}
			q := apiReq.URL.Query()
			q.Add("name", "artifactory-integration")
			apiReq.URL.RawQuery = q.Encode()
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
			require.Equal(t, sdk.V2WorkflowRunResultStatusCompleted, rrRequest.RunResult.Status)

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

	mockWorker.EXPECT().V2UpdateRunResult(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, req workerruntime.V2RunResultRequest) (*workerruntime.V2UpdateResultResponse, error) {
			details, err := req.RunResult.GetDetailAsV2WorkflowRunResultDockerDetail()
			require.NoError(t, err)
			require.Equal(t, artifactoryRepoPrefix+"-docker."+rtHost+"/alpine:test-1", details.Name)
			t.Logf("details:s %+v", details)
			t.Logf("metadata:s %+v", req.RunResult.ArtifactManagerMetadata)
			return &workerruntime.V2UpdateResultResponse{
				RunResult: req.RunResult,
			}, nil
		},
	)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actPlugin := &dockerPushPlugin{
				Common: actionplugin.Common{
					HTTPClient: mockHTTPClient,
					HTTPPort:   1,
				},
			}
			if err := actPlugin.perform(context.TODO(), tt.args.image, tt.args.tags, tt.args.registry, ""); (err != nil) != tt.wantErr {
				t.Errorf("dockerPushPlugin.perform() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
