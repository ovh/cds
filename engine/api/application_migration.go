package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/event"
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
		u := getUser(ctx)

		p, errP := project.Load(api.mustDB(), api.Cache, projectKey, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithApplications, project.LoadOptions.WithEnvironments)
		if errP != nil {
			return sdk.WrapError(errP, "migrationApplicationWorkflowHandler")
		}
		oldProj := *p

		cdTree, errT := workflowv0.LoadCDTree(api.mustDB(), api.Cache, projectKey, applicationName, u, "", "", 0)
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
			appToClean, errA := application.LoadByID(api.mustDB(), api.Cache, appID, u, application.LoadOptions.WithPipelines)
			if errA != nil {
				return sdk.WrapError(errA, "migrationApplicationWorkflowHandler> Cannot load app")
			}

			appToClean.WorkflowMigration = migrate.STATUS_CLEANING
			appToClean.ProjectID = p.ID
			if err := application.Update(tx, api.Cache, appToClean, u); err != nil {
				return sdk.WrapError(err, "migrationApplicationWorkflowHandler")
			}
		}

		p.WorkflowMigration = migrate.STATUS_START
		if err := project.Update(tx, api.Cache, p, u); err != nil {
			return sdk.WrapError(err, "migrationApplicationWorkflowHandler")
		}
		event.PublishUpdateProject(p, &oldProj, u)

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
		u := getUser(ctx)
		vars := mux.Vars(r)
		projectKey := vars["key"]
		applicationName := vars["permApplicationName"]

		force := FormBool(r, "force")
		disablePrefix := FormBool(r, "disablePrefix")
		withCurrentVersion := FormBool(r, "withCurrentVersion")
		withRepositoryWebHook := FormBool(r, "withRepositoryWebHook")

		p, errP := project.Load(api.mustDB(), api.Cache, projectKey, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithApplications, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups, project.LoadOptions.WithPermission)
		if errP != nil {
			return sdk.WrapError(errP, "migrationApplicationWorkflowHandler")
		}
		oldProj := *p
		app, errA := application.LoadByName(api.mustDB(), api.Cache, projectKey, applicationName, u)
		if errA != nil {
			return sdk.WrapError(errA, "migrationApplicationWorkflowHandler")
		}

		cdTree, errW := workflowv0.LoadCDTree(api.mustDB(), api.Cache, projectKey, app.Name, u, "", "", 0)
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
			wfs, errM = migrate.ToWorkflow(tx, api.Cache, cdTree, p, u, force, disablePrefix, withCurrentVersion, withRepositoryWebHook)
			if errM != nil {
				return sdk.WrapError(errM, "migrationApplicationWorkflowHandler")
			}
			app.WorkflowMigration = migrate.STATUS_START
		}

		if err := application.Update(tx, api.Cache, app, u); err != nil {
			return sdk.WrapError(err, "migrationApplicationWorkflowHandler")
		}

		p.WorkflowMigration = migrate.STATUS_START
		if err := project.Update(tx, api.Cache, p, u); err != nil {
			return sdk.WrapError(err, "migrationApplicationWorkflowHandler")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "migrationApplicationWorkflowHandler> Cannot commit transaction")
		}
		event.PublishUpdateProject(p, &oldProj, u)

		return WriteJSON(w, wfs, http.StatusOK)
	}
}
