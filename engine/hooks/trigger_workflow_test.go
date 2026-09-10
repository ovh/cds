package hooks

import (
	"context"
	"testing"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestTriggerWorkflow(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	s, cancel := setupTestHookService(t)
	defer cancel()

	ctx := context.TODO()
	hre := sdk.HookRepositoryEvent{
		Initiator: &sdk.V2Initiator{
			UserID: "1234567890",
		},
		ExtractData: sdk.HookRepositoryEventExtractData{
			Paths: []string{"src/main/main.test", "src/resources/readme.md"},
		},
		Body: []byte(`{"foo": "bar"}`),
		WorkflowHooks: []sdk.HookRepositoryEventWorkflow{
			{
				Status: sdk.HookEventWorkflowStatusScheduled,
				PathFilters: []string{
					"src/main/**/*.java",
				},
			},
		},
	}

	require.NoError(t, s.triggerWorkflows(ctx, &hre))
	require.Equal(t, sdk.HookEventWorkflowStatusSkipped, hre.WorkflowHooks[0].Status)
	require.Equal(t, "no file matches path filters", hre.WorkflowHooks[0].Error)
}

func TestTriggerWorkflow_WorkflowRunPathFilterOnParentChangeSets(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	s, cancel := setupTestHookService(t)
	defer cancel()

	ctx := context.TODO()
	hre := sdk.HookRepositoryEvent{
		UUID:      sdk.UUID(),
		EventName: sdk.WorkflowHookEventNameWorkflowRun,
		Initiator: &sdk.V2Initiator{UserID: "1234567890"},
		ExtractData: sdk.HookRepositoryEventExtractData{
			WorkflowRun: &sdk.HookRepositoryEventExtractedDataWorkflowRun{
				Workflow:      "WorkflowFrom",
				WorkflowRunID: "run-id",
			},
		},
		Body: []byte(`{"foo": "bar"}`),
		WorkflowHooks: []sdk.HookRepositoryEventWorkflow{
			{
				Type:               sdk.WorkflowHookTypeWorkflowRun,
				Status:             sdk.HookEventWorkflowStatusScheduled,
				PathFilters:        []string{"src/main/**/*.java"},
				ParentUpdatedFiles: []string{"src/main/foo/Bar.java"},
				// Target repository changesets must not be used by the path filter
				UpdatedFiles: []string{"README.md"},
			},
		},
	}

	var runRequest sdk.V2WorkflowRunHookRequest
	s.Client.(*mock_cdsclient.MockInterface).EXPECT().
		WorkflowV2RunFromHook(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _, _, _ string, req sdk.V2WorkflowRunHookRequest, _ ...cdsclient.RequestModifier) (*sdk.V2WorkflowRun, error) {
			runRequest = req
			return &sdk.V2WorkflowRun{ID: "child-run-id", RunNumber: 1}, nil
		}).Times(1)

	require.NoError(t, s.triggerWorkflows(ctx, &hre))
	require.Equal(t, sdk.HookEventWorkflowStatusDone, hre.WorkflowHooks[0].Status)
	require.Equal(t, "child-run-id", hre.WorkflowHooks[0].RunID)
	require.Equal(t, sdk.HookEventStatusDone, hre.Status)
	// The child run keeps the target repository changesets, not the parent ones
	require.Equal(t, []string{"README.md"}, runRequest.ChangeSets)
	require.Equal(t, "WorkflowFrom", runRequest.WorkflowRun)
	require.Equal(t, "run-id", runRequest.WorkflowRunID)
}

func TestTriggerWorkflow_WorkflowRunPathFilterSkipsWhenParentChangeSetsDoNotMatch(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	s, cancel := setupTestHookService(t)
	defer cancel()

	ctx := context.TODO()
	hre := sdk.HookRepositoryEvent{
		UUID:      sdk.UUID(),
		EventName: sdk.WorkflowHookEventNameWorkflowRun,
		Initiator: &sdk.V2Initiator{UserID: "1234567890"},
		ExtractData: sdk.HookRepositoryEventExtractData{
			WorkflowRun: &sdk.HookRepositoryEventExtractedDataWorkflowRun{Workflow: "WorkflowFrom"},
		},
		Body: []byte(`{"foo": "bar"}`),
		WorkflowHooks: []sdk.HookRepositoryEventWorkflow{
			{
				Type:               sdk.WorkflowHookTypeWorkflowRun,
				Status:             sdk.HookEventWorkflowStatusScheduled,
				PathFilters:        []string{"src/main/**/*.java"},
				ParentUpdatedFiles: []string{"docs/readme.md"},
				// Matching target repository changesets must not trigger the workflow
				UpdatedFiles: []string{"src/main/foo/Bar.java"},
			},
			{
				// Parent run without changesets (manual, scheduled or tag run) is skipped too
				Type:        sdk.WorkflowHookTypeWorkflowRun,
				Status:      sdk.HookEventWorkflowStatusScheduled,
				PathFilters: []string{"src/main/**/*.java"},
			},
		},
	}

	// No WorkflowV2RunFromHook expectation: any run attempt fails the test
	require.NoError(t, s.triggerWorkflows(ctx, &hre))
	for _, wh := range hre.WorkflowHooks {
		require.Equal(t, sdk.HookEventWorkflowStatusSkipped, wh.Status)
		require.Equal(t, "no file matches path filters", wh.Error)
	}
	require.Equal(t, sdk.HookEventStatusDone, hre.Status)
}

func TestSkipNonMatchingPullRequestCommentHooks(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	ctx := context.TODO()

	hre := &sdk.HookRepositoryEvent{
		EventName: sdk.WorkflowHookEventNamePullRequestComment,
		ExtractData: sdk.HookRepositoryEventExtractData{
			Comment: "deploy the app",
		},
		WorkflowHooks: []sdk.HookRepositoryEventWorkflow{
			{Status: sdk.HookEventWorkflowStatusScheduled, Data: sdk.V2WorkflowHookData{CommentFilter: "deploy*"}},
			{Status: sdk.HookEventWorkflowStatusScheduled, Data: sdk.V2WorkflowHookData{CommentFilter: "release*"}},
			{Status: sdk.HookEventWorkflowStatusScheduled},
		},
	}

	skipNonMatchingPullRequestCommentHooks(ctx, hre)

	// Matching filter stays scheduled
	require.Equal(t, sdk.HookEventWorkflowStatusScheduled, hre.WorkflowHooks[0].Status)
	// Non-matching filter is skipped with a reason
	require.Equal(t, sdk.HookEventWorkflowStatusSkipped, hre.WorkflowHooks[1].Status)
	require.Equal(t, "comment does not match comment filter", hre.WorkflowHooks[1].Error)
	// No filter stays scheduled
	require.Equal(t, sdk.HookEventWorkflowStatusScheduled, hre.WorkflowHooks[2].Status)
}
