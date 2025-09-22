package migrate

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/rockbears/log"
)

func MigrateRunSerialization(ctx context.Context, db *gorp.DbMap, store cache.Store) error {
	prjs, err := project.LoadAll(ctx, db, store)
	if err != nil {
		return err
	}

	var lastError error
	for _, p := range prjs {
		ctx = context.WithValue(ctx, cdslog.Project, p.Key)
		if err := migrateProjectRuns(ctx, db, store, p.Key); err != nil {
			lastError = err
			log.ErrorWithStackTrace(ctx, err)
		}
	}
	return lastError
}

func migrateProjectRuns(ctx context.Context, db *gorp.DbMap, store cache.Store, projectKey string) error {
	var totalWorkflow, totalOK, totalFixed int
	log.Info(ctx, "MigrateRunSerialization on project %s started", projectKey)
	defer func() {
		log.Info(ctx, "MigrateRunSerialization on project %s - Total workflow %d - AlreadyOK %d - Fixed %d", projectKey, totalWorkflow, totalOK, totalFixed)
	}()
	lockKey := cache.Key("cds", "api", "migrate", "run_serialization", projectKey)
	b, err := store.Lock(lockKey, 5*time.Minute, 50, 1)
	if err != nil {
		return err
	}
	if !b {
		return nil
	}

	runIds, err := workflow_v2.LoadRunIDsByProject(ctx, db, projectKey)
	if err != nil {
		return err
	}

	totalWorkflow = len(runIds)

	for _, id := range runIds {
		// If no error, skip it
		_, err := workflow_v2.LoadRunByID(ctx, db, id)
		if err == nil {
			totalOK++
			continue
		}
		if !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return sdk.WrapError(err, "unable to load workflow run %s", id)
		}

		// If run not found => data corrupted
		log.Info(ctx, "MigrateRunSerialization fixing run %s", id)
		run, err := workflow_v2.LoadRunUnsafeByID(ctx, db, id)
		if err != nil {
			return sdk.WrapError(err, "unable to load workflow run %s unsafe", id)
		}
		tx, err := db.Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		if err := workflow_v2.UpdateRun(ctx, tx, run); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			_ = tx.Rollback()
			return sdk.WithStack(err)
		}
		totalFixed++
	}
	return nil
}
