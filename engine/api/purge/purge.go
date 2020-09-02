package purge

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-gorp/gorp"
	"go.opencensus.io/stats"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

//Initialize starts goroutines for workflows
func Initialize(ctx context.Context, store cache.Store, DBFunc func() *gorp.DbMap, sharedStorage objectstore.Driver, workflowRunsMarkToDelete, workflowRunsDeleted *stats.Int64Measure) {
	tickPurge := time.NewTicker(15 * time.Minute)
	defer tickPurge.Stop()

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Exiting purge: %v", ctx.Err())
				return
			}
		case <-tickPurge.C:
			// Check all workflows to mark runs that should be deleted
			if err := workflow.PurgeWorkflowRuns(ctx, DBFunc(), workflowRunsMarkToDelete); err != nil {
				log.Warning(ctx, "purge> Error: %v", err)
			}

			log.Debug("purge> Deleting all workflow run marked to delete...")
			if err := deleteWorkflowRunsHistory(ctx, DBFunc(), store, sharedStorage, workflowRunsDeleted); err != nil {
				log.Warning(ctx, "purge> Error on deleteWorkflowRunsHistory : %v", err)
			}

			log.Debug("purge> Deleting all workflow marked to delete....")
			if err := workflows(ctx, DBFunc(), store, workflowRunsMarkToDelete); err != nil {
				log.Warning(ctx, "purge> Error on workflows : %v", err)
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
	var projects = map[int64]sdk.ProjectIdentifiers{}

	for i, r := range res {
		// Force delete workflow runs if any
		_, err := workflow.PurgeAllWorkflowRunsByWorkflowID(ctx, db, r.ID)
		if err != nil {
			log.Error(ctx, "unable to mark workflow runs to delete with workflow_id %d: %v", r.ID, err)
			continue
		}
		workflow.CountWorkflowRunsMarkToDelete(ctx, db, workflowRunsMarkToDelete)

		// Checks if there is any workflow_runs
		nbWorkflowRuns, err := db.SelectInt("select count(1) from workflow_run where workflow_id = $1", r.ID)
		if err != nil {
			log.Error(ctx, "unable to count workflow runs for workflow_id %d: %v", r.ID, err)
			continue
		}
		if nbWorkflowRuns > 0 {
			log.Info(ctx, "skip workflow %d deletion because there are still %d workflow_runs to delete", r.ID, nbWorkflowRuns)
			continue
		}

		projIdent, has := projects[r.ProjectID]
		if !has {
			p, err := project.LoadByID(db, r.ProjectID)
			if err != nil {
				log.Error(ctx, "purge.Workflows> unable to load project %d: %v", r.ProjectID, err)
				continue
			}
			projects[r.ProjectID] = p.Identifiers()
			projIdent = projects[r.ProjectID]
		}

		w, err := workflow.LoadByID(ctx, db, projIdent, r.ID, workflow.LoadOptions{})
		if err != nil {
			log.Warning(ctx, "unable to load workflow %d due to error %v, we try to delete it", r.ID, err)
			if _, err := db.Exec("delete from w_node_trigger where child_node_id IN (SELECT id from w_node where workflow_id = $1)", r.ID); err != nil {
				log.Error(ctx, "Unable to delete from w_node_trigger for workflow %d: %v", r.ID, err)
			}
			if _, err := db.Exec("delete from w_node where workflow_id = $1", r.ID); err != nil {
				log.Error(ctx, "Unable to delete from w_node for workflow %d: %v", r.ID, err)
			}
			if _, err := db.Exec("delete from workflow where id = $1", r.ID); err != nil {
				log.Error(ctx, "Unable to delete from workflow with id %d: %v", r.ID, err)
			} else {
				log.Warning(ctx, "workflow with id %d is deleted", r.ID)
			}
			continue
		}
		wrkflws[i] = *w
	}

	for _, w := range wrkflws {
		projIdent, has := projects[w.ProjectID]
		if !has {
			continue
		}

		tx, err := db.Begin()
		if err != nil {
			return sdk.WrapError(err, "unable to start tx")
		}

		if err := workflow.Delete(ctx, tx, store, projIdent, &w); err != nil {
			log.Error(ctx, "purge.Workflows> unable to delete workflow %d: %v", w.ID, err)
			_ = tx.Rollback()
			continue
		}

		if err := tx.Commit(); err != nil {
			log.Error(ctx, "purge.Workflows> unable to commit tx: %v", err)
			_ = tx.Rollback()
			continue
		}
	}

	return nil
}

