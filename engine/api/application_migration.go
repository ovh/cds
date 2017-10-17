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

func (api *API) migrationApplicationWorkflowHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		applicationName := vars["permApplicationName"]

		force := r.FormValue("force") == "true"

		p, errP := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx), project.LoadOptions.WithPipelines, project.LoadOptions.WithApplications, project.LoadOptions.WithEnvironments)
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

		if errM := migrate.MigrateToWorkflow(tx, api.Cache, cdTree, p, getUser(ctx), force); errM != nil {
			return sdk.WrapError(errM, "migrationApplicationWorkflowHandler")
		}
		app.WorkflowMigration = migrate.STATUS_START
		if err := application.Update(tx, api.Cache, app, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "migrationApplicationWorkflowHandler")
		}

		p.WorkflowMigration = migrate.STATUS_START
		if err := project.Update(tx, api.Cache, p, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "migrationApplicationWorkflowHandler")
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p); err != nil {
			return sdk.WrapError(err, "migrationApplicationWorkflowHandler")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "migrationApplicationWorkflowHandler> Cannot commit transaction")
		}
		return nil
	}
}
