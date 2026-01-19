package migrate

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
	"github.com/rockbears/yaml"
)

func MigrationRunWithActions(ctx context.Context, db *gorp.DbMap) error {
	limit := 50

	// Clean entities of type actions
	offset := 0
	count := 0
	for {
		entities, err := entity.LoadEntitiesByTypeUnsafeWithPagination(ctx, db, sdk.EntityTypeAction, offset, limit)
		if err != nil {
			return err
		}
		for i := range entities {
			ent := &entities[i]
			var act sdk.V2Action
			if err := yaml.Unmarshal([]byte(ent.Data), &act); err != nil {
				return err
			}
			(&act).Clean()
			bts, _ := yaml.Marshal(act)
			ent.Data = string(bts)

			tx, err := db.Begin()
			if err != nil {
				return err
			}
			if err := entity.Update(ctx, tx, ent); err != nil {
				return err
			}
			if err := tx.Commit(); err != nil {
				return err
			}
		}
		count += len(entities)
		log.Info(ctx, "MigrationRunWithActions> Cleaned %d action entities", count)

		if len(entities) < limit {
			break
		}
		offset += limit
	}
	log.Info(ctx, "MigrationRunWithActions> Cleaned all action entities")

	// Clean actions in workflow run
	offset = 0
	count = 0
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
