package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
	"github.com/rockbears/log"
)

const (
	jobLockKey           = "jobs:lock"
	jobSchedulingTimeout = 600.0
)

func (api *API) StopDeadJobs(ctx context.Context) {
	tickScheduledJob := time.NewTicker(1 * time.Minute)
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "%v", ctx.Err())
			}
			return
		case <-tickScheduledJob.C:
			jobs, err := workflow_v2.LoadDeadJobs(ctx, api.mustDB())
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
				continue
			}
			for i := range jobs {
				if err := stopDeadJob(ctx, api.Cache, api.mustDB(), jobs[i].ID); err != nil {
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

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // nolint

	runJob.Status = sdk.StatusWaiting
	runJob.HatcheryName = ""

	if err := workflow_v2.UpdateJobRun(ctx, tx, runJob); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	// Enqueue the job
	runJobEvent := sdk.WebsocketJobQueueEvent{
		Region:       runJob.Region,
		ModelType:    runJob.ModelType,
		JobRunID:     runJob.ID,
		RunNumber:    runJob.RunNumber,
		WorkflowName: runJob.WorkflowName,
		ProjectKey:   runJob.ProjectKey,
		JobID:        runJob.JobID,
	}
	bts, _ := json.Marshal(runJobEvent)
	if err := store.Publish(ctx, event.JobQueuedPubSubKey, string(bts)); err != nil {
		log.Error(ctx, "%v", err)
	}
	return nil
}

func stopDeadJob(ctx context.Context, store cache.Store, db *gorp.DbMap, runJobID string) error {
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

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // nolint

	runJob.Status = sdk.StatusStopped

	if err := workflow_v2.UpdateJobRun(ctx, tx, runJob); err != nil {
		return err
	}

	info := sdk.V2WorkflowRunJobInfo{
		Level:            sdk.WorkflowRunInfoLevelError,
		WorkflowRunJobID: runJob.ID,
		Message:          fmt.Sprintf("worker %s don't respond anymore.", runJob.WorkerName),
		IssuedAt:         time.Now(),
		WorkflowRunID:    runJob.WorkflowRunID,
	}

	if err := workflow_v2.InsertRunJobInfo(ctx, tx, &info); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	// Enqueue the job
	runJobEvent := sdk.WebsocketJobQueueEvent{
		Region:       runJob.Region,
		ModelType:    runJob.ModelType,
		JobRunID:     runJob.ID,
		RunNumber:    runJob.RunNumber,
		WorkflowName: runJob.WorkflowName,
		ProjectKey:   runJob.ProjectKey,
		JobID:        runJob.JobID,
	}
	bts, _ := json.Marshal(runJobEvent)
	if err := store.Publish(ctx, event.JobQueuedPubSubKey, string(bts)); err != nil {
		log.Error(ctx, "%v", err)
	}
	return nil
}
