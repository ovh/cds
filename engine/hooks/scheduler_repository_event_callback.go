package hooks

import (
	"context"
	"time"

	"github.com/rockbears/log"
	"go.opencensus.io/trace"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func (s *Service) dequeueRepositoryEventCallback(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			log.ErrorWithStackTrace(ctx, ctx.Err())
			return
		}
		size, err := s.Dao.RepositoryEventCallbackQueueLen()
		if err != nil {
			log.Error(ctx, "dequeueRepositoryEventCallback > Unable to get queueLen: %v", err)
			continue
		}
		log.Debug(ctx, "dequeueRepositoryEventCallback> current queue size: %d", size)

		// Dequeuing context
		var callback sdk.HookEventCallback
		if ctx.Err() != nil {
			log.Error(ctx, "%v", ctx.Err())
			return
		}

		// Get next EventKEY
		if err := s.Cache.DequeueWithContext(ctx, repositoryEventCallbackQueue, 250*time.Millisecond, &callback); err != nil {
			continue
		}
		s.Dao.dequeuedRepositoryEventCallbackIncr()
		if callback.AnalysisCallback == nil && callback.SigningKeyCallback == nil {
			continue
		}
		log.Info(ctx, "dequeueRepositoryEventCallback> work on event: %s", callback.HookEventUUID)
		ctx := telemetry.New(ctx, s, "hooks.dequeueRepositoryEventCallback", nil, trace.SpanKindUnspecified)
		telemetry.Current(ctx,
			telemetry.Tag(telemetry.TagVCSServer, callback.VCSServerName),
			telemetry.Tag(telemetry.TagRepository, callback.RepositoryName),
			telemetry.Tag(telemetry.TagEventID, callback.HookEventUUID))

		if err := s.updateHookEventWithCallback(ctx, callback); err != nil {
			log.ErrorWithStackTrace(ctx, err)
		}
	}
}

func (s *Service) updateHookEventWithCallback(ctx context.Context, callback sdk.HookEventCallback) error {
	ctx, next := telemetry.Span(ctx, "s.updateHookEventWithCallback")
	defer next()

	b, err := s.Dao.LockRepositoryEvent(callback.VCSServerName, callback.RepositoryName, callback.HookEventUUID)
	if err != nil {
		return sdk.WrapError(err, "unable to lock hook event %s to manage callback", callback.HookEventUUID)
	}
	if !b {
		// Reenqueue
		if err := s.Dao.EnqueueRepositoryEventCallback(ctx, callback); err != nil {
			return sdk.WrapError(err, "unable to reenqueue repository hook callback")
		}
	}
	defer s.Dao.UnlockRepositoryEvent(callback.VCSServerName, callback.RepositoryName, callback.HookEventUUID)

	// Load the event
	var hre sdk.HookRepositoryEvent
	eventKey := cache.Key(repositoryEventRootKey, s.Dao.GetRepositoryMemberKey(callback.VCSServerName, callback.RepositoryName), callback.HookEventUUID)
	find, err := s.Cache.Get(eventKey, &hre)
	if err != nil {
		return sdk.WrapError(err, "unable to get hook event %s", eventKey)
	}
	if !find {
		log.Info(ctx, "repository hook %s does not exist anymore", eventKey)
		return nil
	}

	if hre.Status != sdk.HookEventStatusAnalysis && hre.Status != sdk.HookEventStatusSignKey {
		return nil
	}

	if callback.AnalysisCallback != nil {
		if callback.AnalysisCallback.UserID != "" {
			hre.UserID = callback.AnalysisCallback.UserID
		}
		for i := range hre.Analyses {
			a := &hre.Analyses[i]
			if a.AnalyzeID == callback.AnalysisCallback.AnalysisID {
				if a.Status == sdk.RepositoryAnalysisStatusInProgress {
					a.Status = callback.AnalysisCallback.AnalysisStatus
					hre.ModelUpdated = append(hre.ModelUpdated, callback.AnalysisCallback.Models...)
					hre.WorkflowUpdated = append(hre.WorkflowUpdated, callback.AnalysisCallback.Workflows...)
					if err := s.Dao.SaveRepositoryEvent(ctx, &hre); err != nil {
						return err
					}
					break
				}
			}
		}
	}

	// Manage signinKey
	if callback.SigningKeyCallback != nil {
		if callback.SigningKeyCallback.SignKey != "" && callback.SigningKeyCallback.Error != "" {
			// event on error commit unverified
			hre.Status = sdk.HookEventStatusError
			hre.LastError = callback.SigningKeyCallback.Error
			hre.NbErrors++
		} else if callback.SigningKeyCallback.SignKey != "" && callback.SigningKeyCallback.Error == "" {
			// commit verified
			hre.SignKey = callback.SigningKeyCallback.SignKey
		} else if callback.SigningKeyCallback.Error != "" {
			hre.LastError = callback.SigningKeyCallback.Error
			hre.NbErrors++
		}
		if err := s.Dao.SaveRepositoryEvent(ctx, &hre); err != nil {
			return err
		}
	}

	if err := s.Dao.EnqueueRepositoryEvent(ctx, &hre); err != nil {
		return err
	}
	return nil
}
