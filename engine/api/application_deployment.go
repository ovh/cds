package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getApplicationDeploymentStrategiesConfigHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		appName := vars["applicationName"]

		app, err := application.LoadByProjectKeyAndName(ctx, api.mustDB(), projectKey, appName,
			application.LoadOptions.WithDeploymentStrategies)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, app.DeploymentStrategies, http.StatusOK)
	}
}

func (api *API) postApplicationDeploymentStrategyConfigHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		appName := vars["applicationName"]
		pfName := vars["integration"]

		var pfConfig sdk.IntegrationConfig
		if err := service.UnmarshalBody(r, &pfConfig); err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback() // nolint

		proj, err := project.Load(ctx, tx, key, project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WrapError(err, "unable to load project")
		}

		var pf *sdk.ProjectIntegration
		for i := range proj.Integrations {
			if proj.Integrations[i].Name == pfName {
				pf = &proj.Integrations[i]
				break
			}
		}

		if pf == nil {
			return sdk.WrapError(sdk.ErrNotFound, "postApplicationDeploymentStrategyConfigHandler> integration not found on project")
		}

		if !pf.Model.Deployment {
			return sdk.WrapError(sdk.ErrForbidden, "postApplicationDeploymentStrategyConfigHandler> integration doesn't support deployment")
		}

		app, err := application.LoadByProjectKeyAndName(ctx, tx, proj.Key, appName,
			application.LoadOptions.WithClearDeploymentStrategies)
		if err != nil {
			return sdk.WrapError(err, "unable to load application")
		}
		if app.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		oldPfConfig, has := app.DeploymentStrategies[pfName]
		if !has {
			if pf.Model.DeploymentDefaultConfig != nil {
				oldPfConfig = pf.Model.DeploymentDefaultConfig
			} else {
				oldPfConfig = sdk.IntegrationConfig{}
			}
		}
		oldPfConfig.MergeWith(pfConfig)

		if err := application.SetDeploymentStrategy(tx, proj.ID, app.ID, pf.Model.ID, pfName, oldPfConfig); err != nil {
			return err
		}

		app, err = application.LoadByProjectKeyAndName(ctx, tx, proj.Key, appName,
			application.LoadOptions.WithDeploymentStrategies)
		if err != nil {
			return sdk.WrapError(err, "unable to load application")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "unable to commit tx")
		}

		return service.WriteJSON(w, app, http.StatusOK)
	}
}

func (api *API) deleteApplicationDeploymentStrategyConfigHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		appName := vars["applicationName"]
		pfName := vars["integration"]

		tx, err := api.mustDB().Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback() // nolint

		proj, err := project.Load(ctx, tx, key, project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WrapError(err, "unable to load project")
		}

		var pf *sdk.ProjectIntegration
		for i := range proj.Integrations {
			if proj.Integrations[i].Name == pfName {
				pf = &proj.Integrations[i]
				break
			}
		}

		if pf == nil {
			return sdk.WrapError(sdk.ErrNotFound, "integration not found on project")
		}

		if !pf.Model.Deployment {
			return sdk.WrapError(sdk.ErrForbidden, "integration doesn't support deployment")
		}

		app, err := application.LoadByProjectKeyAndName(ctx, tx, proj.Key, appName,
			application.LoadOptions.WithDeploymentStrategies)
		if err != nil {
			return sdk.WrapError(err, "unable to load application")
		}
		if app.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		isUsed, err := workflow.IsDeploymentIntegrationUsed(tx, proj.ID, app.ID, pfName)
		if err != nil {
			return sdk.WrapError(err, "unable to check if integration is used")
		}

		if isUsed {
			return sdk.NewError(sdk.ErrForbidden, fmt.Errorf("integration is still used in a workflow"))
		}

		if _, has := app.DeploymentStrategies[pfName]; !has {
			return sdk.WrapError(sdk.ErrNotFound, "unable to find strategy")
		}

		delete(app.DeploymentStrategies, pfName)
		if err := application.DeleteDeploymentStrategy(tx, proj.ID, app.ID, pf.ID); err != nil {
			return sdk.WithStack(err)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "unable to commit tx")
		}

		return service.WriteJSON(w, app, http.StatusOK)
	}
}

func (api *API) getApplicationDeploymentStrategyConfigHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		appName := vars["applicationName"]
		pfName := vars["integration"]
		withClearPassword := service.FormBool(r, "withClearPassword")

		opts := []application.LoadOptionFunc{
			application.LoadOptions.WithDeploymentStrategies,
		}
		if withClearPassword {
			opts = []application.LoadOptionFunc{application.LoadOptions.WithClearDeploymentStrategies}
		}

		app, err := application.LoadByProjectKeyAndName(ctx, api.mustDB(), projectKey, appName, opts...)
		if err != nil {
			return err
		}

		cfg, ok := app.DeploymentStrategies[pfName]
		if !ok {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		return service.WriteJSON(w, cfg, http.StatusOK)
	}
}
