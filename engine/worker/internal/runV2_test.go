package internal

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/engine/worker/internal/plugin/mock"
	"github.com/ovh/cds/engine/worker/pkg/workerruntime"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
	"github.com/ovh/cds/sdk/jws"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/log/hook/graylog"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestIntegToEnvVar(t *testing.T) {
	integ := sdk.JobIntegrationsContext{
		Name: "myInteg",
		Config: sdk.JobIntegratiosContextConfig{
			"build": map[string]interface{}{
				"info": map[string]interface{}{
					"prefix": "myvalue",
					"prefixx": map[string]interface{}{
						"tot": "titi",
					},
				},
				"titi": map[string]interface{}{
					"subPrefix": "my2ndValue",
				},
			},
			"url":        "myurl",
			"token":      "mytoken",
			"token_name": "myuser",
		},
	}

	vars := computeIntegrationConfigToEnvVar(integ, "ARTIFACT_MANAGER")

	require.Equal(t, 7, len(vars))
	require.Equal(t, vars["CDS_INTEGRATION_ARTIFACT_MANAGER_BUILD_INFO_PREFIX"], "myvalue")
	require.Equal(t, vars["CDS_INTEGRATION_ARTIFACT_MANAGER_BUILD_INFO_PREFIXX_TOT"], "titi")
	require.Equal(t, vars["CDS_INTEGRATION_ARTIFACT_MANAGER_BUILD_TITI_SUBPREFIX"], "my2ndValue")
	require.Equal(t, vars["CDS_INTEGRATION_ARTIFACT_MANAGER_URL"], "myurl")
	require.Equal(t, vars["CDS_INTEGRATION_ARTIFACT_MANAGER_NAME"], "myInteg")
	require.Equal(t, vars["CDS_INTEGRATION_ARTIFACT_MANAGER_TOKEN"], "mytoken")
	require.Equal(t, vars["CDS_INTEGRATION_ARTIFACT_MANAGER_TOKEN_NAME"], "myuser")
}

