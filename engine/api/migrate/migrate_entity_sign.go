package migrate

import (
	"context"
	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/cache"
)

func MigrateEntitiesSignature(ctx context.Context, db *gorp.DbMap, c cache.Store) error {
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
	return nil
}
