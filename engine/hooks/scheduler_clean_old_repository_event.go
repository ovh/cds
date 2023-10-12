package hooks

import (
	"context"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

// Get from queue task execution
func (s *Service) scheduleCleanOldRepositoryEvent(ctx context.Context) {
	tick := time.NewTicker(1 * time.Hour).C

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Exiting cleanOldRepositoryEvent: %v", ctx.Err())
			}
			return
		case <-tick:
			repos, err := s.Dao.ListRepositories(ctx, "")
			if err != nil {

			}
			for _, r := range repos {
				if err := s.cleanRepositoryEvent(ctx, r); err != nil {
					log.ErrorWithStackTrace(ctx, err)
				}
			}
		}
	}
}

func (s *Service) cleanRepositoryEvent(ctx context.Context, repoKey string) error {
	lockKey := cache.Key(repositoryLock, repoKey)
	b, err := s.Dao.store.Lock(lockKey, time.Minute, 100, 1)
	if err != nil {
		return err
	}
	if !b {
		return nil
	}
	defer s.Dao.store.Unlock(lockKey)

	var hookRepo sdk.HookRepository
	exist, err := s.Dao.store.Get(cache.Key(repositoryRootKey, repoKey), &hookRepo)
	if err != nil {
		return err
	}
	if !exist {
		return nil
	}

	events, err := s.Dao.ListRepositoryEvents(ctx, hookRepo.VCSServerName, hookRepo.RepositoryName)
	if err != nil {
		return err
	}
	for len(events) > s.Cfg.RepositoryEventRetention {
		var repoEvent sdk.HookRepositoryEvent
		repoEvent, events = events[0], events[1:]
		if err := s.Dao.DeleteRepositoryEvent(ctx, repoEvent.VCSServerName, repoEvent.RepositoryName, repoEvent.UUID); err != nil {
			return err
		}
	}
	return nil
}
