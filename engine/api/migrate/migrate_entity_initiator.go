package migrate

import (
	"context"
	"fmt"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/sdk"
)

const MigrateEntityInitiatorName = "MigrateEntityInitiator"

// MigrateEntityInitiator gives an owner to the entities written before the initiator column existed,
// head entities first, and re-signs them.
// The overall progress is the number of rows still having a NULL initiator.
func MigrateEntityInitiator(ctx context.Context, db *gorp.DbMap) error {
	ids, err := entity.LoadIDsWithoutInitiator(ctx, db)
	if err != nil {
		return err
	}
	log.Info(ctx, "%s: %d entities to migrate", MigrateEntityInitiatorName, len(ids))
	return migrateEntityInitiators(ctx, db, ids)
}

func migrateEntityInitiators(ctx context.Context, db *gorp.DbMap, ids []string) error {
	var migrated, skipped, failed int
	for i, id := range ids {
		done, err := migrateEntityInitiator(ctx, db, id)
		switch {
		case err != nil:
			failed++
			log.Error(ctx, "%s: entity %s: %v", MigrateEntityInitiatorName, id, err)
		case done:
			migrated++
		default:
			skipped++
		}
		if (i+1)%1000 == 0 {
			log.Info(ctx, "%s: %d/%d entities processed", MigrateEntityInitiatorName, i+1, len(ids))
		}
	}
	log.Info(ctx, "%s: %d migrated, %d skipped, %d failed", MigrateEntityInitiatorName, migrated, skipped, failed)
	if failed > 0 {
		return fmt.Errorf("%d entities could not be migrated, run the migration again to retry them", failed)
	}
	return nil
}

// migrateEntityInitiator rebuilds the owner of one entity from its user_id column
func migrateEntityInitiator(ctx context.Context, db *gorp.DbMap, id string) (bool, error) {
	tx, err := db.Begin()
	if err != nil {
		return false, sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	e, err := entity.LoadAndLockByID(ctx, tx, id)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	if e.Initiator != nil {
		return false, nil
	}
	if err := entity.Update(ctx, tx, e); err != nil {
		return false, err
	}
	return true, sdk.WithStack(tx.Commit())
}
