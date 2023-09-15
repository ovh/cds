package hooks

import (
	"context"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
)

const (
	RetryDelayMilli = 120000
)

// Get from queue task execution
func (s *Service) manageOldRepositoryEvent(ctx context.Context) {
	tick := time.NewTicker(1 * time.Minute).C

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Exiting manageOldRepositoryEvent: %v", ctx.Err())
			}
			return
		case <-tick:
			repositoryEventKeys, err := s.Dao.ListInProgressRepositoryEvent(ctx)
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
				continue
			}
			for _, k := range repositoryEventKeys {
				if err := s.checkInProgressEvent(ctx, k); err != nil {
					log.ErrorWithStackTrace(ctx, err)
					continue
				}
			}
		}
	}
}

func (s *Service) checkInProgressEvent(ctx context.Context, repoEventKey string) error {
	var repoEventTmp sdk.HookRepositoryEvent
	find, err := s.Cache.Get(repoEventKey, &repoEventTmp)
	if err != nil {
		return err
	}
	if !find {
		log.Info(ctx, "repository event %s does not exist anymore.", repoEventTmp.GetFullName())
		if err := s.Dao.RemoveRepositoryEventFromInProgressList(ctx, repoEventTmp); err != nil {
			return err
		}
	}

	b, err := s.Dao.LockRepositoryEvent(repoEventTmp.VCSServerType, repoEventTmp.VCSServerName, repoEventTmp.RepositoryName, repoEventTmp.UUID)
	if err != nil {
		return sdk.WrapError(err, "unable to lock repository event %s", repoEventTmp.GetFullName())
	}
	if !b {
		return nil
	}
	defer s.Dao.UnlockRepositoryEvent(repoEventTmp.VCSServerType, repoEventTmp.VCSServerName, repoEventTmp.RepositoryName, repoEventTmp.UUID)

	var hre sdk.HookRepositoryEvent
	find, err = s.Cache.Get(repoEventKey, &hre)
	if err != nil {
		return sdk.WrapError(err, "unable to retrieve repository event")
	}
	if !find {
		log.Info(ctx, "repository event %s does not exist anymore.", hre.GetFullName())
		if err := s.Dao.RemoveRepositoryEventFromInProgressList(ctx, hre); err != nil {
			return err
		}
		return nil
	}

	// Check last update time
	if time.Now().UnixMilli()-hre.LastUpdate > RetryDelayMilli {
		log.Info(ctx, "re-enqueue event %s", hre.GetFullName())
		if err := s.Dao.EnqueueRepositoryEvent(ctx, &hre); err != nil {
			return err
		}
	}
	return nil
}
