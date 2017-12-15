package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/artifact"
	"github.com/ovh/cds/engine/api/migrate"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/poller"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/scheduler"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/engine/api/workflowv0"
	"github.com/ovh/cds/sdk"
)

func (api *API) migrationApplicationWorkflowCleanHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		applicationName := vars["permApplicationName"]

		p, errP := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx), project.LoadOptions.WithPipelines, project.LoadOptions.WithApplications, project.LoadOptions.WithEnvironments)
		if errP != nil {
			return sdk.WrapError(errP, "migrationApplicationWorkflowHandler")
		}

		cdTree, errT := workflowv0.LoadCDTree(api.mustDB(), api.Cache, projectKey, applicationName, getUser(ctx), "", "", 0)
		if errT != nil {
			return sdk.WrapError(errT, "migrationApplicationWorkflowCleanHandler")
		}

		appIDs := map[int64]bool{}
		for _, tree := range cdTree {
			getApplicationFromCDPipeline(tree, &appIDs)
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "migrationApplicationWorkflowHandler > Cannot start transaction")
		}
		defer tx.Rollback()

		for appID := range appIDs {
			appToClean, errA := application.LoadByID(api.mustDB(), api.Cache, appID, getUser(ctx), application.LoadOptions.WithPipelines)
			if errA != nil {
				return sdk.WrapError(errA, "migrationApplicationWorkflowHandler> Cannot load app")
			}

			for _, appPip := range appToClean.Pipelines {
				if err := trigger.DeletePipelineTriggers(tx, appPip.Pipeline.ID); err != nil {
					return sdk.WrapError(err, "migrationApplicationWorkflowHandler")
				}
				if err := application.DeleteAllApplicationPipeline(tx, appToClean.ID); err != nil {
					return sdk.WrapError(err, "migrationApplicationWorkflowHandler")
				}
				// Delete test results
				if err := pipeline.DeletePipelineTestResults(tx, appPip.Pipeline.ID); err != nil {
					return sdk.WrapError(err, "migrationApplicationWorkflowHandler")
				}
			}
			if err := scheduler.DeleteByApplicationID(tx, appID); err != nil {
				return sdk.WrapError(err, "migrationApplicationWorkflowHandler")
			}

			if err := poller.DeleteAll(tx, appID); err != nil {
				return sdk.WrapError(err, "migrationApplicationWorkflowHandler")
			}

			if err := artifact.DeleteArtifactsByApplicationID(tx, appID); err != nil {
				return sdk.WrapError(err, "migrationApplicationWorkflowHandler")
			}

			// Delete application_pipeline_notif
			query := `DELETE FROM application_pipeline_notif WHERE application_pipeline_id IN (SELECT id FROM application_pipeline WHERE application_id = $1)`
			if _, err := tx.Exec(query, appID); err != nil {
				return sdk.WrapError(err, "migrationApplicationWorkflowHandler> Delete notification")
			}

			if err := pipeline.DeletePipelineBuildByApplicationID(tx, appToClean.ID); err != nil {
				return sdk.WrapError(err, "migrationApplicationWorkflowHandler> DeletePipelineBuildByApplicationID")
			}

			appToClean.WorkflowMigration = migrate.STATUS_DONE
			appToClean.ProjectID = p.ID
			if err := application.Update(tx, api.Cache, appToClean, getUser(ctx)); err != nil {
				return sdk.WrapError(err, "migrationApplicationWorkflowHandler")
			}
		}

		p.WorkflowMigration = migrate.STATUS_START
		if err := project.Update(tx, api.Cache, p, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "migrationApplicationWorkflowHandler")
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectLastModificationType); err != nil {
			return sdk.WrapError(err, "migrationApplicationWorkflowHandler")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "migrationApplicationWorkflowHandler> Cannot commit transaction")
		}
		return nil
	}
}

func getApplicationFromCDPipeline(tree sdk.CDPipeline, appIDs *map[int64]bool) {
	(*appIDs)[tree.Application.ID] = true
	for _, cdtree := range tree.SubPipelines {
		getApplicationFromCDPipeline(cdtree, appIDs)
	}
}

func (api *API) migrationApplicationWorkflowHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		applicationName := vars["permApplicationName"]

		force := r.FormValue("force") == "true"

		p, errP := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx), project.LoadOptions.WithPipelines, project.LoadOptions.WithApplications, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups, project.LoadOptions.WithPermission)
		if errP != nil {
			return sdk.WrapError(errP, "migrationApplicationWorkflowHandler")
		}
		app, errA := application.LoadByName(api.mustDB(), api.Cache, projectKey, applicationName, getUser(ctx))
		if errA != nil {
			return sdk.WrapError(errA, "migrationApplicationWorkflowHandler")
		}

		cdTree, errW := workflowv0.LoadCDTree(api.mustDB(), api.Cache, projectKey, app.Name, getUser(ctx), "", "", 0)
		if errW != nil {
			return sdk.WrapError(errW, "migrationApplicationWorkflowHandler> Cannot load cd tree")
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "migrationApplicationWorkflowHandler > Cannot start transaction")
		}
		defer tx.Rollback()

		if len(cdTree) == 0 {
			app.WorkflowMigration = migrate.STATUS_DONE
		} else {
			if errM := migrate.MigrateToWorkflow(tx, api.Cache, cdTree, p, getUser(ctx), force); errM != nil {
				return sdk.WrapError(errM, "migrationApplicationWorkflowHandler")
			}
			app.WorkflowMigration = migrate.STATUS_START
		}

		if err := application.Update(tx, api.Cache, app, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "migrationApplicationWorkflowHandler")
		}

		p.WorkflowMigration = migrate.STATUS_START
		if err := project.Update(tx, api.Cache, p, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "migrationApplicationWorkflowHandler")
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectLastModificationType); err != nil {
			return sdk.WrapError(err, "migrationApplicationWorkflowHandler")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "migrationApplicationWorkflowHandler> Cannot commit transaction")
		}
		return nil
	}
}
