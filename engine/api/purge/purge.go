package purge

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//Initialize starts goroutines for workflows
func Initialize(c context.Context, store cache.Store, DBFunc func() *gorp.DbMap) {
	tickPurge := time.NewTicker(30 * time.Second)
	defer tickPurge.Stop()

	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Exiting purge: %v", c.Err())
				return
			}
		case <-tickPurge.C:
			log.Debug("purge> Deleting all workflow run marked to delete...")
			if err := RunsHistory(DBFunc()); err != nil {
				log.Warning("purge> Error : %v", err)
			}

			log.Debug("purge> Deleting all workflow marked to delete....")
			if err := Workflows(DBFunc(), store); err != nil {
				log.Warning("purge> Error : %v", err)
			}
		}
	}
}

// Workflows purges all marked workflows
func Workflows(db *gorp.DbMap, store cache.Store) error {
	query := "SELECT id, project_id FROM workflow WHERE to_delete = true ORDER BY id ASC"
	res := []struct {
		ID        int64 `db:"id"`
		ProjectID int64 `db:"project_id"`
	}{}

	if _, err := db.Select(&res, query); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return sdk.WrapError(err, "purge.Workflows> Unable to load workflows")
	}

	var wrkflws = make([]sdk.Workflow, len(res))
	var projects = map[int64]sdk.Project{}

	for i, r := range res {
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
			return sdk.WrapError(err, "purge.Workflows> unable to load workflow %d", r.ID)
		}
		wrkflws[i] = *w
	}

	for _, w := range wrkflws {
		tx, err := db.Begin()
		if err != nil {
			return sdk.WrapError(err, "purge.Workflows> unable to start tx")
		}

		proj, has := projects[w.ProjectID]
		if !has {
			continue
		}

		if err := workflow.Delete(tx, store, &proj, &w); err != nil {
			log.Error("purge.Workflows> unable to delete workflow %d: %v", w.ID, err)
			_ = tx.Rollback()
			continue
		}

		if err := tx.Commit(); err != nil {
			log.Error("purge.Workflows> unable to commit tx: %v", err)
			continue
		}
	}

	return nil
}

// RunsHistory is useful to delete all the workflow run marked with to delete flag in db
func RunsHistory(db gorp.SqlExecutor) error {
	query := `DELETE FROM workflow_run WHERE workflow_run.id IN (SELECT id FROM workflow_run WHERE to_delete = true LIMIT 30)`

	if _, err := db.Exec(query); err != nil {
		log.Warning("deleteWorkflowRunsHistory> Unable to delete workflow history %s", err)
		return err
	}
	return nil
}
