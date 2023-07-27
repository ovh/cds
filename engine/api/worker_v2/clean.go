package worker_v2

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

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
	tickDelete := time.NewTicker(1 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Exiting worker ticker: %v", ctx.Err())
			}
			return
		case <-tickDelete.C:
			workers, err := LoadDeadWorkers(ctx, db, workerHeartbeatTimeout, []string{sdk.StatusWaiting, sdk.StatusBuilding})
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
				continue
			}
			for i := range workers {
				if err := DisableDeadWorker(ctx, store, db, workers[i].ID); err != nil {
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
			workers, err := LoadWorkerByStatus(ctx, db, sdk.StatusDisabled)
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
				continue
			}
			for i := range workers {
				if err := DeleteDisabledWorker(ctx, store, db, workers[i].ID); err != nil {
					log.ErrorWithStackTrace(ctx, err)
					continue
				}
			}
		}
	}
}

func DeleteDisabledWorker(ctx context.Context, store cache.Store, db *gorp.DbMap, workerID string) error {
	ctx, next := telemetry.Span(ctx, "deleteDisabledWorker")
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

	worker, err := LoadByID(ctx, db, workerID)
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	if err := deleteWorker(ctx, tx, *worker); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

func DisableDeadWorker(ctx context.Context, store cache.Store, db *gorp.DbMap, workerID string) error {
	ctx, next := telemetry.Span(ctx, "disableDeadWorker")
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

	worker, err := LoadByID(ctx, db, workerID)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	log.Debug(ctx, "Disable worker %s[%s] LastBeat:%v status:%s", worker.Name, worker.ID, worker.LastBeat, worker.Status)
	worker.Status = sdk.StatusDisabled
	if err := Update(ctx, tx, worker); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}
