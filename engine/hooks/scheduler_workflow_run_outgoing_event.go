package hooks

import (
	"context"
	"encoding/json"
	"fmt"

	"time"

	"go.opencensus.io/trace"

	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
	"github.com/rockbears/log"
)

func (s *Service) dequeueWorkflowRunOutgoingEvent(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			log.ErrorWithStackTrace(ctx, ctx.Err())
			return
		}
		size, err := s.Dao.WorkflowRunOutgoingEventQueueLen()
		if err != nil {
			log.Error(ctx, "dequeueRepositoryOutgoingEvent > Unable to get queueLen: %v", err)
			continue
		}
		log.Debug(ctx, "dequeueWorkflowRunOutgoingEvent> current queue size: %d", size)

		if s.Maintenance {
			log.Info(ctx, "Maintenance enable, wait 1 minute. Queue %d", size)
			time.Sleep(1 * time.Minute)
			continue
		}

		// Dequeuing context
		var eventKey string
		if ctx.Err() != nil {
			log.Error(ctx, "%v", err)
			return
		}

		// Get next EventKEY
		if err := s.Cache.DequeueWithContext(ctx, workflowRunOutgoingEventQueue, 250*time.Millisecond, &eventKey); err != nil {
			continue
		}
		s.Dao.dequeuedWorkflowRunOutgoingEventIncr()
		if eventKey == "" {
			continue
		}
		log.Info(ctx, "dequeueRepositoryOutgoingEvent> work on event: %s", eventKey)
		ctx := telemetry.New(ctx, s, "hooks.dequeueRepositoryOutgoingEvent", nil, trace.SpanKindUnspecified)
		if err := s.manageWorkflowRunOutgoingEvent(ctx, eventKey); err != nil {
			log.ErrorWithStackTrace(ctx, err)
			continue
		}

	}
}

func (s *Service) manageWorkflowRunOutgoingEvent(ctx context.Context, eventKey string) error {
	ctx, next := telemetry.Span(ctx, "s.manageWorkflowRunOutgoingEvent")
	defer next()

	// Load the event
	var outgoingEvent sdk.HookWorkflowRunOutgoingEvent
	find, err := s.Cache.Get(eventKey, &outgoingEvent)
	if err != nil {
		log.Error(ctx, "manageWorkflowRunOutgoingEvent> cannot get workflow run outgoing event from cache %s: %v", eventKey, err)
	}
	if !find {
		return nil
	}
	ctx = context.WithValue(ctx, cdslog.HookEventID, outgoingEvent.UUID)
	ctx = context.WithValue(ctx, cdslog.VCSServer, outgoingEvent.Event.WorkflowVCSServer)
	ctx = context.WithValue(ctx, cdslog.Repository, outgoingEvent.Event.WorkflowRepository)
	ctx = context.WithValue(ctx, cdslog.Project, outgoingEvent.Event.WorkflowProject)
	ctx = context.WithValue(ctx, cdslog.Workflow, outgoingEvent.Event.WorkflowName)
	ctx = context.WithValue(ctx, cdslog.WorkflowRunID, outgoingEvent.Event.WorkflowRunID)

	telemetry.Current(ctx,
		telemetry.Tag(telemetry.TagVCSServer, outgoingEvent.Event.WorkflowVCSServer),
		telemetry.Tag(telemetry.TagRepository, outgoingEvent.Event.WorkflowVCSServer),
		telemetry.Tag(telemetry.TagProjectKey, outgoingEvent.Event.WorkflowProject),
		telemetry.Tag(telemetry.TagWorkflow, outgoingEvent.Event.WorkflowName),
		telemetry.Tag(telemetry.TagEventID, outgoingEvent.UUID))

	b, err := s.Dao.LockWorkflowRunOutgoingEvent(outgoingEvent.UUID)
	if err != nil {
		return sdk.WrapError(err, "unable to lock outgoing hook event %s", outgoingEvent.GetFullName())
	}
	defer s.Dao.UnlockWorkflowRunOutgoingEvent(outgoingEvent.UUID)

	if !b {
		// reenqueue
		if err := s.Dao.EnqueueWorkflowRunOutgoingEvent(ctx, &outgoingEvent); err != nil {
			return sdk.WrapError(err, "unable to reenqueue workflow run outgoing event")
		}
	}

	find, err = s.Cache.Get(eventKey, &outgoingEvent)
	if err != nil {
		log.Error(ctx, "manageWorkflowRunOutgoingEvent> cannot get workflow run outgoingevent from cache %s: %v", eventKey, err)
	}
	if !find {
		return nil
	}

	if outgoingEvent.NbErrors >= s.Cfg.RetryError {
		log.Info(ctx, "manageWorkflowRunOutgoingEvent> Event %s stopped: to many errors:%d lastError:%s", outgoingEvent.GetFullName(), outgoingEvent.NbErrors, outgoingEvent.LastError)
		outgoingEvent.Status = sdk.HookEventStatusError
		if err := s.Dao.SaveWorkflowRunOutgoingEvent(ctx, &outgoingEvent); err != nil {
			return sdk.WrapError(err, "maxerror > unable to save workflow run event %s", outgoingEvent.GetFullName())
		}
		if err := s.Dao.RemoveWorkflowRunOutgoingEventFromInProgressList(ctx, outgoingEvent); err != nil {
			return sdk.WrapError(err, "maxerror > unable to remove event %s from inprogress list", outgoingEvent.GetFullName())
		}
		return nil
	}

	if err := s.executeOutgoingEvent(ctx, &outgoingEvent); err != nil {
		log.Warn(ctx, "outgoingEvent> %s failed err[%d]: %v", outgoingEvent.GetFullName(), outgoingEvent.NbErrors, err)
		outgoingEvent.LastError = err.Error()
		outgoingEvent.NbErrors++
		if err := s.Dao.SaveWorkflowRunOutgoingEvent(ctx, &outgoingEvent); err != nil {
			return sdk.WrapError(err, "unable to save workflow run outgoing event %s", outgoingEvent.GetFullName())
		}
		if err := s.Dao.EnqueueWorkflowRunOutgoingEvent(ctx, &outgoingEvent); err != nil {
			return sdk.WrapError(err, "unable to enqueue workflow run outgoing event %s", outgoingEvent.GetFullName())
		}
	}
	return nil
}