// deleteWorkflowRunsHistory is useful to delete all the workflow run marked with to delete flag in db
func deleteWorkflowRunsHistory(ctx context.Context, db gorp.SqlExecutor, store cache.Store, sharedStorage objectstore.Driver, workflowRunsDeleted *stats.Int64Measure) error {
	var workflowRunIDs []int64
	if _, err := db.Select(&workflowRunIDs, "SELECT id FROM workflow_run WHERE to_delete = true ORDER BY id ASC LIMIT 2000"); err != nil {
		return err
	}

	for _, workflowRunID := range workflowRunIDs {
		if err := DeleteArtifacts(ctx, db, store, sharedStorage, workflowRunID); err != nil {
			log.Error(ctx, "DeleteArtifacts> error while deleting artifacts: %v", err)
			continue
		}

		res, err := db.Exec("DELETE FROM workflow_run WHERE workflow_run.id = $1", workflowRunID)
		if err != nil {
			log.Error(ctx, "deleteWorkflowRunsHistory> unable to delete workflow run %d: %v", workflowRunID, err)
			continue
		}
		n, _ := res.RowsAffected()
		if workflowRunsDeleted != nil {
			telemetry.Record(ctx, workflowRunsDeleted, n)
		}
		time.Sleep(10 * time.Millisecond) // avoid DDOS the database
	}
	return nil
}

// DeleteArtifacts removes artifacts from storage
func DeleteArtifacts(ctx context.Context, db gorp.SqlExecutor, store cache.Store, sharedStorage objectstore.Driver, workflowRunID int64) error {
	wr, err := workflow.LoadRunByID(db, workflowRunID, workflow.LoadRunOptions{WithArtifacts: true, DisableDetailledNodeRun: false, WithDeleted: true})
	if err != nil {
		return sdk.WrapError(err, "error on load LoadRunByID:%d", workflowRunID)
	}

	proj, errprj := project.LoadProjectByWorkflowID(db, wr.WorkflowID)
	if errprj != nil {
		return sdk.WrapError(errprj, "error while load project for workflow %d", wr.WorkflowID)
	}

	type driversContainersT struct {
		projectKey      string
		integrationName string
		containerPath   string
	}

	driversContainers := []driversContainersT{}
	for _, wnrs := range wr.WorkflowNodeRuns {
		for _, wnr := range wnrs {
			for _, art := range wnr.Artifacts {
				var integrationName string
				if art.ProjectIntegrationID != nil && *art.ProjectIntegrationID > 0 {
					projectIntegration, err := integration.LoadProjectIntegrationByID(db, *art.ProjectIntegrationID)
					if err != nil {
						log.Error(ctx, "Cannot load LoadProjectIntegrationByID %s/%d", proj.Key, *art.ProjectIntegrationID)
						continue
					}
					integrationName = projectIntegration.Name
				} else {
					integrationName = sdk.DefaultStorageIntegrationName
				}

				var found bool
				for _, dc := range driversContainers {
					if dc.containerPath == art.GetPath() && proj.Key == dc.projectKey && integrationName == dc.integrationName {
						found = true
						break
					}
				}

				// container not found, add it to list to delete
				if !found {
					driversContainers = append(driversContainers, driversContainersT{
						projectKey:      proj.Key,
						integrationName: integrationName,
						containerPath:   art.GetPath(),
					})
				}

				storageDriver, err := objectstore.GetDriver(ctx, db, sharedStorage, proj.Key, integrationName)
				if err != nil {
					log.Error(ctx, "error while getting driver prj:%v integrationName:%v err:%v", proj.Key, integrationName, err)
					continue
				}

				log.Debug("DeleteArtifacts> deleting %+v", art)
				if err := storageDriver.Delete(ctx, &art); err != nil {
					log.Error(ctx, "error while deleting container prj:%v wnr:%v name:%v err:%v", proj.Key, wnr.ID, art.GetPath(), err)
					continue
				}
			}
		}
	}

	for _, dc := range driversContainers {
		storageDriver, err := objectstore.GetDriver(ctx, db, sharedStorage, dc.projectKey, dc.integrationName)
		if err != nil {
			log.Error(ctx, "error while getting driver prj:%v integrationName:%v err:%v", dc.projectKey, dc.integrationName, err)
			continue
		}

		if err := storageDriver.DeleteContainer(ctx, dc.containerPath); err != nil {
			log.Error(ctx, "error while deleting container prj:%v path:%v err:%v", dc.projectKey, dc.containerPath, err)
			continue
		}
	}

	return nil
}
