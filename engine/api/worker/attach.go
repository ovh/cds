package worker

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// ReleaseAllFromHatchery remove dependency to given given hatchery for all workers linked to it.
func ReleaseAllFromHatchery(db gorp.SqlExecutor, hatcheryID int64) error {
	if _, err := db.Exec("UPDATE worker SET hatchery_id = NULL WHERE hatchery_id = $1", hatcheryID); err != nil {
		return sdk.WrapError(err, "cannot release workers for hatchery with id %d", hatcheryID)
	}
	return nil
}

// ReAttachAllToHatchery search for workers without hatchery an re-attach workers if the hatchery consumer match worker consumer's parent.
func ReAttachAllToHatchery(ctx context.Context, db gorpmapper.SqlExecutorWithTx, hatchery sdk.Service) error {
	query := gorpmapping.NewQuery(`
    SELECT worker.* FROM worker
    JOIN auth_consumer ON auth_consumer.id = worker.auth_consumer_id
    WHERE auth_consumer.parent_id = $1 and worker.hatchery_id IS NULL
  `).Args(hatchery.ConsumerID)
	ws, err := getAll(ctx, db, query)
	if err != nil {
		return err
	}

	for i := range ws {
		log.Info(ctx, "worker.ReAttachAllToHatchery> re-attach worker %s (%s) to hatchery %d (%s)", ws[i].ID, ws[i].Name, hatchery.ID, hatchery.Name)
		ws[i].HatcheryID = &hatchery.ID
		ws[i].HatcheryName = hatchery.Name
		if err := gorpmapping.UpdateAndSign(ctx, db, &dbWorker{Worker: ws[i]}); err != nil {
			return err
		}
	}

	return nil
}
