package hooks

import (
	"context"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func (s *Service) dequeueRepositoryEventCallback(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		size, err := s.Dao.RepositoryEventCallbackQueueLen()
		if err != nil {
			log.Error(ctx, "dequeueRepositoryEventCallback > Unable to get queueLen: %v", err)
			continue
		}
		log.Debug(ctx, "dequeueRepositoryEventCallback> current queue size: %d", size)

		// Dequeuing context
		var callback sdk.HookAnalysisCallback
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Get next EventKEY
		if err := s.Cache.DequeueWithContext(ctx, repositoryEventCallbackQueue, 250*time.Millisecond, &callback); err != nil {
			continue
		}
		s.Dao.dequeuedRepositoryEventCallbackIncr()
		if callback.AnalysisID == "" {
			continue
		}
		log.Info(ctx, "dequeueRepositoryEventCallback> work on event: %s", callback.HookEventUUID)
		if err := s.updateHookEventWithCallback(ctx, callback); err != nil {
			log.ErrorWithStackTrace(ctx, err)
		}
	}
}

func (s *Service) updateHookEventWithCallback(ctx context.Context, callback sdk.HookAnalysisCallback) error {
	b, err := s.Dao.LockRepositoryEvent(callback.VCSServerType, callback.VCSServerName, callback.RepositoryName, callback.HookEventUUID)
	if err != nil {
		return sdk.WrapError(err, "unable to lock hook event %s to manage callback", callback.HookEventUUID)
	}
	if !b {
		// Reenqueue
		if err := s.Dao.EnqueueRepositoryEventCallback(ctx, callback); err != nil {
			return sdk.WrapError(err, "unable to reenqueue repository hook callback")
		}
	}
	defer s.Dao.UnlockRepositoryEvent(callback.VCSServerType, callback.VCSServerName, callback.RepositoryName, callback.HookEventUUID)

	// Load the event
	var hre *sdk.HookRepositoryEvent
	eventKey := cache.Key(repositoryEventRootKey, s.Dao.GetRepositoryMemberKey(hre.VCSServerType, hre.VCSServerName, hre.RepositoryName))
	find, err := s.Cache.Get(eventKey, hre)
	if err != nil {
		return sdk.WrapError(err, "unable to get hook event %s", eventKey)
	}
	if !find {
		return nil
	}

	if hre.Status != sdk.HookEventStatusAnalysis {
		return nil
	}

	for i := range hre.Analyses {
		a := &hre.Analyses[i]
		if a.AnalyzeID == callback.AnalysisID {
			if a.Status == sdk.RepositoryAnalysisStatusInProgress {
				a.Status = callback.AnalysisStatus
				if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
					return err
				}
				break
			}
		}
	}
	return nil
}
