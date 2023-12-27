package api

import (
	"context"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/event_v2"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
	"github.com/rockbears/log"
)

const (
	jobLockKey           = "jobs:lock"
	jobSchedulingTimeout = 600.0
)

func (api *API) StopDeadJobs(ctx context.Context) {
	tickStopDeadJobs := time.NewTicker(1 * time.Minute)
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "%v", ctx.Err())
			}
			return
		case <-tickStopDeadJobs.C:
			jobs, err := workflow_v2.LoadDeadJobs(ctx, api.mustDB())
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
				continue
			}
			for i := range jobs {
				if err := api.stopDeadJob(ctx, api.Cache, api.mustDB(), jobs[i].ID); err != nil {
					log.ErrorWithStackTrace(ctx, err)
				}
			}
		}
	}
}

func (api *API) ReEnqueueScheduledJobs(ctx context.Context) {
	tickScheduledJob := time.NewTicker(1 * time.Minute)
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "%v", ctx.Err())
			}
			return
		case <-tickScheduledJob.C:
			jobs, err := workflow_v2.LoadOldScheduledRunJob(ctx, api.mustDB(), jobSchedulingTimeout)
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
				continue
			}
			for i := range jobs {
				if err := reEnqueueScheduledJob(ctx, api.Cache, api.mustDB(), jobs[i].ID); err != nil {
					log.ErrorWithStackTrace(ctx, err)
				}
			}
		}
	}
}

func reEnqueueScheduledJob(ctx context.Context, store cache.Store, db *gorp.DbMap, runJobID string) error {
	ctx, next := telemetry.Span(ctx, "reEnqueueScheduledJob")
	defer next()

	_, next = telemetry.Span(ctx, "reEnqueueScheduledJob.lock")
	lockKey := cache.Key(jobLockKey, runJobID)
	b, err := store.Lock(lockKey, 1*time.Minute, 0, 1)
	if err != nil {
		next()
		return err
	}
	if !b {
		next()
		return nil
	}
	next()
	defer func() {
		_ = store.Unlock(lockKey)
	}()

	runJob, err := workflow_v2.LoadRunJobByID(ctx, db, runJobID)
	if err != nil {
		return err
	}
	if runJob.Status != sdk.StatusScheduling {
		return nil
	}

	ctx = context.WithValue(ctx, cdslog.Workflow, runJob.WorkflowName)

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // nolint

	log.Info(ctx, fmt.Sprintf("reEnqueueScheduledJob: re-enqueue job %s/%s (timeout %s) on workflow %s run %d", runJob.JobID, runJob.ID, time.Now().Sub(runJob.Scheduled).String(), runJob.WorkflowName, runJob.RunNumber))

	runJob.Status = sdk.StatusWaiting
	runJob.HatcheryName = ""

	if err := workflow_v2.UpdateJobRun(ctx, tx, runJob); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	// Enqueue the job
	run, err := workflow_v2.LoadRunByID(ctx, db, runJob.WorkflowRunID)
	if err != nil {
		return err
	}
	event_v2.PublishRunJobEvent(ctx, store, sdk.EventRunJobEnqueued, run.Contexts.Git.Server, run.Contexts.Git.Repository, *runJob)
	return nil
}

func (api *API) stopDeadJob(ctx context.Context, store cache.Store, db *gorp.DbMap, runJobID string) error {
	ctx, next := telemetry.Span(ctx, "stopDeadJob")
	defer next()

	_, next = telemetry.Span(ctx, "stopDeadJob.lock")
	lockKey := cache.Key(jobLockKey, runJobID)
	b, err := store.Lock(lockKey, 1*time.Minute, 0, 1)
	if err != nil {
		next()
		return err
	}
	if !b {
		next()
		return nil
	}
	next()
	defer func() {
		_ = store.Unlock(lockKey)
	}()

	runJob, err := workflow_v2.LoadRunJobByID(ctx, db, runJobID)
	if err != nil {
		return err
	}

	run, err := workflow_v2.LoadRunByID(ctx, db, runJob.WorkflowRunID)
	if err != nil {
		return err
	}

	ctx = context.WithValue(ctx, cdslog.Workflow, runJob.WorkflowName)

	if sdk.StatusIsTerminated(runJob.Status) {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // nolint

	log.Info(ctx, fmt.Sprintf("stopDeadJob: stopping job %s/%s on workflow %s run %d", runJob.JobID, runJob.ID, runJob.WorkflowName, runJob.RunNumber))
	runJob.Status = sdk.StatusStopped

	if err := workflow_v2.UpdateJobRun(ctx, tx, runJob); err != nil {
		return err
	}

	info := sdk.V2WorkflowRunJobInfo{
		Level:            sdk.WorkflowRunInfoLevelError,
		WorkflowRunJobID: runJob.ID,
		Message:          fmt.Sprintf("worker %q doesn't respond anymore.", runJob.WorkerName),
		IssuedAt:         time.Now(),
		WorkflowRunID:    runJob.WorkflowRunID,
	}

	if err := workflow_v2.InsertRunJobInfo(ctx, tx, &info); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	// Trigger workflow
	event_v2.PublishRunJobEvent(ctx, api.Cache, sdk.EventRunJobEnded, run.Contexts.Git.Server, run.Contexts.Git.Repository, *runJob)
	api.EnqueueWorkflowRun(ctx, runJob.WorkflowRunID, runJob.UserID, runJob.WorkflowName, runJob.RunNumber)
	return nil
}
