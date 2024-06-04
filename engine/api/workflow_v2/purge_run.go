package workflow_v2

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/rockbears/log"
)

const (
	KeyV2RunPurge = "v2:purge:run"
)

func PurgeWorkflowRun(ctx context.Context, DBFunc func() *gorp.DbMap, purgeRoutineTIcker int64) {
	tickPurge := time.NewTicker(time.Duration(purgeRoutineTIcker) * time.Minute)
	defer tickPurge.Stop()

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Exiting purge workflow: %v", ctx.Err())
				return
			}
		case <-tickPurge.C:
			ids, err := LoadRunIDsToDelete(ctx, DBFunc())
			if err != nil {
				log.ErrorWithStackTrace(ctx, err)
			}
			for _, id := range ids {
				if err := deleteRun(ctx, DBFunc(), id); err != nil {
					log.ErrorWithStackTrace(ctx, err)
				}
			}
		}
	}

}

func deleteRun(ctx context.Context, db *gorp.DbMap, id string) error {
	ctx = context.WithValue(ctx, cdslog.WorkflowRunID, id)

	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	run, err := LoadAndLockRunByID(ctx, db, id)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil
		}
		return err
	}
	if err := gorpmapping.Delete(db, run); err != nil {
		return err
	}

	return sdk.WithStack(tx.Commit())
}
