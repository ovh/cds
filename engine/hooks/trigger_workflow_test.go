package hooks

import (
	"context"
	"testing"

	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
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

func TestTriggerWorkflow_EntityOwnerIdentifiesTheUser(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	ctx := context.TODO()

	newEvent := func(owner *sdk.V2Initiator) sdk.HookRepositoryEvent {
		return sdk.HookRepositoryEvent{
			UUID:           sdk.UUID(),
			VCSServerName:  "github",
			RepositoryName: "ovh/cds",
			EventName:      sdk.WorkflowHookEventNameScheduler,
			ExtractData: sdk.HookRepositoryEventExtractData{
				Ref:    "refs/heads/master",
				Commit: "123456",
				Scheduler: &sdk.HookRepositoryEventExtractedDataScheduler{
					TargetVCS:      "github",
					TargetRepo:     "ovh/cds",
					TargetProject:  "PROJ",
					TargetWorkflow: "myworkflow",
					Cron:           "* * * * *",
					Timezone:       "UTC",
				},
			},
			WorkflowHooks: []sdk.HookRepositoryEventWorkflow{
				{
					Type:                 sdk.WorkflowHookTypeScheduler,
					Status:               sdk.HookEventWorkflowStatusScheduled,
					ProjectKey:           "PROJ",
					VCSIdentifier:        "github",
					RepositoryIdentifier: "ovh/cds",
					WorkflowName:         "myworkflow",
					Ref:                  "refs/heads/master",
					Commit:               "123456",
					TargetCommit:         "123456",
					Initiator:            owner,
				},
			},
		}
	}
	mock := s.Client.(*mock_cdsclient.MockInterface)

	// A VCS owner is a known user: the run is requested on its behalf
	vcsOwner := &sdk.V2Initiator{VCS: "github", VCSUsername: "octocat"}
	var sent sdk.V2WorkflowRunHookRequest
	mock.EXPECT().WorkflowV2RunFromHook(gomock.Any(), "PROJ", "github", "ovh/cds", "myworkflow", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _, _, _ string, runRequest sdk.V2WorkflowRunHookRequest, _ ...cdsclient.RequestModifier) (*sdk.V2WorkflowRun, error) {
			sent = runRequest
			return &sdk.V2WorkflowRun{ID: "run-id", RunNumber: 1}, nil
		}).Times(1)
	hre := newEvent(vcsOwner)
	require.NoError(t, s.triggerWorkflows(ctx, &hre))
	require.Equal(t, sdk.HookEventWorkflowStatusDone, hre.WorkflowHooks[0].Status)
	require.Equal(t, vcsOwner, sent.Initiator)
	require.Empty(t, sent.DeprecatedUserID)

	// Nobody identified: skipped, whether the owner is absent or empty
	for _, owner := range []*sdk.V2Initiator{nil, {}} {
		hre := newEvent(owner)
		require.NoError(t, s.triggerWorkflows(ctx, &hre))
		require.Equal(t, sdk.HookEventWorkflowStatusSkipped, hre.WorkflowHooks[0].Status)
		require.Equal(t, "unknown user", hre.WorkflowHooks[0].Error)
	}
}
