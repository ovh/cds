package purge

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
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
	tickPurge := time.NewTicker(30 * time.Minute)
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
			if err := deleteWorkflowRunsHistory(DBFunc()); err != nil {
				log.Warning("purge> Error on deleteWorkflowRunsHistory : %v", err)
			}

			log.Debug("purge> Deleting all workflow marked to delete....")
			if err := Workflows(c, DBFunc(), store); err != nil {
				log.Warning("purge> Error on workflows : %v", err)
			}

			if err := stopRunsBlocked(DBFunc()); err != nil {
				log.Warning("purge> Error on stopRunsBlocked : %v", err)
			}
		}
	}
}

// Workflows purges all marked workflows
func Workflows(ctx context.Context, db *gorp.DbMap, store cache.Store) error {
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
		proj, has := projects[w.ProjectID]
		if !has {
			continue
		}

		tx, err := db.Begin()
		if err != nil {
			return sdk.WrapError(err, "purge.Workflows> unable to start tx")
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
func deleteWorkflowRunsHistory(db gorp.SqlExecutor) error {
	query := `DELETE FROM workflow_run WHERE workflow_run.id IN (SELECT id FROM workflow_run WHERE to_delete = true LIMIT 30)`

	if _, err := db.Exec(query); err != nil {
		log.Warning("deleteWorkflowRunsHistory> Unable to delete workflow history %s", err)
		return err
	}
	return nil
}

// stopRunsBlocked is useful to force stop all workflow that is running more than 24hrs
func stopRunsBlocked(db *gorp.DbMap) error {
	query := `SELECT workflow_run.id
		FROM workflow_run
		WHERE (workflow_run.status = $1 or workflow_run.status = $2 or workflow_run.status = $3)
		AND now() - workflow_run.start > interval '1 day'
		LIMIT 30`
	ids := []struct {
		ID int64 `db:"id"`
	}{}

	if _, err := db.Select(&ids, query, sdk.StatusWaiting.String(), sdk.StatusChecking.String(), sdk.StatusBuilding.String()); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return sdk.WrapError(err, "stopRunsBlocked>")
	}

	tx, errTx := db.Begin()
	if errTx != nil {
		return sdk.WrapError(errTx, "stopRunsBlocked>")
	}
	defer tx.Rollback() // nolint

	wfIds := make([]string, len(ids))
	for i := range wfIds {
		wfIds[i] = fmt.Sprintf("%d", ids[i].ID)
	}
	wfIdsJoined := strings.Join(wfIds, ",")
	queryUpdateWf := `UPDATE workflow_run SET status = $1 WHERE id = ANY(string_to_array($2, ',')::bigint[])`
	if _, err := tx.Exec(queryUpdateWf, sdk.StatusStopped.String(), wfIdsJoined); err != nil {
		return sdk.WrapError(err, "stopRunsBlocked> Unable to stop workflow run history")
	}
	args := []interface{}{sdk.StatusStopped.String(), wfIdsJoined, sdk.StatusBuilding.String(), sdk.StatusChecking.String(), sdk.StatusWaiting.String()}
	queryUpdateNodeRun := `UPDATE workflow_node_run SET status = $1, done = now()
	WHERE workflow_run_id = ANY(string_to_array($2, ',')::bigint[])
	AND (status = $3 OR status = $4 OR status = $5)`
	if _, err := tx.Exec(queryUpdateNodeRun, args...); err != nil {
		return sdk.WrapError(err, "stopRunsBlocked> Unable to stop workflow node run history")
	}
	queryUpdateNodeJobRun := `UPDATE workflow_node_run_job SET status = $1, done = now()
	WHERE workflow_node_run_job.workflow_node_run_id IN (
		SELECT workflow_node_run.id
		FROM workflow_node_run
		WHERE workflow_node_run.workflow_run_id = ANY(string_to_array($2, ',')::bigint[])
		AND (status = $3 OR status = $4 OR status = $5)
	)`
	if _, err := tx.Exec(queryUpdateNodeJobRun, args...); err != nil {
		return sdk.WrapError(err, "stopRunsBlocked> Unable to stop workflow node job run history")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "stopRunsBlocked> Unable to commit transaction")
	}
	return nil
}
