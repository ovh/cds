package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"

	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime/mock_workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

type streamServer struct {
	grpc.ServerStream
	results []*actionplugin.StreamResult
}

func (s *streamServer) Send(res *actionplugin.StreamResult) error {
	s.results = append(s.results, res)
	return nil
}

func TestStream(t *testing.T) {
	var deleteReq *http.Request
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deleteReq = r
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(ts.Close)

	ctrl := gomock.NewController(t)

	mockHTTPClient := mock_cdsclient.NewMockHTTPClient(ctrl)
	mockWorker := mock_workerruntime.NewMockRuntime(ctrl)

	mockWorker.EXPECT().V2GetJobContext(gomock.Any()).Return(
		&sdk.WorkflowRunJobsContext{
			Integrations: &sdk.JobIntegrationsContexts{
				Deployment: sdk.JobIntegrationsContext{
					Name:   "arsenal-eu",
					Config: sdk.JobIntegrationsContextConfig{"host": ts.URL},
				},
			},
		},
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

	plugin := new(arsenalDeploymentPlugin)
	plugin.Common = actionplugin.Common{HTTPPort: 1, HTTPClient: mockHTTPClient}

	stream := new(streamServer)
	err := plugin.Stream(&actionplugin.ActionQuery{
		Options: map[string]string{
			"alternative_name": "my-alternative",
			"token":            "my-token",
		},
	}, stream)
	require.NoError(t, err)
	require.NotEmpty(t, stream.results)
	res := stream.results[len(stream.results)-1]
	require.Equal(t, sdk.StatusSuccess, res.Status, "failed with details: %s", res.Details)

	require.NotNil(t, deleteReq)
	require.Equal(t, http.MethodDelete, deleteReq.Method)
	require.Equal(t, "/alternative/my-alternative", deleteReq.URL.Path)
	require.Equal(t, "my-token", deleteReq.Header.Get("X-Arsenal-Deployment-Token"))
}

func TestStreamMissingToken(t *testing.T) {
	plugin := new(arsenalDeploymentPlugin)

	stream := new(streamServer)
	err := plugin.Stream(&actionplugin.ActionQuery{
		Options: map[string]string{
			"alternative_name": "my-alternative",
		},
	}, stream)
	require.NoError(t, err)
	require.Len(t, stream.results, 1)
	require.Equal(t, sdk.StatusFail, stream.results[0].Status)
	require.Equal(t, "missing arsenal deployment token", stream.results[0].Details)
}
