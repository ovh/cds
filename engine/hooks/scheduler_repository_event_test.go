package hooks

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient/mock_cdsclient"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestManageAnalysisCallback(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	s, cancel := setupTestHookService(t)
	defer cancel()

	event := GiteaEventPayload{}
	event.Repository.FullName = "ovh/cds"
	event.Ref = "master"
	event.After = "123456"

	bts, _ := json.Marshal(event)

	// Create event
	hr := sdk.HookRepositoryEvent{
		UUID:           sdk.UUID(),
		VCSServerName:  "private-github",
		RepositoryName: "ovh/cds",
		Status:         sdk.HookEventStatusAnalysis,
		EventName:      sdk.WorkflowHookEventNamePush,
		Created:        time.Now().UnixNano(),
		Body:           bts,
		Analyses: []sdk.HookRepositoryEventAnalysis{
			{
				ProjectKey: "MYPROJECT",
				Status:     sdk.RepositoryAnalysisStatusInProgress,
				AnalyzeID:  sdk.UUID(),
			},
		},
	}
	require.NoError(t, s.Dao.SaveRepositoryEvent(context.TODO(), &hr))

	// Create repo
	_, err := s.Dao.CreateRepository(context.TODO(), hr.VCSServerName, hr.RepositoryName)
	require.NoError(t, err)

	eventKey := strings.ToLower(cache.Key(repositoryEventRootKey, s.Dao.GetRepositoryMemberKey(hr.VCSServerName, hr.RepositoryName), hr.UUID))
	callback := sdk.HookEventCallback{
		RepositoryName: hr.RepositoryName,
		VCSServerName:  hr.VCSServerName,
		HookEventUUID:  hr.UUID,
		HookEventKey:   eventKey,
		AnalysisCallback: &sdk.HookAnalysisCallback{
			AnalysisID:     hr.Analyses[0].AnalyzeID,
			AnalysisStatus: sdk.RepositoryAnalysisStatusSucceed,
			Initiator:      &sdk.V2Initiator{},
		},
	}

	require.NoError(t, s.updateHookEventWithCallback(context.TODO(), callback))

	k := cache.Key(repositoryEventRootKey, s.Dao.GetRepositoryMemberKey(hr.VCSServerName, hr.RepositoryName), hr.UUID)
	var hreUpdate sdk.HookRepositoryEvent
	f, err := s.Cache.Get(k, &hreUpdate)
	require.NoError(t, err)
	require.True(t, f)
	require.Equal(t, sdk.RepositoryAnalysisStatusSucceed, hreUpdate.Analyses[0].Status)

}