func TestRunJobContinueOnError(t *testing.T) {
	var w = new(CurrentWorker)
	w.pluginFactory = &mock.MockFactory{Result: []string{sdk.StatusFail, sdk.StatusSuccess}}
	ctx := context.TODO()
	w.currentJobV2.runJob = &sdk.V2WorkflowRunJob{
		ID:     sdk.UUID(),
		Status: sdk.V2WorkflowRunJobStatusBuilding,
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
	w.SetContextForTestJobV2(t, ctx)
	w.currentJobV2.runJobContext = sdk.WorkflowRunJobsContext{}

	l, h, err := cdslog.New(ctx, &graylog.Config{Hostname: ""})
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
	require.Equal(t, sdk.V2WorkflowRunJobStatusSuccess, w.currentJobV2.runJob.StepsStatus["step-0"].Conclusion)
	require.Equal(t, sdk.V2WorkflowRunJobStatusFail, w.currentJobV2.runJob.StepsStatus["step-0"].Outcome)

	require.Equal(t, sdk.V2WorkflowRunJobStatusSuccess, w.currentJobV2.runJob.StepsStatus["step-1"].Conclusion)
	require.Equal(t, sdk.V2WorkflowRunJobStatusSuccess, w.currentJobV2.runJob.StepsStatus["step-1"].Outcome)

	require.Equal(t, sdk.V2WorkflowRunJobStatusSuccess, result.Status)
}

func TestRunJobContinueAlways(t *testing.T) {
	var w = new(CurrentWorker)
	w.pluginFactory = &mock.MockFactory{Result: []string{sdk.StatusFail, sdk.StatusSuccess}}
	ctx := context.TODO()
	w.currentJobV2.runJob = &sdk.V2WorkflowRunJob{
		ID:     sdk.UUID(),
		Status: sdk.V2WorkflowRunJobStatusBuilding,
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
	w.SetContextForTestJobV2(t, ctx)
	w.currentJobV2.runJobContext = sdk.WorkflowRunJobsContext{}

	l, h, err := cdslog.New(ctx, &graylog.Config{Hostname: ""})
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
	require.Equal(t, sdk.V2WorkflowRunJobStatusFail, w.currentJobV2.runJob.StepsStatus["step-0"].Conclusion)
	require.Equal(t, sdk.V2WorkflowRunJobStatusFail, w.currentJobV2.runJob.StepsStatus["step-0"].Outcome)

	require.Equal(t, sdk.V2WorkflowRunJobStatusSuccess, w.currentJobV2.runJob.StepsStatus["step-1"].Conclusion)
	require.Equal(t, sdk.V2WorkflowRunJobStatusSuccess, w.currentJobV2.runJob.StepsStatus["step-1"].Outcome)

	require.Equal(t, sdk.V2WorkflowRunJobStatusFail, result.Status)
}

func TestCurrentWorker_runJobServicesReadinessNoService(t *testing.T) {
	var w = new(CurrentWorker)
	w.currentJobV2.runJob = &sdk.V2WorkflowRunJob{}
	result := w.runJobServicesReadiness(context.TODO())
	require.Equal(t, sdk.V2WorkflowRunJobStatusSuccess, result.Status)
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
	require.Equal(t, sdk.V2WorkflowRunJobStatusSuccess, result.Status)
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

	require.Equal(t, sdk.V2WorkflowRunJobStatusFail, result.Status)
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

	w.SetContextForTestJobV2(t, ctx)

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
	require.Equal(t, sdk.V2WorkflowRunJobStatusSuccess, result.Status)
}

func TestCurrentWorker_executeHooksSetupV2(t *testing.T) {
	fs := afero.NewOsFs()
	basedir := "test-" + test.GetTestName(t) + "-" + sdk.RandomString(10) + "-" + fmt.Sprintf("%d", time.Now().Unix())
	t.Logf("Creating worker basedir at %s", basedir)
	require.NoError(t, fs.MkdirAll(basedir, os.FileMode(0755)))

	ctrl := gomock.NewController(t)
	mockClient := mock_cdsclient.NewMockV2WorkerInterface(ctrl)

	mockClient.EXPECT().V2QueuePushJobInfo(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
		func(ctx context.Context, regionName string, jobRunID string, msg sdk.V2SendJobRunInfo) error {
			require.Equal(t, sdk.WorkflowRunInfoLevelInfo, msg.Level)
			return nil
		},
	)
	mockClient.EXPECT().ProjectIntegrationWorkerHookGet(gomock.Any(), gomock.Any()).DoAndReturn(
		func(projectKey string, integrationName string) (*sdk.WorkerHookProjectIntegrationModel, error) {
			return nil, sdk.ErrNotFound
		},
	)

	// Setup test worker
	wk := &CurrentWorker{
		basedir:  afero.NewBasePathFs(fs, basedir),
		cfg:      &workerruntime.WorkerConfig{CDNEndpoint: "https://cdn.local"},
		clientV2: mockClient,
	}

	wk.currentJobV2.runJob = &sdk.V2WorkflowRunJob{
		ID:     sdk.UUID(),
		Status: sdk.StatusBuilding,
		JobID:  "myjob",
		Region: "build",
		Job: sdk.V2Job{
			Region: "build",
			Steps: []sdk.ActionStep{
				{
					ID:              "step-0",
					Run:             "exit 0",
					ContinueOnError: true,
				},
			},
		},
	}
	wk.SetContextForTestJobV2(t, context.TODO())
	wk.currentJobV2.runJobContext = sdk.WorkflowRunJobsContext{}
	wk.currentJobV2.runJobContext.Integrations = &sdk.JobIntegrationsContexts{
		ArtifactManager: sdk.JobIntegrationsContext{
			Name:   "foo",
			Config: sdk.JobIntegratiosContextConfig{},
		},
	}
	wk.currentJobV2.integrations = make(map[string]sdk.ProjectIntegration)

	wk.hooks = []workerHook{{
		Config: sdk.WorkerHookSetupTeardownScripts{
			Priority: 0,
			Label:    "foo",
			Setup: `#!/bin/bash
export NEW_VAR=testfoo
`,
			Teardown: `#!/bin/bash
echo 'done'
`,
		},
		SetupPath:    path.Join(basedir, "setup", "test-hook"),
		TeardownPath: path.Join(basedir, "teardown", "test-hook"),
	}}

	err := wk.setupHooksV2(context.TODO(), wk.currentJobV2, wk.basedir, basedir)
	require.NoError(t, err)

	err = wk.executeHooksSetupV2(context.TODO(), wk.basedir)
	require.NoError(t, err)

	var found bool
	for k, v := range wk.currentJobV2.envFromHooks {
		if k == "NEW_VAR" && v == "testfoo" {
			found = true
			break
		}
	}
	t.Logf("envFromHooks: %v", wk.currentJobV2.envFromHooks)
	require.True(t, found)
}
