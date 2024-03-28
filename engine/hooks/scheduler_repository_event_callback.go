package hooks

import (
	"context"
	"time"

	"github.com/rockbears/log"
	"go.opencensus.io/trace"

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
	find, err := s.Cache.Get(callback.HookEventKey, &hre)
	if err != nil {
		return sdk.WrapError(err, "unable to get hook event %s", callback.HookEventKey)
	}
	if !find {
		log.Info(ctx, "repository hook %s does not exist anymore", callback.HookEventKey)
		return nil
	}

	switch hre.Status {
	case sdk.HookEventStatusAnalysis:
		if callback.AnalysisCallback != nil {
			for i := range hre.Analyses {
				a := &hre.Analyses[i]
				if a.AnalyzeID == callback.AnalysisCallback.AnalysisID {
					if a.Status == sdk.RepositoryAnalysisStatusInProgress {
						a.Status = callback.AnalysisCallback.AnalysisStatus
						a.Error = callback.AnalysisCallback.Error
						hre.ModelUpdated = append(hre.ModelUpdated, callback.AnalysisCallback.Models...)
						hre.WorkflowUpdated = append(hre.WorkflowUpdated, callback.AnalysisCallback.Workflows...)
						if err := s.Dao.SaveRepositoryEvent(ctx, &hre); err != nil {
							return err
						}
						break
					}
				}
			}
		} else {
			return sdk.Errorf("missing analysis callback data")
		}

	case sdk.HookEventStatusSignKey, sdk.HookEventStatusGitInfo:
		if callback.SigningKeyCallback != nil {
			if err := s.manageRepositoryOperationCallback(ctx, *callback.SigningKeyCallback, &hre); err != nil {
				return err
			}
		} else {
			return sdk.Errorf("missing analysis callback data")
		}
	default:
		return nil
	}

	if err := s.Dao.SaveRepositoryEvent(ctx, &hre); err != nil {
		return err
	}

	// if hre is in error or skipped, remove it from in progress list
	if hre.Status == sdk.HookEventStatusError || hre.Status == sdk.HookEventStatusSkipped {
		if err := s.Dao.RemoveRepositoryEventFromInProgressList(ctx, hre); err != nil {
			return err
		}
	} else {
		if err := s.Dao.EnqueueRepositoryEvent(ctx, &hre); err != nil {
			return err
		}
	}
	return nil
}
