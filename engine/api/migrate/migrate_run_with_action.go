package migrate

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/rockbears/log"
)

func MigrationRunWithActions(ctx context.Context, db *gorp.DbMap) error {
	limit := 50

	offset := 0
	count := 0
	for {
		runs, err := workflow_v2.LoadRunsUnsafeWithPagination(ctx, db, offset, limit)
		if err != nil {
			return err
		}
		for _, r := range runs {
			tx, err := db.Begin()
			if err != nil {
				return err
			}
			// Clean actions
			for k, a := range r.WorkflowData.Actions {
				(&a).Clean()
				r.WorkflowData.Actions[k] = a
			}
			if err := workflow_v2.UpdateRun(ctx, tx, &r); err != nil {
				return err
			}
			if err := tx.Commit(); err != nil {
				return err
			}
		}
		count += len(runs)
		log.Info(ctx, "MigrationRunWithActions> Cleaned %d runs", count)
		if len(runs) < limit {
			break
		}
		offset += limit
	}
	log.Info(ctx, "MigrationRunWithActions> Cleaned all action in workflow run")

	return nil
}
