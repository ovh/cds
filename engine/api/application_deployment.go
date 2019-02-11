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
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getApplicationDeploymentStrategiesConfigHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		appName := vars["applicationName"]

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, application.LoadOptions.WithDeploymentStrategies)
		if err != nil {
			return sdk.WrapError(err, "getApplicationDeploymentStrategiesConfigHandler")
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

		tx, errtx := api.mustDB().Begin()
		if errtx != nil {
			return errtx
		}
		defer tx.Rollback()

		proj, err := project.Load(tx, api.Cache, key, deprecatedGetUser(ctx), project.LoadOptions.WithIntegrations)
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

		app, err := application.LoadByName(tx, api.Cache, key, appName, application.LoadOptions.WithClearDeploymentStrategies)
		if err != nil {
			return sdk.WrapError(err, "unable to load application")
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
			return sdk.WrapError(err, "postApplicationDeploymentStrategyConfigHandler")
		}

		app, err = application.LoadByName(tx, api.Cache, key, appName, application.LoadOptions.WithDeploymentStrategies)
		if err != nil {
			return sdk.WrapError(err, "unable to load application")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "unable to commit tx")
		}

		if getProvider(ctx) != nil {
			p := getProvider(ctx)
			log.Info("postApplicationDeploymentStrategyConfigHandler> application %s configuration successfully updated by provider %s", appName, *p)
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

		tx, errtx := api.mustDB().Begin()
		if errtx != nil {
			return errtx
		}
		defer tx.Rollback()

		proj, err := project.Load(tx, api.Cache, key, deprecatedGetUser(ctx), project.LoadOptions.WithIntegrations)
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
			return sdk.WrapError(sdk.ErrNotFound, "deleteApplicationDeploymentStrategyConfigHandler> integration not found on project")
		}

		if !pf.Model.Deployment {
			return sdk.WrapError(sdk.ErrForbidden, "deleteApplicationDeploymentStrategyConfigHandler> integration doesn't support deployment")
		}

		app, err := application.LoadByName(tx, api.Cache, key, appName, application.LoadOptions.WithDeploymentStrategies)
		if err != nil {
			return sdk.WrapError(err, "unable to load application")
		}

		isUsed, err := workflow.IsDeploymentIntegrationUsed(tx, proj.ID, app.ID, pfName)
		if err != nil {
			return sdk.WrapError(err, "unable to check if integration is used")
		}

		if isUsed {
			return sdk.NewError(sdk.ErrForbidden, fmt.Errorf("integration is still used in a workflow"))
		}

		if _, has := app.DeploymentStrategies[pfName]; !has {
			return sdk.WrapError(sdk.ErrNotFound, "deleteApplicationDeploymentStrategyConfigHandler> unable to find strategy")
		}

		delete(app.DeploymentStrategies, pfName)
		if err := application.DeleteDeploymentStrategy(tx, proj.ID, app.ID, pf.ID); err != nil {
			return sdk.WrapError(err, "deleteApplicationDeploymentStrategyConfigHandler")
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
		key := vars[permProjectKey]
		appName := vars["applicationName"]
		pfName := vars["integration"]
		withClearPassword := FormBool(r, "withClearPassword")

		opts := []application.LoadOptionFunc{
			application.LoadOptions.WithDeploymentStrategies,
		}
		if withClearPassword {
			opts = []application.LoadOptionFunc{application.LoadOptions.WithClearDeploymentStrategies}
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, opts...)
		if err != nil {
			return sdk.WrapError(err, "getApplicationDeploymentStrategyConfigHandler")
		}

		cfg, ok := app.DeploymentStrategies[pfName]
		if !ok {
			return sdk.ErrNotFound
		}

		return service.WriteJSON(w, cfg, http.StatusOK)
	}
}
