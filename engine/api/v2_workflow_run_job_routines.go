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

const jobLockKey = "jobs:lock"

func (api *API) CancelAbandonnedRunResults(ctx context.Context) {
	tick := time.NewTicker(5 * time.Minute)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "%v", ctx.Err())
			}
			return
		case <-tick.C:
			ids, err := workflow_v2.LoadAbandonnedRunResultsID(ctx, api.mustDB())
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
				continue
			}
			for _, id := range ids {
				if err := api.cancelAbandonnedRunResult(ctx, api.mustDB(), id); err != nil {
					log.ErrorWithStackTrace(ctx, err)
				}
			}
		}
	}
}

func (api *API) cancelAbandonnedRunResult(ctx context.Context, db *gorp.DbMap, id string) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}

	defer tx.Rollback()

	runResult, err := workflow_v2.LoadAndLockRunResultByID(ctx, tx, id)
	if err != nil {
		return err
	}

	if runResult == nil {
		log.Debug(ctx, "RunResult %s skipped", id)
		return nil
	}

	log.Debug(ctx, "cancelAbandonnedRunResult: %s", id)

	runResult.Status = sdk.V2WorkflowRunResultStatusCanceled
	if err := workflow_v2.UpdateRunResult(ctx, tx, runResult); err != nil {
		return err
	}

	return sdk.WithStack(tx.Commit())
}

func (api *API) StopUnstartedJobs(ctx context.Context) {
	tickUnstartedJob := time.NewTicker(1 * time.Minute)
	defer tickUnstartedJob.Stop()
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "%v", ctx.Err())
			}
			return
		case <-tickUnstartedJob.C:
			jobWaitingTimeout := api.Config.WorkflowV2.JobWaitingTimeout
			if jobWaitingTimeout == 0 {
				jobWaitingTimeout = 3600
			}
			jobs, err := workflow_v2.LoadOldWaitingRunJob(ctx, api.mustDB(), jobWaitingTimeout)
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
				continue
			}
			for i := range jobs {
				if err := api.failOldWaitingRunJob(ctx, api.Cache, api.mustDB(), jobs[i].ID); err != nil {
					log.ErrorWithStackTrace(ctx, err)
				}
			}
		}
	}
}

func (api *API) StopDeadJobs(ctx context.Context) {
	tickStopDeadJobs := time.NewTicker(1 * time.Minute)
	defer tickStopDeadJobs.Stop()
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
	defer tickScheduledJob.Stop()
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "%v", ctx.Err())
			}
			return
		case <-tickScheduledJob.C:
			jobSchedulingTimeout := api.Config.WorkflowV2.JobSchedulingTimeout
			if jobSchedulingTimeout == 0 {
				jobSchedulingTimeout = 600
			}
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

func (api *API) failOldWaitingRunJob(ctx context.Context, store cache.Store, db *gorp.DbMap, runJobID string) error {
	ctx, next := telemetry.Span(ctx, "failOldWaitingRunJob")
	defer next()

	_, next = telemetry.Span(ctx, "failOldWaitingRunJob.lock")
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
	if runJob.Status != sdk.V2WorkflowRunJobStatusWaiting {
		return nil
	}

	ctx = context.WithValue(ctx, cdslog.WorkflowRunID, runJob.WorkflowRunID)
	ctx = context.WithValue(ctx, cdslog.Workflow, runJob.WorkflowName)

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // nolint

	log.Info(ctx, fmt.Sprintf("failRunJob: job %s/%s (timeout %s) on workflow %s run %d", runJob.JobID, runJob.ID, time.Now().Sub(sdk.TimeSafe(&runJob.Queued)).String(), runJob.WorkflowName, runJob.RunNumber))

	runJob.Status = sdk.V2WorkflowRunJobStatusFail

	if err := workflow_v2.UpdateJobRun(ctx, tx, runJob); err != nil {
		return err
	}

	info := sdk.V2WorkflowRunJobInfo{
		WorkflowRunID:    runJob.WorkflowRunID,
		IssuedAt:         time.Now(),
		Level:            sdk.WorkflowRunInfoLevelError,
		WorkflowRunJobID: runJob.ID,
		Message:          "the job has been in waiting state for too long, stopping it. Please contact an administrator",
	}
	if err := workflow_v2.InsertRunJobInfo(ctx, tx, &info); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	// Trigger workflow
	api.EnqueueWorkflowRun(ctx, runJob.WorkflowRunID, runJob.Initiator, runJob.WorkflowName, runJob.RunNumber)

	// Trigger other workflow regarding concurrency
	api.manageEndJobConcurrency(*runJob)

	return nil
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
	if runJob.Status != sdk.V2WorkflowRunJobStatusScheduling {
		return nil
	}

	ctx = context.WithValue(ctx, cdslog.WorkflowRunID, runJob.WorkflowRunID)
	ctx = context.WithValue(ctx, cdslog.Workflow, runJob.WorkflowName)

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // nolint

	log.Info(ctx, fmt.Sprintf("reEnqueueScheduledJob: re-enqueue job %s/%s (timeout %s) on workflow %s run %d", runJob.JobID, runJob.ID, time.Now().Sub(sdk.TimeSafe(runJob.Scheduled)).String(), runJob.WorkflowName, runJob.RunNumber))

	runJob.Status = sdk.V2WorkflowRunJobStatusWaiting
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
	event_v2.PublishRunJobEvent(ctx, store, sdk.EventRunJobEnqueued, *run, *runJob)
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

	ctx = context.WithValue(ctx, cdslog.WorkflowRunID, runJob.WorkflowRunID)
	ctx = context.WithValue(ctx, cdslog.Workflow, runJob.WorkflowName)

	if runJob.Status.IsTerminated() {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // nolint

	log.Info(ctx, fmt.Sprintf("stopDeadJob: stopping job %s/%s on workflow %s run %d", runJob.JobID, runJob.ID, runJob.WorkflowName, runJob.RunNumber))
	runJob.Status = sdk.V2WorkflowRunJobStatusStopped

	now := time.Now()
	runJob.Ended = &now

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
	event_v2.PublishRunJobEvent(ctx, api.Cache, sdk.EventRunJobEnded, *run, *runJob)
	api.EnqueueWorkflowRun(ctx, runJob.WorkflowRunID, runJob.Initiator, runJob.WorkflowName, runJob.RunNumber)

	// Trigger other workflow regarding concurrency
	api.manageEndJobConcurrency(*runJob)
	return nil
}
