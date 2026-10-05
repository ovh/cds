package hooks

import (
	"context"
	"testing"
	"time"

	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
)

func TestAnalysisError(t *testing.T) {
	failed := sdk.HookRepositoryEventAnalysis{ProjectKey: "A", Status: sdk.RepositoryAnalysisStatusError, Error: "boom"}
	succeed := sdk.HookRepositoryEventAnalysis{ProjectKey: "B", Status: sdk.RepositoryAnalysisStatusSucceed}
	skipped := sdk.HookRepositoryEventAnalysis{ProjectKey: "C", Status: sdk.RepositoryAnalysisStatusSkipped, Error: "no cds files found"}

	tests := []struct {
		name     string
		analyses []sdk.HookRepositoryEventAnalysis
		want     string
	}{
		{name: "no analysis", want: ""},
		{name: "no failed analysis", analyses: []sdk.HookRepositoryEventAnalysis{succeed, skipped}, want: ""},
		{name: "single failed analysis", analyses: []sdk.HookRepositoryEventAnalysis{failed}, want: "repository analysis failed"},
		{name: "all analyses failed", analyses: []sdk.HookRepositoryEventAnalysis{failed, failed}, want: "all repository analyses failed"},
		{name: "some analyses failed", analyses: []sdk.HookRepositoryEventAnalysis{succeed, failed, skipped}, want: "1 of 3 repository analyses failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, analysisError(&sdk.HookRepositoryEvent{Analyses: tt.analyses}))
		})
	}
}

func TestSkipHooksOfFailedAnalyses(t *testing.T) {
	hre := sdk.HookRepositoryEvent{
		VCSServerName:  "github",
		RepositoryName: "ovh/cds",
		Analyses: []sdk.HookRepositoryEventAnalysis{
			{ProjectKey: "FAILED", Status: sdk.RepositoryAnalysisStatusError, Error: "unable to retrieve files"},
			{ProjectKey: "OK", Status: sdk.RepositoryAnalysisStatusSucceed},
		},
		WorkflowHooks: []sdk.HookRepositoryEventWorkflow{
			// Local workflow of the project whose analysis failed
			{ProjectKey: "FAILED", VCSIdentifier: "GitHub", RepositoryIdentifier: "OVH/cds", WorkflowName: "local", Status: sdk.HookEventWorkflowStatusScheduled},
			// Distant workflow of the project whose analysis failed
			{ProjectKey: "FAILED", VCSIdentifier: "github", RepositoryIdentifier: "ovh/other", WorkflowName: "distant", Status: sdk.HookEventWorkflowStatusScheduled},
			// Distant workflow of a project that does not declare the repository
			{ProjectKey: "DISTANT", VCSIdentifier: "github", RepositoryIdentifier: "ovh/other", WorkflowName: "distant", Status: sdk.HookEventWorkflowStatusScheduled},
			// Local workflow of a project whose analysis succeeded
			{ProjectKey: "OK", VCSIdentifier: "github", RepositoryIdentifier: "ovh/cds", WorkflowName: "local", Status: sdk.HookEventWorkflowStatusScheduled},
			// Already skipped hook keeps its own reason
			{ProjectKey: "FAILED", VCSIdentifier: "github", RepositoryIdentifier: "ovh/cds", WorkflowName: "filtered", Status: sdk.HookEventWorkflowStatusSkipped, Error: "comment does not match comment filter"},
		},
	}

	skipHooksOfFailedAnalyses(&hre)

	require.Equal(t, sdk.HookEventWorkflowStatusSkipped, hre.WorkflowHooks[0].Status)
	require.Equal(t, "repository analysis failed: unable to retrieve files", hre.WorkflowHooks[0].Error)
	for _, i := range []int{1, 2, 3} {
		require.Equal(t, sdk.HookEventWorkflowStatusScheduled, hre.WorkflowHooks[i].Status, "hook %d", i)
		require.Empty(t, hre.WorkflowHooks[i].Error, "hook %d", i)
	}
	require.Equal(t, "comment does not match comment filter", hre.WorkflowHooks[4].Error)
}

func TestTriggerAnalyses_FailedAnalysisDoesNotBlockWorkflowHooks(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	ctx := context.TODO()

	hre := sdk.HookRepositoryEvent{
		UUID:           sdk.UUID(),
		VCSServerName:  "github",
		RepositoryName: "ovh/cds",
		Status:         sdk.HookEventStatusAnalysis,
		EventName:      sdk.WorkflowHookEventNamePush,
		Created:        time.Now().UnixNano(),
		ExtractData: sdk.HookRepositoryEventExtractData{
			Ref:    "refs/heads/master",
			Commit: "123456",
		},
		Analyses: []sdk.HookRepositoryEventAnalysis{
			{ProjectKey: "FAILED", AnalyzeID: sdk.UUID(), Status: sdk.RepositoryAnalysisStatusError, Error: "unable to retrieve files"},
		},
	}
	require.NoError(t, s.Dao.SaveRepositoryEvent(ctx, &hre))

	mock := s.Client.(*mock_cdsclient.MockInterface)
	mock.EXPECT().ListWorkflowToTrigger(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, req sdk.HookListWorkflowRequest) ([]sdk.V2WorkflowHook, error) {
		require.Empty(t, req.AnalyzedProjectKeys)
		return []sdk.V2WorkflowHook{
			{ProjectKey: "FAILED", VCSName: "github", RepositoryName: "ovh/cds", WorkflowName: "local", Type: sdk.WorkflowHookTypeRepository},
		}, nil
	}).Times(1)

	require.NoError(t, s.triggerAnalyses(ctx, &hre))

	require.Equal(t, sdk.HookEventStatusError, hre.Status)
	require.Equal(t, "repository analysis failed", hre.LastError)
	require.Len(t, hre.WorkflowHooks, 1)
	require.Equal(t, sdk.HookEventWorkflowStatusSkipped, hre.WorkflowHooks[0].Status)
	require.Equal(t, "repository analysis failed: unable to retrieve files", hre.WorkflowHooks[0].Error)
}
