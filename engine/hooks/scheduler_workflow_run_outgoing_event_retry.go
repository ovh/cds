package hooks

import (
	"context"
	"time"

	"github.com/rockbears/log"
	"go.opencensus.io/trace"

	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

// Get from queue task execution
func (s *Service) manageOldWorkflowRunOutgoingEvent(ctx context.Context) {
	tick := time.NewTicker(time.Duration(s.Cfg.OldRepositoryEventRetry) * time.Minute).C

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Exiting manageOldWorkflowRunOutgoingEvent: %v", ctx.Err())
			}
			return
		case <-tick:
			if s.Maintenance {
				log.Info(ctx, "Maintenance enable, wait 1 minute")
				time.Sleep(1 * time.Minute)
				continue
			}

			routgoingEventKeys, err := s.Dao.ListInProgressWorkflowRunOutgoingEvent(ctx)
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
				continue
			}
			for _, k := range routgoingEventKeys {
				ctx := telemetry.New(ctx, s, "hooks.manageOldWorkflowRunOutgoingEvent", nil, trace.SpanKindUnspecified)
				if err := s.checkInProgressOutgoingEvent(ctx, k); err != nil {
					log.ErrorWithStackTrace(ctx, err)
					continue
				}
			}
		}
	}
}

func (s *Service) checkInProgressOutgoingEvent(ctx context.Context, repoEventKey string) error {
	ctx, next := telemetry.Span(ctx, "s.checkInProgressOutgoingEvent")
	defer next()

	var eventTmp sdk.HookWorkflowRunOutgoingEvent
	find, err := s.Cache.Get(repoEventKey, &eventTmp)
	if err != nil {
		return err
	}
	if !find {
		log.Info(ctx, "workflow run outgoing event %s does not exist anymore.", eventTmp.GetFullName())
		if err := s.Dao.RemoveWorkflowRunOutgoingEventFromInProgressList(ctx, eventTmp); err != nil {
			return err
		}
	}

	telemetry.Current(ctx,
		telemetry.Tag(telemetry.TagVCSServer, eventTmp.Event.WorkflowVCSServer),
		telemetry.Tag(telemetry.TagRepository, eventTmp.Event.WorkflowRepository),
		telemetry.Tag(telemetry.TagWorkflow, eventTmp.Event.WorkflowName),
		telemetry.Tag(telemetry.TagProjectKey, eventTmp.Event.WorkflowProject),
		telemetry.Tag(telemetry.TagWorkflowRun, eventTmp.Event.WorkflowRunID),
		telemetry.Tag(telemetry.TagEventID, eventTmp.UUID))

	ctx = context.WithValue(ctx, cdslog.Project, eventTmp.Event.WorkflowProject)
	ctx = context.WithValue(ctx, cdslog.VCSServer, eventTmp.Event.WorkflowVCSServer)
	ctx = context.WithValue(ctx, cdslog.Repository, eventTmp.Event.WorkflowRepository)
	ctx = context.WithValue(ctx, cdslog.Workflow, eventTmp.Event.WorkflowName)
	ctx = context.WithValue(ctx, cdslog.WorkflowRunID, eventTmp.Event.WorkflowRunID)

	b, err := s.Dao.LockWorkflowRunOutgoingEvent(eventTmp.UUID)
	if err != nil {
		return sdk.WrapError(err, "unable to lock outgoing event %s", eventTmp.GetFullName())
	}
	if !b {
		return nil
	}
	defer s.Dao.UnlockWorkflowRunOutgoingEvent(eventTmp.UUID)

	var outgoingEvent sdk.HookWorkflowRunOutgoingEvent
	find, err = s.Cache.Get(repoEventKey, &outgoingEvent)
	if err != nil {
		return sdk.WrapError(err, "unable to retrieve outgoing event")
	}
	if !find {
		log.Info(ctx, "workflow run outgoing event %s does not exist anymore.", outgoingEvent.GetFullName())
		if err := s.Dao.RemoveWorkflowRunOutgoingEventFromInProgressList(ctx, outgoingEvent); err != nil {
			return err
		}
		return nil
	}

	queueLen, err := s.Dao.WorkflowRunOutgoingEventQueueLen()
	if err != nil {
		return err
	}

	// Check last update time
	if time.Now().UnixMilli()-outgoingEvent.LastUpdate > RetryDelayMilli && queueLen < s.Cfg.OldRepositoryEventQueueLen {
		log.Info(ctx, "re-enqueue outgoing event %s", outgoingEvent.GetFullName())
		if err := s.Dao.EnqueueWorkflowRunOutgoingEvent(ctx, &outgoingEvent); err != nil {
			return err
		}
	}
	return nil
}
