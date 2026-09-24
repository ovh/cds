package hooks

import (
	"context"
	"testing"

	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
)

func TestEntityOwner(t *testing.T) {
	userID := "user-id"
	vcsOwner := &sdk.V2Initiator{VCS: "github", VCSUsername: "octocat"}
	cdsOwner := &sdk.V2Initiator{UserID: userID, User: &sdk.V2InitiatorUser{Username: "alice"}}

	require.Same(t, vcsOwner, entityOwner(&sdk.Entity{Initiator: vcsOwner}))
	require.Same(t, cdsOwner, entityOwner(&sdk.Entity{Initiator: cdsOwner, DeprecatedUserID: &userID}))

	// Nobody identified: the owner is nil so the event initiator keeps precedence
	require.Nil(t, entityOwner(&sdk.Entity{Initiator: &sdk.V2Initiator{}, DeprecatedUserID: &userID}))
	require.Nil(t, entityOwner(&sdk.Entity{}))
	empty := ""
	require.Nil(t, entityOwner(&sdk.Entity{DeprecatedUserID: &empty}))

	// No initiator sent, only user_id: the owner is rebuilt with an empty snapshot, safe for Username()
	rebuilt := entityOwner(&sdk.Entity{DeprecatedUserID: &userID})
	require.Equal(t, &sdk.V2Initiator{UserID: userID, User: &sdk.V2InitiatorUser{}}, rebuilt)
	require.Empty(t, rebuilt.Username())
}

func TestHandleScheduler_ForwardsTheEntityOwner(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	s, cancel := setupTestHookService(t)
	defer cancel()
	ctx := context.TODO()

	newEvent := func() sdk.HookRepositoryEvent {
		return sdk.HookRepositoryEvent{
			UUID:           sdk.UUID(),
			VCSServerName:  "github",
			RepositoryName: "ovh/cds",
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
		}
	}
	mock := s.Client.(*mock_cdsclient.MockInterface)

	vcsOwner := &sdk.V2Initiator{VCS: "github", VCSUsername: "octocat"}
	mock.EXPECT().EntityGet(gomock.Any(), "PROJ", "github", "ovh/cds", sdk.EntityTypeWorkflow, "myworkflow").
		Return(&sdk.Entity{Initiator: vcsOwner}, nil).Times(1)
	hre := newEvent()
	require.NoError(t, s.handleScheduler(ctx, &hre))
	require.Len(t, hre.WorkflowHooks, 1)
	require.Equal(t, vcsOwner, hre.WorkflowHooks[0].Initiator)

	mock.EXPECT().EntityGet(gomock.Any(), "PROJ", "github", "ovh/cds", sdk.EntityTypeWorkflow, "myworkflow").
		Return(&sdk.Entity{Initiator: &sdk.V2Initiator{}}, nil).Times(1)
	hre = newEvent()
	require.NoError(t, s.handleScheduler(ctx, &hre))
	require.Nil(t, hre.WorkflowHooks[0].Initiator)
}
