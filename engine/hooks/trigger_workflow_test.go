package hooks

import (
	"context"
	"testing"

	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
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
