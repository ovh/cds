package hooks

import (
	"context"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCleanRepositoryEvent(t *testing.T) {
	log.Factory = log.NewTestingWrapper(t)
	s, cancel := setupTestHookService(t)
	defer cancel()

	repo, err := s.Dao.CreateRepository(context.TODO(), "github", "mypvtgithub", "ovh/myrepo")
	require.NoError(t, err)

	require.NoError(t, s.Dao.DeleteAllRepositoryEvent(context.TODO(), repo.VCSServerName, repo.RepositoryName))

	for i := 0; i < 100; i++ {
		hre := sdk.HookRepositoryEvent{
			UUID:           sdk.UUID(),
			VCSServerName:  repo.VCSServerName,
			RepositoryName: repo.RepositoryName,
		}
		require.NoError(t, s.Dao.SaveRepositoryEvent(context.TODO(), &hre))
	}
	events, err := s.Dao.ListRepositoryEvents(context.TODO(), repo.VCSServerName, repo.RepositoryName)
	require.NoError(t, err)
	require.Equal(t, 100, len(events))

	s.Cfg.RepositoryEventRetention = 30
	require.NoError(t, s.cleanRepositoryEvent(context.TODO(), repo.VCSServerName+"-"+repo.RepositoryName))

	eventsAfter, err := s.Dao.ListRepositoryEvents(context.TODO(), repo.VCSServerName, repo.RepositoryName)
	require.NoError(t, err)
	require.Equal(t, 30, len(eventsAfter))
}
