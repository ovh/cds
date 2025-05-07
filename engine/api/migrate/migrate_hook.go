package migrate

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/workflow_v2"
)

func MigrateWorkflowHookSignatureWithoutData(ctx context.Context, db *gorp.DbMap) error {
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
