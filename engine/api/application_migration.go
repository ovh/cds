package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/migrate"
	"github.com/ovh/cds/engine/api/project"
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

			appToClean.WorkflowMigration = migrate.STATUS_CLEANING
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

		force := FormBool(r, "force")
		disablePrefix := FormBool(r, "disablePrefix")
		withCurrentVersion := FormBool(r, "withCurrentVersion")
		withRepositoryWebHook := FormBool(r, "withRepositoryWebHook")

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

		var wfs []sdk.Workflow
		if len(cdTree) == 0 {
			app.WorkflowMigration = migrate.STATUS_CLEANING
		} else {
			var errM error
			wfs, errM = migrate.MigrateToWorkflow(tx, api.Cache, cdTree, p, getUser(ctx), force, disablePrefix, withCurrentVersion, withRepositoryWebHook)
			if errM != nil {
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

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectLastModificationType, sdk.ProjectWorkflowLastModificationType); err != nil {
			return sdk.WrapError(err, "migrationApplicationWorkflowHandler %s", sdk.ProjectWorkflowLastModificationType)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "migrationApplicationWorkflowHandler> Cannot commit transaction")
		}
		return WriteJSON(w, wfs, http.StatusOK)
	}
}
