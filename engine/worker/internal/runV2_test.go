package internal

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/ovh/cds/engine/worker/internal/plugin/mock"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/log/hook"
	"github.com/stretchr/testify/require"
)

func TestRunJobContinueOnError(t *testing.T) {
	var w = new(CurrentWorker)
	w.pluginFactory = &mock.MockFactory{Result: []string{sdk.StatusFail, sdk.StatusSuccess}}
	ctx := context.TODO()
	w.currentJobV2.runJob = &sdk.V2WorkflowRunJob{
		ID:     sdk.UUID(),
		Status: sdk.StatusBuilding,
		JobID:  "myjob",
		Region: "build",
		Job: sdk.V2Job{
			Region: "build",
			Steps: []sdk.ActionStep{
				{
					ID:              "step-0",
					Run:             "exit 1",
					ContinueOnError: true,
				},
				{
					ID:  "step-1",
					Run: "exit 0",
				},
			},
		},
	}
	w.SetContext(ctx)
	w.currentJobV2.runJobContext = sdk.WorkflowRunJobsContext{}

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
	mockClient.EXPECT().V2QueueJobStepUpdate(gomock.Any(), "build", w.currentJobV2.runJob.ID, gomock.Any()).MaxTimes(4)

	result := w.runJobAsCode(ctx)

	require.Equal(t, 2, len(w.currentJobV2.runJob.StepsStatus))
	require.Equal(t, sdk.StatusSuccess, w.currentJobV2.runJob.StepsStatus["step-0"].Conclusion)
	require.Equal(t, sdk.StatusFail, w.currentJobV2.runJob.StepsStatus["step-0"].Outcome)

	require.Equal(t, sdk.StatusSuccess, w.currentJobV2.runJob.StepsStatus["step-1"].Conclusion)
	require.Equal(t, sdk.StatusSuccess, w.currentJobV2.runJob.StepsStatus["step-1"].Outcome)

	require.Equal(t, sdk.StatusSuccess, result.Status)
}

func TestRunJobContinueAlways(t *testing.T) {
	var w = new(CurrentWorker)
	w.pluginFactory = &mock.MockFactory{Result: []string{sdk.StatusFail, sdk.StatusSuccess}}
	ctx := context.TODO()
	w.currentJobV2.runJob = &sdk.V2WorkflowRunJob{
		ID:     sdk.UUID(),
		Status: sdk.StatusBuilding,
		JobID:  "myjob",
		Region: "build",
		Job: sdk.V2Job{
			Region: "build",
			Steps: []sdk.ActionStep{
				{
					ID:  "step-0",
					Run: "exit 1",
				},
				{
					ID:  "step-1",
					Run: "exit 0",
					If:  "always()",
				},
			},
		},
	}
	w.SetContext(ctx)
	w.currentJobV2.runJobContext = sdk.WorkflowRunJobsContext{}

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
	mockClient.EXPECT().V2QueueJobStepUpdate(gomock.Any(), "build", w.currentJobV2.runJob.ID, gomock.Any()).MaxTimes(4)

	result := w.runJobAsCode(ctx)

	require.Equal(t, 2, len(w.currentJobV2.runJob.StepsStatus))
	require.Equal(t, sdk.StatusFail, w.currentJobV2.runJob.StepsStatus["step-0"].Conclusion)
	require.Equal(t, sdk.StatusFail, w.currentJobV2.runJob.StepsStatus["step-0"].Outcome)

	require.Equal(t, sdk.StatusSuccess, w.currentJobV2.runJob.StepsStatus["step-1"].Conclusion)
	require.Equal(t, sdk.StatusSuccess, w.currentJobV2.runJob.StepsStatus["step-1"].Outcome)

	require.Equal(t, sdk.StatusFail, result.Status)
}
