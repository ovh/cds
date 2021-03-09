package purge

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"
	"go.opencensus.io/stats"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/featureflipping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

var defaultRunRetentionPolicy string

func SetDefaultRunRetentionPolicy(rule string) error {
	if rule == "" {
		return sdk.WithStack(fmt.Errorf("invalid empty rule for default workflow run retention policy"))
	}
	defaultRunRetentionPolicy = rule
	return nil
}

// MarkRunsAsDelete mark workflow run as delete
func MarkRunsAsDelete(ctx context.Context, store cache.Store, DBFunc func() *gorp.DbMap, workflowRunsMarkToDelete *stats.Int64Measure) {
	tickMark := time.NewTicker(15 * time.Minute)
	defer tickMark.Stop()

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Exiting mark runs as delete: %v", ctx.Err())
				return
			}
		case <-tickMark.C:
			// Mark workflow run to delete
			if err := markWorkflowRunsToDelete(ctx, store, DBFunc(), workflowRunsMarkToDelete); err != nil {
				ctx = sdk.ContextWithStacktrace(ctx, err)
				log.Error(ctx, "%v", err)
			}
		}
	}
}

// PurgeWorkflow deletes workflow runs marked as to delete
func WorkflowRuns(ctx context.Context, DBFunc func() *gorp.DbMap, sharedStorage objectstore.Driver, workflowRunsMarkToDelete, workflowRunsDeleted *stats.Int64Measure) {
	tickPurge := time.NewTicker(15 * time.Minute)
	defer tickPurge.Stop()

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Exiting purge workflow runs: %v", ctx.Err())
				return
			}
		case <-tickPurge.C:
			// Check all workflows to mark runs that should be deleted
			if err := MarkWorkflowRuns(ctx, DBFunc(), workflowRunsMarkToDelete); err != nil {
				log.Warn(ctx, "purge> Error: %v", err)
			}

			log.Debug(ctx, "purge> Deleting all workflow run marked to delete...")
			if err := deleteWorkflowRunsHistory(ctx, DBFunc(), sharedStorage, workflowRunsDeleted); err != nil {
				log.Warn(ctx, "purge> Error on deleteWorkflowRunsHistory : %v", err)
			}
		}
	}
}

// Workflow deletes workflows marked as to delete
func Workflow(ctx context.Context, store cache.Store, DBFunc func() *gorp.DbMap, workflowRunsMarkToDelete *stats.Int64Measure) {
	tickPurge := time.NewTicker(15 * time.Minute)
	defer tickPurge.Stop()

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Exiting purge workflow: %v", ctx.Err())
				return
			}
		case <-tickPurge.C:
			log.Debug(ctx, "purge> Deleting all workflow marked to delete....")
			if err := workflows(ctx, DBFunc(), store, workflowRunsMarkToDelete); err != nil {
				log.Warn(ctx, "purge> Error on workflows : %v", err)
			}
		}
	}
}

// MarkWorkflowRuns Deprecated: old method to mark runs to delete
func MarkWorkflowRuns(ctx context.Context, db *gorp.DbMap, workflowRunsMarkToDelete *stats.Int64Measure) error {
	dao := new(workflow.WorkflowDAO)
	dao.Filters.DisableFilterDeletedWorkflow = false
	wfs, err := dao.LoadAll(ctx, db)
	if err != nil {
		return err
	}
	for _, wf := range wfs {
		_, enabled := featureflipping.IsEnabled(ctx, gorpmapping.Mapper, db, sdk.FeaturePurgeName, map[string]string{"project_key": wf.ProjectKey})
		if enabled {
			continue
		}
		tx, err := db.Begin()
		if err != nil {
			log.Error(ctx, "workflow.PurgeWorkflowRuns> error %v", err)
			tx.Rollback() // nolint
			continue
		}
		if err := workflow.PurgeWorkflowRun(ctx, tx, wf); err != nil {
			log.Error(ctx, "workflow.PurgeWorkflowRuns> error %v", err)
			tx.Rollback() // nolint
			continue
		}
		if err := tx.Commit(); err != nil {
			log.Error(ctx, "workflow.PurgeWorkflowRuns> unable to commit transaction:  %v", err)
			_ = tx.Rollback()
			continue
		}
	}

	workflow.CountWorkflowRunsMarkToDelete(ctx, db, workflowRunsMarkToDelete)
	return nil
}

