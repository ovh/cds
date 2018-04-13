package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
)

func (api *API) getApplicationDeploymentStrategiesConfigHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx), application.LoadOptions.WithDeploymentStrategies)
		if err != nil {
			return sdk.WrapError(err, "getApplicationDeploymentStrategiesConfigHandler")
		}

		return WriteJSON(w, app.DeploymentStrategies, http.StatusOK)
	}
}

func (api *API) postApplicationDeploymentStrategyConfigHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]
		pfName := vars["platform"]

		var pfConfig sdk.PlatformConfig
		if err := UnmarshalBody(r, &pfConfig); err != nil {
			return err
		}

		tx, errtx := api.mustDB().Begin()
		if errtx != nil {
			return errtx
		}
		defer tx.Rollback()

		proj, err := project.Load(tx, api.Cache, key, getUser(ctx), project.LoadOptions.WithPlatforms)
		if err != nil {
			return sdk.WrapError(err, "postApplicationDeploymentStrategyConfigHandler> unable to load project")
		}

		var pf *sdk.ProjectPlatform
		for i := range proj.Platforms {
			if proj.Platforms[i].Name == pfName {
				pf = &proj.Platforms[i]
				break
			}
		}

		if pf == nil {
			return sdk.WrapError(sdk.ErrNotFound, "postApplicationDeploymentStrategyConfigHandler> platform not found on project")
		}

		if !pf.Model.Deployment {
			return sdk.WrapError(sdk.ErrForbidden, "postApplicationDeploymentStrategyConfigHandler> platform doesn't support deployment")
		}

		app, err := application.LoadByName(tx, api.Cache, key, appName, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "postApplicationDeploymentStrategyConfigHandler> unable to load application")
		}

		if err := application.SetDeploymentStrategies(tx, proj.ID, app.ID, pf.Model.ID, pfConfig); err != nil {
			return sdk.WrapError(err, "postApplicationDeploymentStrategyConfigHandler")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "postApplicationDeploymentStrategyConfigHandler> unable to commit tx")
		}

		w.WriteHeader(http.StatusOK)
		return nil
	}
}

func (api *API) getApplicationDeploymentStrategyConfigHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]
		pfName := vars["platform"]
		withClearPassword := FormBool(r, "withClearPassword")

		opts := []application.LoadOptionFunc{
			application.LoadOptions.WithDeploymentStrategies,
		}
		if withClearPassword {
			opts = []application.LoadOptionFunc{application.LoadOptions.WithClearDeploymentStrategies}
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx), opts...)
		if err != nil {
			return sdk.WrapError(err, "getApplicationDeploymentStrategyConfigHandler")
		}

		cfg, ok := app.DeploymentStrategies[pfName]
		if !ok {
			return sdk.ErrNotFound
		}

		return WriteJSON(w, cfg, http.StatusOK)
	}
}
