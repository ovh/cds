package migrate

import (
	"context"
	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/workflow_v2"
)

func MigrateRunJobSignature(ctx context.Context, db *gorp.DbMap) error {
	runJobs, err := workflow_v2.UnsafeLoadAllRunJobs(ctx, db)
	if err != nil {
		return err
	}
	for _, rj := range runJobs {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if err := workflow_v2.UpdateJobRun(ctx, tx, &rj); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
