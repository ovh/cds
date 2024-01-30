package migrate

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/cache"
)

func MigrateHashSignature(ctx context.Context, db *gorp.DbMap, c cache.Store) error {
	entities, err := entity.LoadAllUnsafe(ctx, db)
	if err != nil {
		return err
	}
	for _, e := range entities {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if err := entity.Update(ctx, tx, &e); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	wrs, err := workflow_v2.LoadRunsUnsafe(ctx, db)
	if err != nil {
		return err
	}
	for _, wr := range wrs {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if err := workflow_v2.UpdateRun(ctx, tx, &wr); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

func MigrateBranchToRef(ctx context.Context, db *gorp.DbMap) error {
	entities, err := entity.LoadAllUnsafe(ctx, db)
	if err != nil {
		return err
	}
	for _, e := range entities {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if err := entity.Update(ctx, tx, &e); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	wrs, err := workflow_v2.LoadRunsUnsafe(ctx, db)
	if err != nil {
		return err
	}
	for _, wr := range wrs {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if err := workflow_v2.UpdateRun(ctx, tx, &wr); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	hooks, err := workflow_v2.LoadAllHooksUnsafe(ctx, db)
	if err != nil {
		return err
	}
	for _, h := range hooks {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if err := workflow_v2.UpdateWorkflowHook(ctx, tx, &h); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}
