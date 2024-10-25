package api

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/worker_v2"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

const (
	workerLockKey          = "worker:lock"
	workerHeartbeatTimeout = 300.0
)

func DisabledDeadWorkers(ctx context.Context, store cache.Store, DBFunc func() *gorp.DbMap) {
	db := DBFunc()
	tickDisable := time.NewTicker(1 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Exiting worker ticker: %v", ctx.Err())
			}
			return
		case <-tickDisable.C:
			workers, err := worker_v2.LoadDeadWorkers(ctx, db, workerHeartbeatTimeout, []string{sdk.StatusWaiting, sdk.StatusBuilding})
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
				continue
			}
			for i := range workers {
				if err := DisableDeadWorker(ctx, store, db, workers[i].ID, workers[i].Name); err != nil {
					log.ErrorWithStackTrace(ctx, err)
					continue
				}
			}
		}
	}
}

func DeleteDisabledWorkers(ctx context.Context, store cache.Store, DBFunc func() *gorp.DbMap) {
	db := DBFunc()
	tickDelete := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Exiting worker ticker: %v", ctx.Err())
			}
			return
		case <-tickDelete.C:
			workers, err := worker_v2.LoadWorkerByStatus(ctx, db, sdk.StatusDisabled)
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
				continue
			}
			for i := range workers {
				if err := DeleteDisabledWorker(ctx, store, db, workers[i].ID, workers[i].Name); err != nil {
					log.Error(ctx, "unable to delete disable worker %s: %v", workers[i].ID, err)
					continue
				}
			}
		}
	}
}

func DeleteDisabledWorker(ctx context.Context, store cache.Store, db *gorp.DbMap, workerID string, workerName string) error {
	ctx, next := telemetry.Span(ctx, "deleteDisabledWorker", telemetry.Tag(telemetry.TagWorker, workerName))
	defer next()

	_, next = telemetry.Span(ctx, "deleteDisabledWorker.lock")
	lockKey := cache.Key(workerLockKey, workerID)
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

	worker, err := worker_v2.LoadByID(ctx, db, workerID)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return sdk.WrapError(err, "unable to load worker %s", workerID)
	}
	if err != nil && sdk.ErrorIs(err, sdk.ErrNotFound) {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	// remove consumer
	if err := authentication.DeleteConsumerByID(tx, worker.ConsumerID); err != nil {
		return sdk.WrapError(err, "unable to delete worker consumer")
	}

	if err := worker_v2.DeleteWorker(ctx, tx, *worker); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

func DisableDeadWorker(ctx context.Context, store cache.Store, db *gorp.DbMap, workerID string, workerName string) error {
	ctx, next := telemetry.Span(ctx, "disableDeadWorker", telemetry.Tag(telemetry.TagWorker, workerName))
	defer next()

	_, next = telemetry.Span(ctx, "disableDeadWorker.lock")
	lockKey := cache.Key(workerLockKey, workerID)
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

	worker, err := worker_v2.LoadByID(ctx, db, workerID)
	if err != nil {
		return err
	}
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return err
	}
	if err != nil && sdk.ErrorIs(err, sdk.ErrNotFound) {
		return nil
	}
	if worker.Status == sdk.StatusDisabled {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	log.Debug(ctx, "Disable worker %s[%s] LastBeat:%v status:%s", worker.Name, worker.ID, worker.LastBeat, worker.Status)
	worker.Status = sdk.StatusDisabled
	if err := worker_v2.Update(ctx, tx, worker); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}
