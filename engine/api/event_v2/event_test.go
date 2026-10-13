package event_v2

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestVCSBuildStatus(t *testing.T) {
	require.Equal(t, sdk.StatusBuilding, vcsBuildStatus(sdk.V2WorkflowRunStatusCrafting))
	require.Equal(t, sdk.StatusBuilding, vcsBuildStatus(sdk.V2WorkflowRunStatusBlocked))
	require.Equal(t, sdk.StatusBuilding, vcsBuildStatus(sdk.V2WorkflowRunStatusBuilding))
	require.Equal(t, sdk.StatusSuccess, vcsBuildStatus(sdk.V2WorkflowRunStatusSuccess))
	require.Equal(t, sdk.StatusFail, vcsBuildStatus(sdk.V2WorkflowRunStatusFail))
	require.Equal(t, sdk.StatusStopped, vcsBuildStatus(sdk.V2WorkflowRunStatusStopped))
	require.Equal(t, sdk.StatusSkipped, vcsBuildStatus(sdk.V2WorkflowRunStatusSkipped))
	require.Equal(t, sdk.StatusCancelled, vcsBuildStatus(sdk.V2WorkflowRunStatusCancelled))
}

func TestWorkflowNotificationsBuildStatus(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
	ctx := context.TODO()

	proj := assets.InsertTestProject(t, db, cache, sdk.RandomString(10), sdk.RandomString(10))
	_ = assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")

	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	t.Cleanup(func() {
		_ = services.Delete(db, s)
		services.NewClient = services.NewDefaultClient
	})

	var sent []sdk.VCSBuildStatus
	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "POST", "/vcs/github/status", gomock.Any(), gomock.Any(), gomock.Any()).Do(func(ctx, method, path, in, out interface{}, mods ...interface{}) {
		sent = append(sent, in.(sdk.VCSBuildStatus))
	}).AnyTimes()

	newEvent := func(eventType sdk.EventType, status sdk.V2WorkflowRunStatus) sdk.FullEventV2 {
		payload, err := json.Marshal(sdk.EventWorkflowRunPayload{
			WorkflowName: "myWorkflow",
			Status:       status,
			Contexts: sdk.EventWorkflowRunPayloadContexts{
				Git: sdk.GitContext{Server: "github", Repository: "my/repo", Sha: "abcdef"},
			},
		})
		require.NoError(t, err)
		return sdk.FullEventV2{
			Type:       eventType,
			Payload:    payload,
			ProjectKey: proj.Key,
			VCSName:    "github",
			Repository: "my/repo",
			Status:     string(status),
		}
	}

	// A run blocked by its concurrency is in progress for the VCS
	require.NoError(t, workflowNotifications(ctx, db.DbMap, cache, newEvent(sdk.EventRunBuilding, sdk.V2WorkflowRunStatusBlocked), "http://ui"))
	// A restart resets the status left by the previous attempt
	require.NoError(t, workflowNotifications(ctx, db.DbMap, cache, newEvent(sdk.EventRunRestart, sdk.V2WorkflowRunStatusBuilding), "http://ui"))
	require.NoError(t, workflowNotifications(ctx, db.DbMap, cache, newEvent(sdk.EventRunEnded, sdk.V2WorkflowRunStatusFail), "http://ui"))
	// Other run events do not reach the VCS
	require.NoError(t, workflowNotifications(ctx, db.DbMap, cache, newEvent(sdk.EventRunUpdated, sdk.V2WorkflowRunStatusBuilding), "http://ui"))

	require.Len(t, sent, 3)
	require.Equal(t, sdk.StatusBuilding, sent[0].Status)
	require.Equal(t, sdk.StatusBuilding, sent[1].Status)
	require.Equal(t, sdk.StatusFail, sent[2].Status)
}

func TestWorkflowEventInterpolateAndTemplating(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
	ctx := context.TODO()

	proj := assets.InsertTestProject(t, db, cache, sdk.RandomString(10), sdk.RandomString(10))
	_ = assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")

	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	t.Cleanup(func() {
		_ = services.Delete(db, s)
		services.NewClient = services.NewDefaultClient
	})

	var captureComment sdk.VCSPullRequestCommentRequest
	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/my/repo/pullrequests/1", nil, gomock.Any(), gomock.Any())
	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "POST", "/vcs/github/repos/my/repo/pullrequests/comments", gomock.Any(), gomock.Any(), gomock.Any()).Do(func(ctx, method, path, in, out interface{}, mods ...interface{}) {
		captureComment = in.(sdk.VCSPullRequestCommentRequest)
	})

	eventPayload := sdk.EventWorkflowRunPayload{
		WorkflowName: "myWorkflow",
		Contexts: sdk.EventWorkflowRunPayloadContexts{
			CDS: sdk.CDSContext{
				Workflow: "myWorkflow",
			},
			Git: sdk.GitContext{
				PullRequestID: 1,
				Server:        "github",
				Repository:    "my/repo",
			},
			Jobs: map[string]sdk.JobResultContext{
				"job1": {
					Result: "Success",
				},
			},
		},
	}
	comment := `[[- if eq .cds.workflow "myWorkflow"]]I'm workflow [[.cds.workflow ]][[- end]]`
	vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, db, cache, proj.Key, "github")
	require.NoError(t, err)

	require.NoError(t, sendVCSPullRequestComment(ctx, db.DbMap, vcsClient, eventPayload, comment))
	require.Equal(t, "I'm workflow myWorkflow", captureComment.Message)
}

func TestWorkflowEventErrorTempalte(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
	ctx := context.TODO()

	proj := assets.InsertTestProject(t, db, cache, sdk.RandomString(10), sdk.RandomString(10))
	_ = assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")

	wr := sdk.V2WorkflowRun{
		ID:           sdk.UUID(),
		VCSServerID:  sdk.UUID(),
		RepositoryID: sdk.UUID(),
		ProjectKey:   proj.Key,
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	t.Cleanup(func() {
		_ = services.Delete(db, s)
		services.NewClient = services.NewDefaultClient
	})

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/my/repo/pullrequests/1", nil, gomock.Any(), gomock.Any())

	eventPayload := sdk.EventWorkflowRunPayload{
		WorkflowName: "myWorkflow",
		ID:           wr.ID,
		Contexts: sdk.EventWorkflowRunPayloadContexts{
			CDS: sdk.CDSContext{
				Workflow: "myWorkflow",
			},
			Git: sdk.GitContext{
				PullRequestID: 1,
				Server:        "github",
				Repository:    "my/repo",
			},
			Jobs: map[string]sdk.JobResultContext{
				"job1": {
					Result: "Success",
				},
			},
		},
	}
	comment := `[[-if eq .WorkflowName "myWorkflow"]]`
	vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, db, cache, proj.Key, "github")
	require.NoError(t, err)

	require.Error(t, sendVCSPullRequestComment(ctx, db.DbMap, vcsClient, eventPayload, comment))

	runInfos, err := workflow_v2.LoadRunInfosByRunID(ctx, db, eventPayload.ID)
	require.NoError(t, err)

	require.Equal(t, 1, len(runInfos))
	require.Contains(t, runInfos[0].Message, "bad number syntax: \"-if\"")
}
