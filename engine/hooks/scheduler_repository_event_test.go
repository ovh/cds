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

	// Force dequeue
	k := cache.Key(repositoryEventRootKey, s.Dao.GetRepositoryMemberKey(hr.VCSServerName, hr.RepositoryName), hr.UUID)
	require.NoError(t, s.manageRepositoryEvent(context.TODO(), k))
}