func (s *Service) executeOutgoingEvent(ctx context.Context, outgoingEvent *sdk.HookWorkflowRunOutgoingEvent) error {
	ctx, next := telemetry.Span(ctx, "s.executeOutgoingEvent")
	defer next()

	// Retrive hooks to trigger
	if len(outgoingEvent.HooksToTriggers) == 0 {
		// Retrieve hooks from API
		request := sdk.HookListWorkflowRequest{
			HookEventUUID:       outgoingEvent.UUID,
			Ref:                 outgoingEvent.Event.Request.Git.Ref, // ref of workflow.repository ( target )
			RepositoryEventName: sdk.WorkflowHookEventRun,
			VCSName:             outgoingEvent.Event.Request.Git.Server,
			RepositoryName:      outgoingEvent.Event.Request.Git.Repository,
			Workflows: []sdk.EntityFullName{{
				ProjectKey: outgoingEvent.Event.WorkflowProject,
				VCSName:    outgoingEvent.Event.WorkflowVCSServer,
				RepoName:   outgoingEvent.Event.WorkflowRepository,
				Name:       outgoingEvent.Event.WorkflowName,
			}},
		}
		workflowHooks, err := s.Client.ListWorkflowToTrigger(ctx, request)
		if err != nil {
			return err
		}

		// If no hooks, we can end the process
		if len(workflowHooks) == 0 {
			outgoingEvent.Status = sdk.HookEventStatusSkipped
			if err := s.Dao.RemoveWorkflowRunOutgoingEventFromInProgressList(ctx, *outgoingEvent); err != nil {
				return err
			}
			if err := s.Dao.SaveWorkflowRunOutgoingEvent(ctx, outgoingEvent); err != nil {
				return err
			}
			return nil
		}

		outgoingEvent.HooksToTriggers = make([]sdk.HookWorkflowRunOutgoingEventHooks, 0, len(workflowHooks))
		for _, wh := range workflowHooks {
			outgoingEvent.HooksToTriggers = append(outgoingEvent.HooksToTriggers, sdk.HookWorkflowRunOutgoingEventHooks{
				V2WorkflowHook: wh,
				Status:         sdk.HookEventStatusScheduled,
			})
		}
		if err := s.Dao.SaveWorkflowRunOutgoingEvent(ctx, outgoingEvent); err != nil {
			return err
		}
	}

	// Trigger hooks
	allDone := true
	eventBody, _ := json.Marshal(outgoingEvent.Event.Request)
	for i := range outgoingEvent.HooksToTriggers {
		wh := &outgoingEvent.HooksToTriggers[i]
		if wh.Status != sdk.HookEventStatusScheduled {
			continue
		}

		// Create repository hook event
		hre := &sdk.HookRepositoryEvent{
			UUID:           sdk.UUID(),
			Created:        time.Now().UnixNano(),
			EventName:      sdk.WorkflowHookEventRun,
			VCSServerName:  wh.V2WorkflowHook.VCSName,
			RepositoryName: wh.V2WorkflowHook.RepositoryName,
			Body:           eventBody,
			Status:         sdk.HookEventStatusScheduled,
			ExtractData: sdk.HookRepositoryEventExtractData{
				WorkflowRun: sdk.HookRepositoryEventExtractedDataWorkflowRun{
					Project:               wh.ProjectKey,
					TargetVCS:             wh.Data.VCSServer,
					TargetRepository:      wh.Data.RepositoryName,
					Workflow:              wh.WorkflowName,
					OutgoingHookEventUUID: outgoingEvent.UUID,
				},
			},
			WorkflowHooks: []sdk.HookRepositoryEventWorkflow{
				{
					ProjectKey:           wh.ProjectKey,
					VCSIdentifier:        wh.VCSName,
					RepositoryIdentifier: wh.RepositoryName,
					WorkflowName:         wh.WorkflowName,
					Type:                 wh.Type,
					Status:               sdk.HookEventWorkflowStatusScheduled,
					Ref:                  wh.Ref,
					Commit:               wh.Commit,
					Data:                 wh.Data,
				},
			},
		}
		// Save event
		if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
			wh.Error = fmt.Sprintf("unable to create repository event %s", hre.GetFullName())
			allDone = false
			log.ErrorWithStackTrace(ctx, err)
			outgoingEvent.LastError = wh.Error
			if err := s.Dao.SaveWorkflowRunOutgoingEvent(ctx, outgoingEvent); err != nil {
				return err
			}
			continue
		}

		// Enqueue event
		if err := s.Dao.EnqueueRepositoryEvent(ctx, hre); err != nil {
			wh.Error = fmt.Sprintf("unable to enqueue repository event %s", hre.GetFullName())
			allDone = false
			log.ErrorWithStackTrace(ctx, err)
			outgoingEvent.LastError = wh.Error
			if err := s.Dao.SaveWorkflowRunOutgoingEvent(ctx, outgoingEvent); err != nil {
				return err
			}
			continue
		}

		wh.HookRepositoryEventID = hre.UUID
		wh.Status = sdk.HookEventStatusDone
		if err := s.Dao.SaveWorkflowRunOutgoingEvent(ctx, outgoingEvent); err != nil {
			return err
		}
	}

	if allDone {
		outgoingEvent.Status = sdk.HookEventStatusDone
		if err := s.Dao.RemoveWorkflowRunOutgoingEventFromInProgressList(ctx, *outgoingEvent); err != nil {
			return err
		}
		if err := s.Dao.SaveWorkflowRunOutgoingEvent(ctx, outgoingEvent); err != nil {
			return err
		}
	} else {
		// If there are errors during repository event creation, renqueue the outgoing event
		outgoingEvent.NbErrors++
		if err := s.Dao.SaveWorkflowRunOutgoingEvent(ctx, outgoingEvent); err != nil {
			return err
		}
		if err := s.Dao.EnqueueWorkflowRunOutgoingEvent(ctx, outgoingEvent); err != nil {
			return err
		}
	}
	return nil
}