// workflows purges all marked workflows
func workflows(ctx context.Context, db *gorp.DbMap, store cache.Store, workflowRunsMarkToDelete *stats.Int64Measure) error {
	query := "SELECT id, project_id FROM workflow WHERE to_delete = true ORDER BY id ASC"
	var res []struct {
		ID        int64 `db:"id"`
		ProjectID int64 `db:"project_id"`
	}

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

		proj, has := projects[r.ProjectID]
		if !has {
			p, err := project.LoadByID(db, r.ProjectID)
			if err != nil {
				log.Error(ctx, "purge.Workflows> unable to load project %d: %v", r.ProjectID, err)
				continue
			}
			projects[r.ProjectID] = *p
			proj = *p
		}

		w, err := workflow.LoadByID(ctx, db, store, proj, r.ID, workflow.LoadOptions{})
		if err != nil {
			log.Warn(ctx, "unable to load workflow %d due to error %v, we try to delete it", r.ID, err)
			if _, err := db.Exec("delete from w_node_trigger where child_node_id IN (SELECT id from w_node where workflow_id = $1)", r.ID); err != nil {
				log.Error(ctx, "Unable to delete from w_node_trigger for workflow %d: %v", r.ID, err)
			}
			if _, err := db.Exec("delete from w_node where workflow_id = $1", r.ID); err != nil {
				log.Error(ctx, "Unable to delete from w_node for workflow %d: %v", r.ID, err)
			}
			if _, err := db.Exec("delete from workflow where id = $1", r.ID); err != nil {
				log.Error(ctx, "Unable to delete from workflow with id %d: %v", r.ID, err)
			} else {
				log.Warn(ctx, "workflow with id %d is deleted", r.ID)
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

		if err := workflow.Delete(ctx, tx, store, proj, &w); err != nil {
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
func deleteWorkflowRunsHistory(ctx context.Context, db *gorp.DbMap, sharedStorage objectstore.Driver, workflowRunsDeleted *stats.Int64Measure) error {
	//Load service "CDN"
	srvs, err := services.LoadAllByType(ctx, db, sdk.TypeCDN)
	if err != nil {
		return err
	}
	cdnClient := services.NewClient(db, srvs)

	limit := int64(2000)
	offset := int64(0)
	for {
		workflowRunIDs, _, _, count, err := workflow.LoadRunsIDsToDelete(db, offset, limit)
		if err != nil {
			return err
		}

		for _, workflowRunID := range workflowRunIDs {
			if err := deleteRunHistory(ctx, db, workflowRunID, cdnClient, sharedStorage, workflowRunsDeleted); err != nil {
				log.Error(ctx, "unable to delete run history: %v", err)
			}
			time.Sleep(10 * time.Millisecond) // avoid DDOS the database
		}

		if count > offset+limit {
			offset += limit
			continue
		}
		break
	}
	return nil
}

func deleteRunHistory(ctx context.Context, db *gorp.DbMap, workflowRunID int64, cdnClient services.Client, sharedStorage objectstore.Driver, workflowRunsDeleted *stats.Int64Measure) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	if _, err := workflow.LoadAndLockRunByID(tx, workflowRunID, workflow.LoadRunOptions{DisableDetailledNodeRun: true, WithDeleted: true}); err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil
		}
		return err
	}

	if err := DeleteArtifacts(ctx, tx, sharedStorage, workflowRunID); err != nil {
		return sdk.WithStack(err)
	}

	res, err := tx.Exec("DELETE FROM workflow_run WHERE workflow_run.id = $1", workflowRunID)
	if err != nil {
		return sdk.WithStack(err)
	}

	_, code, err := cdnClient.DoJSONRequest(ctx, http.MethodPost, "/bulk/item/delete", sdk.CDNMarkDelete{RunID: workflowRunID}, nil)
	if err != nil || code >= 400 {
		return sdk.WithStack(err)
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	n, _ := res.RowsAffected()
	if workflowRunsDeleted != nil {
		telemetry.Record(ctx, workflowRunsDeleted, n)
	}
	return nil
}

// DeleteArtifacts removes artifacts from storage
func DeleteArtifacts(ctx context.Context, db gorp.SqlExecutor, sharedStorage objectstore.Driver, workflowRunID int64) error {
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

				log.Debug(ctx, "DeleteArtifacts> deleting %+v", art)
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
