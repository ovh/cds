package internal

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/ovh/cds/engine/worker/internal/plugin/mock"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
	"github.com/ovh/cds/sdk/jws"
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
	w.SetContextForTestJobV2(ctx)
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
	w.SetContextForTestJobV2(ctx)
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

func TestCurrentWorker_runJobServicesReadinessNoService(t *testing.T) {
	var w = new(CurrentWorker)
	w.currentJobV2.runJob = &sdk.V2WorkflowRunJob{}
	result := w.runJobServicesReadiness(context.TODO())
	require.Equal(t, sdk.StatusSuccess, result.Status)
}

func TestCurrentWorker_runJobServicesReadinessWithServiceNoCommand(t *testing.T) {
	var w = new(CurrentWorker)
	w.currentJobV2.runJob = &sdk.V2WorkflowRunJob{
		Job: sdk.V2Job{
			Services: map[string]sdk.V2JobService{
				"srv": {
					Image: "the-image",
					Readiness: sdk.V2JobServiceReadiness{
						Command: "", // no command
					},
				},
			},
		},
	}
	result := w.runJobServicesReadiness(context.TODO())
	require.Equal(t, sdk.StatusSuccess, result.Status)
}

func TestCurrentWorker_runJobServicesReadinessWithServiceCommandNoRetries(t *testing.T) {
	var w = new(CurrentWorker)
	w.currentJobV2.runJob = &sdk.V2WorkflowRunJob{
		Job: sdk.V2Job{
			Services: map[string]sdk.V2JobService{
				"srv": {
					Image: "the-image",
					Readiness: sdk.V2JobServiceReadiness{
						Command: "the_command", // no command
						Retries: 0,             // no retry, it must returns an error
						// notice that internal / timeout are mandatory too
					},
				},
			},
		},
	}
	result := w.runJobServicesReadiness(context.TODO())

	t.Log("err:", result.Error)
	t.Log("status:", result.Status)

	require.Equal(t, sdk.StatusFail, result.Status)
}

func TestCurrentWorker_runJobServicesReadinessWithServiceWithCommand(t *testing.T) {
	var w = new(CurrentWorker)
	w.pluginFactory = &mock.MockFactory{Result: []string{sdk.StatusSuccess}}

	ctx := context.TODO()
	ctx = context.WithValue(ctx, cdslog.Workflow, "THE_WF")
	ctx = context.WithValue(ctx, cdslog.Project, "THE_PRJ")

	workerKey, err := jws.NewRandomSymmetricKey(32)
	require.NoError(t, err)
	signingKey := base64.StdEncoding.EncodeToString(workerKey)

	secretKey := make([]byte, 32)
	_, err = base64.StdEncoding.Decode(secretKey, []byte(signingKey))
	require.NoError(t, err)

	signer, err := jws.NewHMacSigner(secretKey)
	require.NoError(t, err)
	w.signer = signer

	w.SetContextForTestJobV2(ctx)

	ctrl := gomock.NewController(t)
	mockClient := mock_cdsclient.NewMockV2WorkerInterface(ctrl)
	w.clientV2 = mockClient

	b, err := sdk.NewBlur([]string{"the-secret"})
	require.NoError(t, err)

	w.blur = b

	mockClient.EXPECT().V2QueuePushJobInfo(gomock.Any(), "the-region", "the-id-run-job", gomock.Any()).DoAndReturn(
		func(ctx context.Context, regionName string, jobRunID string, msg sdk.V2SendJobRunInfo) error {
			require.Equal(t, sdk.WorkflowRunInfoLevelInfo, msg.Level)
			require.Equal(t, "service srv is ready", msg.Message)
			return nil
		},
	)

	w.currentJobV2.runJob = &sdk.V2WorkflowRunJob{
		Region: "the-region",
		ID:     "the-id-run-job",
		Job: sdk.V2Job{
			Services: map[string]sdk.V2JobService{
				"srv": {
					Image: "the-image",
					Readiness: sdk.V2JobServiceReadiness{
						Command:  "ls",
						Retries:  1,
						Timeout:  "1s",
						Interval: "1s",
					},
				},
			},
		},
	}
	result := w.runJobServicesReadiness(context.TODO())
	require.Equal(t, sdk.StatusSuccess, result.Status)
}
