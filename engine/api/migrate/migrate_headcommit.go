package migrate

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func MigrateHeadCommit(ctx context.Context, db *gorp.DbMap, store cache.Store) error {
	// Migrate entity with commit HEAD
	entities, err := entity.LoadUnmigratedHeadEntities(ctx, db)
	if err != nil {
		return err
	}

	for _, e := range entities {
		tx, err := db.Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		e.Head = true
		if err := entity.Update(ctx, tx, &e); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			_ = tx.Rollback()
			return sdk.WithStack(err)
		}
	}
	return nil
}
