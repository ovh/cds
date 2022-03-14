package purge

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"
	"go.opencensus.io/stats"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/integration/artifact_manager"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/featureflipping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

var defaultRunRetentionPolicy string
var disableDeletion bool

func SetPurgeConfiguration(rule string, disableDeletionConf bool) error {
	if rule == "" {
		return sdk.WithStack(fmt.Errorf("invalid empty rule for default workflow run retention policy"))
	}
	defaultRunRetentionPolicy = rule
	disableDeletion = disableDeletionConf
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
			log.Info(ctx, "purge> Start marking workflow run as delete")
			if err := markWorkflowRunsToDelete(ctx, store, DBFunc(), workflowRunsMarkToDelete); err != nil {
				ctx = sdk.ContextWithStacktrace(ctx, err)
				log.Error(ctx, "%v", err)
			}
		}
	}
}

// PurgeWorkflow deletes workflow runs marked as to delete
func WorkflowRuns(ctx context.Context, DBFunc func() *gorp.DbMap, workflowRunsMarkToDelete, workflowRunsDeleted *stats.Int64Measure) {
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

			if !disableDeletion {
				log.Info(ctx, "purge> Start deleting all workflow run marked to delete...")
				if err := deleteWorkflowRunsHistory(ctx, DBFunc(), workflowRunsDeleted); err != nil {
					log.Warn(ctx, "purge> Error on deleteWorkflowRunsHistory : %v", err)
				}
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
func deleteWorkflowRunsHistory(ctx context.Context, db *gorp.DbMap, workflowRunsDeleted *stats.Int64Measure) error {
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
			if err := deleteRunHistory(ctx, db, workflowRunID, cdnClient, workflowRunsDeleted); err != nil {
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

func deleteRunHistory(ctx context.Context, db *gorp.DbMap, workflowRunID int64, cdnClient services.Client, workflowRunsDeleted *stats.Int64Measure) error {
	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() // nolint

	wr, err := workflow.LoadAndLockRunByID(ctx, tx, workflowRunID, workflow.LoadRunOptions{DisableDetailledNodeRun: true, WithDeleted: true})
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return nil
		}
		return err
	}

	if err := DeleteArtifactsFromArtifactManager(ctx, tx, wr); err != nil {
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

func DeleteArtifactsFromArtifactManager(ctx context.Context, db gorp.SqlExecutor, wr *sdk.WorkflowRun) error {

	// Check if the run is linked to an artifact manager integration
	var artifactManagerInteg *sdk.WorkflowProjectIntegration
	for _, integ := range wr.Workflow.Integrations {
		log.Debug(ctx, "%+v", integ)

		if integ.ProjectIntegration.Model.ArtifactManager {
			artifactManagerInteg = &integ
			break
		}
	}
	if artifactManagerInteg == nil {
		log.Debug(ctx, "no artifactManagerInteg found")
		return nil
	}
	var (
		rtName = artifactManagerInteg.ProjectIntegration.Config[sdk.ArtifactoryConfigPlatform].Value
		rtURL  = artifactManagerInteg.ProjectIntegration.Config[sdk.ArtifactoryConfigURL].Value
	)

	// Load the token from secrets
	secrets, err := workflow.LoadDecryptSecrets(ctx, db, wr, nil)
	if err != nil {
		return err
	}

	var rtToken string
	for _, s := range secrets {
		if s.Name == fmt.Sprintf("cds.integration.artifact_manager.%s", sdk.ArtifactoryConfigToken) {
			rtToken = string(s.Value)
			break
		}
	}
	if rtToken == "" {
		return sdk.NewErrorFrom(sdk.ErrNotFound, "unable to find artifact manager token")
	}

	// Instanciate artifactory client
	artifactClient, err := artifact_manager.NewClient(rtName, rtURL, rtToken)
	if err != nil {
		return err
	}

	// Reload load result to mark them to delete in artifactory
	runResults, err := workflow.LoadRunResultsByRunID(ctx, db, wr.ID)
	if err != nil {
		return err
	}

	lowMaturity := artifactManagerInteg.ProjectIntegration.Config[sdk.ArtifactoryConfigPromotionLowMaturity].Value
	highMaturity := artifactManagerInteg.ProjectIntegration.Config[sdk.ArtifactoryConfigPromotionHighMaturity].Value

	toDeleteProperties := []sdk.KeyValues{
		{
			Key:    "ovh.to_delete",
			Values: []string{"true"},
		},
		{
			Key:    "ovh.to_delete_timestamp",
			Values: []string{strconv.FormatInt(time.Now().Unix(), 10)},
		},
	}

	for i := range runResults {
		res := &runResults[i]
		if res.Type == sdk.WorkflowRunResultTypeArtifactManager {
			art, err := res.GetArtifactManager()
			if err != nil {
				ctx := sdk.ContextWithStacktrace(ctx, err)
				log.Error(ctx, "unable to get artifact from run result %d: %v", res.ID, err)
				continue
			}
			if err := artifactClient.SetProperties(art.RepoName+"-"+lowMaturity, art.Path, toDeleteProperties...); err != nil {
				log.Info(ctx, "unable to mark artifact %q %q (run result %d) to delete: %v", art.RepoName+"-"+lowMaturity, art.Path, res.ID, err)
			} else {
				continue // if snaphot is a success, don't try to delete on release
			}
			if err := artifactClient.SetProperties(art.RepoName+"-"+highMaturity, art.Path, toDeleteProperties...); err != nil {
				log.Error(ctx, "unable to mark artifact %q %q (run result %d) to delete: %v", art.RepoName+"-"+highMaturity, art.Path, res.ID, err)
				continue
			}
		}
	}

	return nil
}
