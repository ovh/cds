package internal

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
	"github.com/ovh/cds/sdk/jws"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/log/hook"
	"github.com/stretchr/testify/require"
)

func TestGetRunResult(t *testing.T) {
	var w = new(CurrentWorker)
	ctx := context.TODO()

	w.currentJobV2.runJob = &sdk.V2WorkflowRunJob{
		ID:     sdk.UUID(),
		Status: sdk.StatusBuilding,
		JobID:  "myjob",
		Region: "build",
		Job:    sdk.V2Job{},
	}
	w.SetContextForTestJobV2(t, ctx)
	w.currentJobV2.runJobContext = sdk.WorkflowRunJobsContext{}
	signer, err := jws.NewHMacSigner([]byte("12345678"))
	require.NoError(t, err)
	w.signer = signer

	l, h, err := cdslog.New(ctx, &hook.Config{Hostname: ""})
	require.NoError(t, err)
	w.SetGelfLogger(h, l)

	ctrl := gomock.NewController(t)
	mockClient := mock_cdsclient.NewMockV2WorkerInterface(ctrl)
	w.clientV2 = mockClient

	t.Cleanup(func() {
		w.clientV2 = nil
		ctrl.Finish()
	})

	mockClient.EXPECT().V2QueueJobRunResultsGet(gomock.Any(), gomock.Any(), gomock.Any()).Return([]sdk.V2WorkflowRunResult{
		{
			Type: sdk.V2WorkflowRunResultTypeDocker,
			Detail: sdk.V2WorkflowRunResultDetail{
				Type: "V2WorkflowRunResultDockerDetail",
				Data: sdk.V2WorkflowRunResultDockerDetail{
					Name: "my.registry.com/my/image:tagggg",
				},
			},
		},
		{
			Type: sdk.V2WorkflowRunResultTypeHelm,
			Detail: sdk.V2WorkflowRunResultDetail{
				Type: "V2WorkflowRunResultHelmDetail",
				Data: sdk.V2WorkflowRunResultHelmDetail{
					Name: "machart",
				},
			},
		},
	}, nil)

	filter := workerruntime.V2FilterRunResult{
		Pattern: "docker:**/my/image:* helm:machart:*",
	}
	result, err := w.V2GetRunResult(ctx, filter)
	require.NoError(t, err)
	require.Equal(t, 2, len(result.RunResults))
}
