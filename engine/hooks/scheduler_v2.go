package hooks

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gorhill/cronexpr"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
)

/*
hooks:v2:schedulers:<vcs>:<repo>:<workflow>:<whID>: Scheduler definition (sdk.V2WorkflowHook)
hooks:queue:schedulers: Contains the next Scheduler executions. MemberKey = whID
hooks:v2:executions:lock:<whID>
*/

type SchedulerExecution struct {
	SchedulerDef      sdk.V2WorkflowHook
	NextExecutionTime int64
}

func (s *Service) instantiateScheduler(ctx context.Context, hooks []sdk.V2WorkflowHook) error {
	// sort hooks by entity
	sortedHooks := make(map[string][]sdk.V2WorkflowHook)
	for _, h := range hooks {
		hooksEntity, has := sortedHooks[h.EntityID]
		if !has {
			hooksEntity = make([]sdk.V2WorkflowHook, 0)
		}
		hooksEntity = append(hooksEntity, h)
		sortedHooks[h.EntityID] = hooksEntity
	}

	for _, hs := range sortedHooks {
		vcsName := hs[0].VCSName
		repoName := hs[0].RepositoryName
		wkfName := hs[0].WorkflowName

		// Remove all schedulers && next execution for the given workflow
		if err := s.removeSchedulersAndNextExecution(ctx, vcsName, repoName, wkfName); err != nil {
			return err
		}

		// For each new scheduler, save definition + create next execution
		for _, h := range hs {
			if err := s.Dao.CreateSchedulerDefinition(ctx, h); err != nil {
				return err
			}
			if err := s.createSchedulerNextExecution(ctx, h); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) createSchedulerNextExecution(ctx context.Context, h sdk.V2WorkflowHook) error {
	// Create the next execution
	//Parse the cron expr
	cronExpr, err := cronexpr.Parse(h.Data.Cron)
	if err != nil {
		return sdk.WrapError(err, "unable to parse cron expression: %v", h.Data.Cron)
	}

	confTimezone := h.Data.CronTimeZone
	loc, err := time.LoadLocation(confTimezone)
	if err != nil {
		return sdk.WrapError(err, "unable to parse timezone: %v", confTimezone)
	}

	//Compute a new date
	t0 := time.Now().In(loc)
	nextSchedule := cronExpr.Next(t0)
	nextExecution := SchedulerExecution{
		SchedulerDef:      h,
		NextExecutionTime: nextSchedule.UnixNano(),
	}
	if err := s.Dao.CreateSchedulerNextExecution(ctx, nextExecution); err != nil {
		return err
	}
	return nil
}

func (s *Service) removeSchedulersAndNextExecution(ctx context.Context, vcs, repo, workflow string) error {
	keys, err := s.Dao.SchedulerKeysByWorkflow(ctx, vcs, repo, workflow)
	if err != nil {
		return err
	}
	for _, k := range keys {
		var h sdk.V2WorkflowHook
		found, err := s.Dao.store.Get(k, &h)
		if err != nil {
			return err
		}
		if !found {
			continue
		}
		log.Info(ctx, "delete scheduler definition and next execution for workflow %s/%s/%s %s %s", vcs, repo, workflow, h.Data.Cron, h.Data.CronTimeZone)
		if err := s.Dao.RemoveScheduler(ctx, vcs, repo, workflow, h.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) schedulerExecutionRoutine(ctx context.Context) {
	tick := time.NewTicker(time.Duration(10) * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Error(ctx, "schedulerExecutionRoutine > exiting goroutine: %v", ctx.Err())
			return
		case <-tick.C:
			schedulerExecutions, err := s.Dao.GetAllSchedulerExecutions(ctx)
			if err != nil {
				log.Error(ctx, "schedulerExecutionRoutine > unable to load all scheduler executions")
				continue
			}

			for _, e := range schedulerExecutions {
				if e.NextExecutionTime > time.Now().UnixNano() {
					continue
				}
				if err := s.enqueueSchedulerAsHookRepositoryEvent(ctx, e); err != nil {
					log.Error(ctx, "schedulerExecutionRoutine > unable to load all scheduler executions")
					continue
				}
			}
		}
	}
}

func (s *Service) enqueueSchedulerAsHookRepositoryEvent(ctx context.Context, e SchedulerExecution) error {
	// Lock execution
	lockKey := cache.Key(schedulerExecutionLockRootKey, e.SchedulerDef.ID)
	b, err := s.Dao.store.Lock(lockKey, 20*time.Second, 10, 1)
	if err != nil {
		return err
	}
	if !b {
		return nil
	}
	defer s.Dao.store.Unlock(lockKey)

	// Reload execution to check execution time
	updatedExecution, err := s.Dao.GetSchedulerExecution(ctx, e.SchedulerDef.ID)
	if err != nil {
		return err
	}
	if updatedExecution == nil {
		return nil
	}

	if updatedExecution.NextExecutionTime > time.Now().UnixNano() {
		return nil
	}

	// Check if hook definition still exists
	existingDef, err := s.Dao.GetSchedulerDefinition(ctx, updatedExecution.SchedulerDef.VCSName, updatedExecution.SchedulerDef.RepositoryName, updatedExecution.SchedulerDef.WorkflowName, updatedExecution.SchedulerDef.ID)
	if err != nil {
		return err
	}
	if existingDef == nil {
		// Remove execution
		log.Info(ctx, "Scheduler definition doesn't exist anymore (%s/%s/%s/%s), skip execution", updatedExecution.SchedulerDef.VCSName, updatedExecution.SchedulerDef.RepositoryName, updatedExecution.SchedulerDef.WorkflowName, updatedExecution.SchedulerDef.ID)
		return s.Dao.RemoveSchedulerExecution(ctx, updatedExecution.SchedulerDef.ID)
	}

	// Create HookRepositoryEvent
	bts, _ := json.Marshal(sdk.V2WorkflowScheduleEvent{Schedule: updatedExecution.SchedulerDef.Data.Cron})
	he := &sdk.HookRepositoryEvent{
		UUID:           sdk.UUID(),
		Created:        time.Now().UnixNano(),
		EventName:      sdk.WorkflowHookScheduler,
		VCSServerName:  updatedExecution.SchedulerDef.VCSName,
		RepositoryName: updatedExecution.SchedulerDef.RepositoryName,
		Body:           bts,
		ExtractData: sdk.HookRepositoryEventExtractData{
			Commit:       updatedExecution.SchedulerDef.Commit,
			Ref:          updatedExecution.SchedulerDef.Ref,
			CDSEventName: sdk.WorkflowHookTypeScheduler,
			Scheduler: sdk.HookRepositoryEventExtractDataScheduler{
				TargetVCS:      updatedExecution.SchedulerDef.Data.VCSServer,
				TargetRepo:     updatedExecution.SchedulerDef.Data.RepositoryName,
				TargetWorkflow: updatedExecution.SchedulerDef.WorkflowName,
				TargetProject:  updatedExecution.SchedulerDef.ProjectKey,
				Cron:           updatedExecution.SchedulerDef.Data.Cron,
				Timezone:       updatedExecution.SchedulerDef.Data.CronTimeZone,
			},
		},
		Status:              sdk.HookEventStatusScheduled,
		ProcessingTimestamp: time.Now().UnixNano(),
		LastUpdate:          time.Now().UnixNano(),
		EventType:           "", //empty for scheduler
	}

	// Save event
	if err := s.Dao.SaveRepositoryEvent(ctx, he); err != nil {
		return sdk.WrapError(err, "unable to create repository event %s", he.GetFullName())
	}

	// Enqueue event
	if err := s.Dao.EnqueueRepositoryEvent(ctx, he); err != nil {
		return sdk.WrapError(err, "unable to enqueue repository event %s", he.GetFullName())
	}
	s.Dao.enqueuedRepositoryEventIncr()

	if err := s.createSchedulerNextExecution(ctx, updatedExecution.SchedulerDef); err != nil {
		return err
	}

	return nil
}
