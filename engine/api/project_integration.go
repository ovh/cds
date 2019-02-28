package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/plugin"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getProjectIntegrationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		integrationName := vars["integrationName"]

		clearPassword := FormBool(r, "clearPassword")

		integration, err := integration.LoadProjectIntegrationByName(api.mustDB(), projectKey, integrationName, clearPassword)
		if err != nil {
			return sdk.WrapError(err, "Cannot load integration %s/%s", projectKey, integrationName)
		}
		plugins, err := plugin.LoadAllByIntegrationModelID(api.mustDB(), integration.IntegrationModelID)
		if err != nil {
			return sdk.WrapError(err, "Cannot load integration %s/%s", projectKey, integrationName)
		}
		integration.GRPCPlugins = plugins
		return service.WriteJSON(w, integration, http.StatusOK)
	}
}

func (api *API) putProjectIntegrationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		integrationName := vars["integrationName"]

		var projectIntegration sdk.ProjectIntegration
		if err := service.UnmarshalBody(r, &projectIntegration); err != nil {
			return sdk.WrapError(err, "Cannot read body")
		}

		p, err := project.Load(api.mustDB(), api.Cache, projectKey, deprecatedGetUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "Cannot load project")
		}

		ppDB, errP := integration.LoadProjectIntegrationByName(api.mustDB(), projectKey, integrationName, true)
		if errP != nil {
			return sdk.WrapError(errP, "putProjectIntegrationHandler> Cannot load integration %s for project %s", integrationName, projectKey)
		}

		//If the integration model is public, it's forbidden to update the integration
		if ppDB.Model.Public {
			return sdk.ErrForbidden
		}

		projectIntegration.ID = ppDB.ID

		for kkBody := range projectIntegration.Config {
			c := projectIntegration.Config[kkBody]
			// if we received a placeholder, replace with the right value
			if c.Type == sdk.IntegrationConfigTypePassword && c.Value == sdk.PasswordPlaceholder {
				for kkDB, ccDB := range ppDB.Config {
					if kkDB == kkBody {
						c.Value = ccDB.Value
						break
					}
				}
			}
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "putProjectIntegrationHandler> Cannot strat transaction")
		}
		defer tx.Rollback()

		projectIntegration.ProjectID = p.ID
		if projectIntegration.IntegrationModelID == 0 {
			projectIntegration.IntegrationModelID = projectIntegration.Model.ID
		}
		if projectIntegration.IntegrationModelID == 0 && projectIntegration.Model.Name != "" {
			pfs, err := integration.LoadModels(api.mustDB())
			if err != nil {
				return err
			}
			for _, pf := range pfs {
				if pf.Name == projectIntegration.Model.Name {
					projectIntegration.IntegrationModelID = pf.ID
					break
				}
			}
		}

		if projectIntegration.IntegrationModelID == 0 {
			return sdk.WrapError(sdk.ErrWrongRequest, "postProjectIntegrationHandler> model not found")
		}

		if err := integration.UpdateIntegration(tx, projectIntegration); err != nil {
			return sdk.WrapError(err, "Cannot update integration")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishUpdateProjectIntegration(p, projectIntegration, ppDB, deprecatedGetUser(ctx))

		return service.WriteJSON(w, projectIntegration, http.StatusOK)
	}
}

func (api *API) deleteProjectIntegrationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]
		integrationName := vars["integrationName"]

		p, err := project.Load(api.mustDB(), api.Cache, projectKey, deprecatedGetUser(ctx), project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WrapError(err, "Cannot load project")
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "deleteProjectIntegrationHandler> Cannot start transaction")
		}
		defer tx.Rollback()
		var deletedIntegration sdk.ProjectIntegration
		for _, plat := range p.Integrations {
			if plat.Name == integrationName {
				//If the integration model is public, it's forbidden to delete the integration
				if plat.Model.Public {
					return sdk.ErrForbidden
				}

				deletedIntegration = plat
				if err := integration.DeleteIntegration(tx, plat); err != nil {
					return sdk.WrapError(err, "Cannot delete integration")
				}
				break
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishDeleteProjectIntegration(p, deletedIntegration, deprecatedGetUser(ctx))
		return nil
	}
}

func (api *API) getProjectIntegrationsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]

		p, errP := project.Load(api.mustDB(), api.Cache, projectKey, deprecatedGetUser(ctx), project.LoadOptions.WithIntegrations)
		if errP != nil {
			return sdk.WrapError(errP, "getProjectIntegrationsHandler> Cannot load project")
		}
		return service.WriteJSON(w, p.Integrations, http.StatusOK)
	}
}

func (api *API) postProjectIntegrationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]

		p, err := project.Load(api.mustDB(), api.Cache, projectKey, deprecatedGetUser(ctx), project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WrapError(err, "Cannot load project")
		}

		var pp sdk.ProjectIntegration
		if err := service.UnmarshalBody(r, &pp); err != nil {
			return sdk.WrapError(err, "Cannot read body")
		}

		pp.ProjectID = p.ID
		if pp.IntegrationModelID == 0 {
			pp.IntegrationModelID = pp.Model.ID
		}
		if pp.IntegrationModelID == 0 && pp.Model.Name != "" {
			pfs, err := integration.LoadModels(api.mustDB())
			if err != nil {
				return err
			}
			for _, pf := range pfs {
				if pf.Name == pp.Model.Name {
					pp.IntegrationModelID = pf.ID
					break
				}
			}
		}

		if pp.IntegrationModelID == 0 {
			return sdk.WrapError(sdk.ErrWrongRequest, "postProjectIntegrationHandler> model not found")
		}

		for _, pprojPlat := range p.Integrations {
			if pprojPlat.Name == pp.Name {
				if pprojPlat.Model.Public {
					return sdk.ErrForbidden
				}
				return sdk.WrapError(sdk.ErrWrongRequest, "postProjectIntegrationHandler> integration already exist")
			}
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "postProjectIntegrationHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := integration.InsertIntegration(tx, &pp); err != nil {
			return sdk.WrapError(err, "Cannot insert integration")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishAddProjectIntegration(p, pp, deprecatedGetUser(ctx))

		return service.WriteJSON(w, pp, http.StatusOK)
	}
}
