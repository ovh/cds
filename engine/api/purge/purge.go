package purge

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-gorp/gorp"
	"go.opencensus.io/stats"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//Initialize starts goroutines for workflows
func Initialize(ctx context.Context, store cache.Store, DBFunc func() *gorp.DbMap, workflowRunsMarkToDelete, workflowRunsDeleted *stats.Int64Measure) {
	tickPurge := time.NewTicker(15 * time.Minute)
	defer tickPurge.Stop()

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error("Exiting purge: %v", ctx.Err())
				return
			}
		case <-tickPurge.C:
			log.Debug("purge> Deleting all workflow run marked to delete...")
			if err := deleteWorkflowRunsHistory(ctx, DBFunc(), workflowRunsDeleted); err != nil {
				log.Warning("purge> Error on deleteWorkflowRunsHistory : %v", err)
			}

			log.Debug("purge> Deleting all workflow marked to delete....")
			if err := workflows(ctx, DBFunc(), store, workflowRunsMarkToDelete); err != nil {
				log.Warning("purge> Error on workflows : %v", err)
			}
		}
	}
}

// workflows purges all marked workflows
func workflows(ctx context.Context, db *gorp.DbMap, store cache.Store, workflowRunsMarkToDelete *stats.Int64Measure) error {
	query := "SELECT id, project_id FROM workflow WHERE to_delete = true ORDER BY id ASC"
	res := []struct {
		ID        int64 `db:"id"`
		ProjectID int64 `db:"project_id"`
	}{}

	if _, err := db.Select(&res, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return sdk.WrapError(err, "Unable to load workflows")
	}

	var wrkflws = make([]sdk.Workflow, len(res))
	var projects = map[int64]sdk.Project{}

	for i, r := range res {
		// Force delete workflow runs if any
		n, err := workflow.PurgeAllWorkflowRunsByWorkflowID(db, r.ID)
		if err != nil {
			log.Error("unable to mark workflow runs to delete with workflow_id %d: %v", r.ID, err)
			continue
		}
		if n > 0 {
			// If there is workflow runs to delete, wait for it...
			if workflowRunsMarkToDelete != nil {
				observability.Record(ctx, workflowRunsMarkToDelete, int64(n))
			}
			continue
		}

		// Checks if there is any workflow_runs
		nbWorkflowRuns, err := db.SelectInt("select count(1) from workflow_run where workflow_id = $1", r.ID)
		if err != nil {
			log.Error("unable to count workflow runs for workflow_id %d: %v", r.ID, err)
			continue
		}
		if nbWorkflowRuns > 0 {
			log.Info("skip workflow %d deletion because there are still %d workflow_runs to delete", r.ID, nbWorkflowRuns)
			continue
		}

		proj, has := projects[r.ProjectID]
		if !has {
			p, err := project.LoadByID(db, store, r.ProjectID, nil)
			if err != nil {
				log.Error("purge.Workflows> unable to load project %d: %v", r.ProjectID, err)
				continue
			}
			projects[r.ProjectID] = *p
			proj = *p
		}

		w, err := workflow.LoadByID(db, store, &proj, r.ID, nil, workflow.LoadOptions{})
		if err != nil {
			log.Warning("unable to load workflow %d due to error %v, we try to delete it", r.ID, err)
			if _, err := db.Exec("delete from w_node_trigger where child_node_id IN (SELECT id from w_node where workflow_id = $1)", r.ID); err != nil {
				log.Error("Unable to delete from w_node_trigger for workflow %d: %v", r.ID, err)
			}
			if _, err := db.Exec("delete from w_node where workflow_id = $1", r.ID); err != nil {
				log.Error("Unable to delete from w_node for workflow %d: %v", r.ID, err)
			}
			if _, err := db.Exec("delete from workflow where id = $1", r.ID); err != nil {
				log.Error("Unable to delete from workflow with id %d: %v", r.ID, err)
			} else {
				log.Warning("workflow with id %d is deleted", r.ID)
			}
			continue
		}
		wrkflws[i] = *w
	}

	for _, w := range wrkflws {
		proj, has := projects[w.ProjectID]
		if !has {
			continue
		}

		tx, err := db.Begin()
		if err != nil {
			return sdk.WrapError(err, "unable to start tx")
		}

		if err := workflow.Delete(ctx, tx, store, &proj, &w); err != nil {
			log.Error("purge.Workflows> unable to delete workflow %d: %v", w.ID, err)
			_ = tx.Rollback()
			continue
		}

		if err := tx.Commit(); err != nil {
			log.Error("purge.Workflows> unable to commit tx: %v", err)
			_ = tx.Rollback()
			continue
		}
	}

	return nil
}

// deleteWorkflowRunsHistory is useful to delete all the workflow run marked with to delete flag in db
func deleteWorkflowRunsHistory(ctx context.Context, db gorp.SqlExecutor, workflowRunsDeleted *stats.Int64Measure) error {
	var ids []int64
	if _, err := db.Select(&ids, "SELECT id FROM workflow_run WHERE to_delete = true ORDER BY id ASC LIMIT 2000"); err != nil {
		return err
	}

	for _, id := range ids {
		res, err := db.Exec("DELETE FROM workflow_run WHERE workflow_run.id = $1", id)
		if err != nil {
			log.Error("deleteWorkflowRunsHistory> unable to delete workflow run %d: %v", id, err)
			continue
		}
		n, _ := res.RowsAffected()
		if workflowRunsDeleted != nil {
			observability.Record(ctx, workflowRunsDeleted, n)
		}
		time.Sleep(10 * time.Millisecond) // avoid DDOS the database
	}
	return nil
}