func TestManageRepositoryEvent_PushEventTriggerAnalysis(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	s, cancel := setupTestHookService(t)
	defer cancel()

	event := GiteaEventPayload{}
	event.Repository.FullName = "ovh/cds"
	event.Ref = "master"
	event.After = "123456"

	bts, _ := json.Marshal(event)

	// Create event
	hr := sdk.HookRepositoryEvent{
		UUID:           sdk.UUID(),
		VCSServerName:  "private-github",
		RepositoryName: "ovh/cds",
		Status:         sdk.HookEventStatusScheduled,
		EventName:      sdk.WorkflowHookEventNamePush,
		Created:        time.Now().UnixNano(),
		Body:           bts,
		ExtractData: sdk.HookRepositoryEventExtractData{
			Ref:    "refs/heads/master",
			Commit: "123456",
		},
	}
	require.NoError(t, s.Dao.SaveRepositoryEvent(context.TODO(), &hr))

	// Create repo
	_, err := s.Dao.CreateRepository(context.TODO(), hr.VCSServerName, hr.RepositoryName)
	require.NoError(t, err)

	s.Client.(*mock_cdsclient.MockInterface).EXPECT().CreateInsightReport(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	s.Client.(*mock_cdsclient.MockInterface).EXPECT().HookRepositoriesList(gomock.Any(), hr.VCSServerName, hr.RepositoryName).Return([]sdk.ProjectRepository{
		{
			Name:       hr.RepositoryName,
			ProjectKey: "TEST",
		},
	}, nil).Times(1)
	s.Client.(*mock_cdsclient.MockInterface).EXPECT().ProjectRepositoryAnalysis(gomock.Any(), gomock.Any()).Times(1)

	// Force dequeue
	k := cache.Key(repositoryEventRootKey, s.Dao.GetRepositoryMemberKey(hr.VCSServerName, hr.RepositoryName), hr.UUID)
	require.NoError(t, s.manageRepositoryEvent(context.TODO(), k))
}

func TestManageRepositoryEvent_NonPushEventWorkflowToTrigger(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	s, cancel := setupTestHookService(t)
	defer cancel()

	event := GiteaEventPayload{}
	event.Repository.FullName = "ovh/cds"
	event.Ref = "master"
	event.After = "123456"

	bts, _ := json.Marshal(event)

	// Create event
	hr := sdk.HookRepositoryEvent{
		UUID:           sdk.UUID(),
		VCSServerName:  "private-github",
		RepositoryName: "ovh/cds",
		Status:         sdk.HookEventStatusScheduled,
		EventName:      sdk.WorkflowHookEventNamePullRequest,
		Created:        time.Now().UnixNano(),
		Body:           bts,
		SignKey:        "AZERTY",
	}
	require.NoError(t, s.Dao.SaveRepositoryEvent(context.TODO(), &hr))

	// Create repo
	_, err := s.Dao.CreateRepository(context.TODO(), hr.VCSServerName, hr.RepositoryName)
	require.NoError(t, err)

	s.Client.(*mock_cdsclient.MockInterface).EXPECT().HookRepositoriesList(gomock.Any(), gomock.Any(), gomock.Any()).Return([]sdk.ProjectRepository{
		{
			ProjectKey: "PROJ",
		},
	}, nil)

	s.Client.(*mock_cdsclient.MockInterface).EXPECT().ProjectRepositoryAnalysisList(gomock.Any(), "PROJ", "private-github", "ovh/cds").Return([]sdk.ProjectRepositoryAnalysis{
		{
			ID:     "123456",
			Status: sdk.RepositoryAnalysisStatusSucceed,
		},
	}, nil)

	s.Client.(*mock_cdsclient.MockInterface).EXPECT().CreateInsightReport(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	s.Client.(*mock_cdsclient.MockInterface).EXPECT().ListWorkflowToTrigger(gomock.Any(), gomock.Any()).Return([]sdk.V2WorkflowHook{
		{
			ProjectKey:     "PROJ",
			VCSName:        "github",
			RepositoryName: "repo",
			WorkflowName:   "myworkflow",
		},
	}, nil)

	s.Client.(*mock_cdsclient.MockInterface).EXPECT().RetrieveHookEventSigningKey(gomock.Any(), gomock.Any()).Times(1)
	userID := "aaa-bbb-ccc"
	s.Client.(*mock_cdsclient.MockInterface).EXPECT().EntityGet(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&sdk.Entity{
		UserID: &userID,
	}, nil).Times(1)

	// Force dequeue
	k := cache.Key(repositoryEventRootKey, s.Dao.GetRepositoryMemberKey(hr.VCSServerName, hr.RepositoryName), hr.UUID)
	require.NoError(t, s.manageRepositoryEvent(context.TODO(), k))
}

// TestExecuteEventRequeuesItselfWhenItAdvances covers the reason the re-queue
// exists. One pass moves the event by a single state, and before this nothing
// put it back on the queue, so every transition waited for the periodic sweep —
// minutes per state for work that takes seconds.
func TestExecuteEventRequeuesItselfWhenItAdvances(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	s, cancel := setupTestHookService(t)
	defer cancel()

	event := GiteaEventPayload{}
	event.Repository.FullName = "ovh/cds"
	event.Ref = "master"
	event.After = "123456"
	bts, _ := json.Marshal(event)

	hr := sdk.HookRepositoryEvent{
		UUID:           sdk.UUID(),
		VCSServerName:  "private-github",
		RepositoryName: "ovh/cds",
		Status:         sdk.HookEventStatusScheduled,
		EventName:      sdk.WorkflowHookEventNamePush,
		Created:        time.Now().UnixNano(),
		Body:           bts,
		ExtractData: sdk.HookRepositoryEventExtractData{
			Ref:    "refs/heads/master",
			Commit: "123456",
		},
	}
	require.NoError(t, s.Dao.SaveRepositoryEvent(context.TODO(), &hr))
	_, err := s.Dao.CreateRepository(context.TODO(), hr.VCSServerName, hr.RepositoryName)
	require.NoError(t, err)

	s.Client.(*mock_cdsclient.MockInterface).EXPECT().CreateInsightReport(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	s.Client.(*mock_cdsclient.MockInterface).EXPECT().HookRepositoriesList(gomock.Any(), hr.VCSServerName, hr.RepositoryName).Return([]sdk.ProjectRepository{
		{Name: hr.RepositoryName, ProjectKey: "TEST"},
	}, nil).Times(1)
	s.Client.(*mock_cdsclient.MockInterface).EXPECT().ProjectRepositoryAnalysis(gomock.Any(), gomock.Any()).Times(1)

	// Relative, not absolute: the queue is shared and other work may sit in it.
	before, err := s.Dao.RepositoryEventQueueLen()
	require.NoError(t, err)

	require.NoError(t, s.executeEvent(context.TODO(), &hr))

	require.Equal(t, sdk.HookEventStatusAnalysis, hr.Status, "the pass must have advanced the event")
	after, err := s.Dao.RepositoryEventQueueLen()
	require.NoError(t, err)
	require.Equal(t, before+1, after, "an event that advanced must be queued again so its next state runs now rather than at the next sweep")
}

// TestExecuteEventDoesNotRequeueWhileWaiting covers the other half of the
// condition, which is the half that carries the risk. An event that did not
// advance is waiting on something it does not own — here an analysis still
// running — and re-queueing it would spin the worker instead of waiting for the
// callback that resolves it. Measured worse when we tried the unconditional
// form: the dequeue loop is serial, so parked events crowd out real work.
func TestExecuteEventDoesNotRequeueWhileWaiting(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	s, cancel := setupTestHookService(t)
	defer cancel()

	event := GiteaEventPayload{}
	event.Repository.FullName = "ovh/cds"
	event.Ref = "master"
	event.After = "123456"
	bts, _ := json.Marshal(event)

	hr := sdk.HookRepositoryEvent{
		UUID:           sdk.UUID(),
		VCSServerName:  "private-github",
		RepositoryName: "ovh/cds",
		Status:         sdk.HookEventStatusAnalysis,
		EventName:      sdk.WorkflowHookEventNamePush,
		Created:        time.Now().UnixNano(),
		// Recent, so the analysis is genuinely considered still in flight
		// rather than due for a status re-check.
		LastUpdate: time.Now().UnixMilli(),
		Body:       bts,
		Analyses: []sdk.HookRepositoryEventAnalysis{
			{
				ProjectKey: "TEST",
				Status:     sdk.RepositoryAnalysisStatusInProgress,
				AnalyzeID:  sdk.UUID(),
			},
		},
	}
	require.NoError(t, s.Dao.SaveRepositoryEvent(context.TODO(), &hr))
	_, err := s.Dao.CreateRepository(context.TODO(), hr.VCSServerName, hr.RepositoryName)
	require.NoError(t, err)

	s.Client.(*mock_cdsclient.MockInterface).EXPECT().CreateInsightReport(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	before, err := s.Dao.RepositoryEventQueueLen()
	require.NoError(t, err)

	require.NoError(t, s.executeEvent(context.TODO(), &hr))

	require.Equal(t, sdk.HookEventStatusAnalysis, hr.Status, "the event must not have advanced")
	after, err := s.Dao.RepositoryEventQueueLen()
	require.NoError(t, err)
	require.Equal(t, before, after, "an event still waiting must be left to the periodic sweep, not re-queued into a spin")
}

// TestExecuteEventDoesNotRequeueTerminalEvents guards the other exit: a
// finished event must leave the queue rather than be put back on it.
func TestExecuteEventDoesNotRequeueTerminalEvents(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	s, cancel := setupTestHookService(t)
	defer cancel()

	event := GiteaEventPayload{}
	event.Repository.FullName = "ovh/cds"
	event.Ref = "master"
	event.After = "123456"
	bts, _ := json.Marshal(event)

	hr := sdk.HookRepositoryEvent{
		UUID:           sdk.UUID(),
		VCSServerName:  "private-github",
		RepositoryName: "ovh/cds",
		Status:         sdk.HookEventStatusDone,
		EventName:      sdk.WorkflowHookEventNamePush,
		Created:        time.Now().UnixNano(),
		Body:           bts,
	}
	require.NoError(t, s.Dao.SaveRepositoryEvent(context.TODO(), &hr))
	_, err := s.Dao.CreateRepository(context.TODO(), hr.VCSServerName, hr.RepositoryName)
	require.NoError(t, err)

	s.Client.(*mock_cdsclient.MockInterface).EXPECT().CreateInsightReport(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	before, err := s.Dao.RepositoryEventQueueLen()
	require.NoError(t, err)

	require.NoError(t, s.executeEvent(context.TODO(), &hr))

	after, err := s.Dao.RepositoryEventQueueLen()
	require.NoError(t, err)
	require.Equal(t, before, after, "a finished event must not be put back on the queue")
}

// TestIsTerminalHookEventStatus pins which states end an event. Getting this
// wrong in either direction is silent: a terminal state treated as
// intermediate is re-queued forever, an intermediate state treated as terminal
// stalls exactly like the bug this change fixes.
func TestIsTerminalHookEventStatus(t *testing.T) {
	require.True(t, isTerminalHookEventStatus(sdk.HookEventStatusDone))
	require.True(t, isTerminalHookEventStatus(sdk.HookEventStatusSkipped))
	require.True(t, isTerminalHookEventStatus(sdk.HookEventStatusError))

	require.False(t, isTerminalHookEventStatus(sdk.HookEventStatusScheduled))
	require.False(t, isTerminalHookEventStatus(sdk.HookEventStatusAnalysis))
	require.False(t, isTerminalHookEventStatus(sdk.HookEventStatusCheckAnalysis))
	require.False(t, isTerminalHookEventStatus(sdk.HookEventStatusWorkflowHooks))
	require.False(t, isTerminalHookEventStatus(sdk.HookEventStatusGitInfo))
	require.False(t, isTerminalHookEventStatus(sdk.HookEventStatusWorkflow))
}
