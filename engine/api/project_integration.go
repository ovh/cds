package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/event_v2"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/plugin"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getProjectIntegrationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKeyWithHooksAllowed"]
		integrationName := vars["integrationName"]

		var integ sdk.ProjectIntegration
		var err error

		clearPassword := service.FormBool(r, "clearPassword")
		if clearPassword {
			if !isHooks(ctx) && !isWorker(ctx) {
				return sdk.WithStack(sdk.ErrForbidden)
			}
			integ, err = integration.LoadProjectIntegrationByNameWithClearPassword(ctx, api.mustDB(), projectKey, integrationName)
			if err != nil {
				return sdk.WrapError(err, "Cannot load integration %s/%s", projectKey, integrationName)
			}
		} else {
			integ, err = integration.LoadProjectIntegrationByName(ctx, api.mustDB(), projectKey, integrationName)
			if err != nil {
				return sdk.WrapError(err, "Cannot load integration %s/%s", projectKey, integrationName)
			}
		}

		plugins, err := plugin.LoadAllByIntegrationModelID(ctx, api.mustDB(), integ.IntegrationModelID)
		if err != nil {
			return sdk.WrapError(err, "Cannot load integration plugin %s/%s", projectKey, integrationName)
		}
		integ.GRPCPlugins = plugins

		return service.WriteJSON(w, integ, http.StatusOK)
	}
}

func (api *API) putProjectIntegrationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKeyWithHooksAllowed"]
		integrationName := vars["integrationName"]

		u := getUserConsumer(ctx)
		if u == nil {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		var projectIntegration sdk.ProjectIntegration
		if err := service.UnmarshalBody(r, &projectIntegration); err != nil {
			return sdk.WrapError(err, "Cannot read body")
		}

		p, err := project.Load(ctx, api.mustDB(), projectKey)
		if err != nil {
			return sdk.WrapError(err, "Cannot load project")
		}

		ppDB, errP := integration.LoadProjectIntegrationByNameWithClearPassword(ctx, api.mustDB(), projectKey, integrationName)
		if errP != nil {
			return sdk.WrapError(errP, "putProjectIntegrationHandler> Cannot load integration %s for project %s", integrationName, projectKey)
		}

		//If the integration model is public, it's forbidden to update the integration
		if ppDB.Model.Public {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		projectIntegration.Name = ppDB.Name
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
			projectIntegration.Config[kkBody] = c
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "putProjectIntegrationHandler> Cannot start transaction")
		}
		defer tx.Rollback() // nolint

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

		if err := integration.UpdateIntegration(ctx, tx, projectIntegration); err != nil {
			return sdk.WrapError(err, "Cannot update integration")
		}

		if projectIntegration.Model.Event {
			if err := event.ResetEventIntegration(ctx, tx, projectIntegration.ID); err != nil {
				return sdk.WrapError(err, "cannot connect to event broker")
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		event.PublishUpdateProjectIntegration(ctx, p, projectIntegration, ppDB, getUserConsumer(ctx))
		event_v2.PublishProjectIntegrationEvent(ctx, api.Cache, sdk.EventIntegrationUpdated, p.Key, ppDB, *u.AuthConsumerUser.AuthentifiedUser)

		return service.WriteJSON(w, projectIntegration, http.StatusOK)
	}
}

func (api *API) deleteProjectIntegrationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKeyWithHooksAllowed"]
		integrationName := vars["integrationName"]

		u := getUserConsumer(ctx)
		if u == nil {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		p, err := project.Load(ctx, api.mustDB(), projectKey, project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WrapError(err, "Cannot load project")
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "deleteProjectIntegrationHandler> Cannot start transaction")
		}
		defer tx.Rollback() // nolint
		var deletedIntegration sdk.ProjectIntegration
		for _, plat := range p.Integrations {
			if plat.Name == integrationName {
				//If the integration model is public
				// it's forbidden to delete the integration if not admin
				if plat.Model.Public && !isAdmin(ctx) {
					return sdk.WithStack(sdk.ErrForbidden)
				}

				deletedIntegration = plat
				if err := integration.DeleteIntegration(tx, plat); err != nil {
					return sdk.WrapError(err, "Cannot delete integration")
				}
				break
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		if deletedIntegration.Model.Event {
			event.DeleteEventIntegration(deletedIntegration.ID)
		}
		event.PublishDeleteProjectIntegration(ctx, p, deletedIntegration, getUserConsumer(ctx))
		event_v2.PublishProjectIntegrationEvent(ctx, api.Cache, sdk.EventIntegrationDeleted, projectKey, deletedIntegration, *u.AuthConsumerUser.AuthentifiedUser)
		return nil
	}
}

func (api *API) getProjectIntegrationsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars[permProjectKey]

		p, errP := project.Load(ctx, api.mustDB(), projectKey, project.LoadOptions.WithIntegrations)
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

		u := getUserConsumer(ctx)
		if u == nil {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		p, err := project.Load(ctx, api.mustDB(), projectKey, project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WrapError(err, "Cannot load project")
		}

		var pp sdk.ProjectIntegration
		if err := service.UnmarshalBody(r, &pp); err != nil {
			return sdk.WrapError(err, "Cannot read body")
		}

		regexp := sdk.NamePatternRegex
		if !regexp.MatchString(pp.Name) {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "name %q do not respect pattern %s", pp.Name, sdk.NamePattern)
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
					return sdk.WithStack(sdk.ErrForbidden)
				}
				return sdk.WrapError(sdk.ErrWrongRequest, "postProjectIntegrationHandler> integration already exist")
			}
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "postProjectIntegrationHandler> Cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err := integration.InsertIntegration(tx, &pp); err != nil {
			return sdk.WrapError(err, "Cannot insert integration")
		}

		if pp.Model.Event {
			if err := event.ResetEventIntegration(ctx, tx, pp.ID); err != nil {
				return sdk.WrapError(err, "cannot connect to event broker")
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		event.PublishAddProjectIntegration(ctx, p, pp, getUserConsumer(ctx))
		event_v2.PublishProjectIntegrationEvent(ctx, api.Cache, sdk.EventIntegrationCreated, projectKey, pp, *u.AuthConsumerUser.AuthentifiedUser)

		return service.WriteJSON(w, pp, http.StatusOK)
	}
}
