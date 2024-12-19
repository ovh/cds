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

func (api *API) deleteProjectV2IntegrationHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManage),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			vars := mux.Vars(r)
			projectKey := vars["projectKey"]
			integrationName := vars["integrationName"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrUnauthorized)
			}

			p, err := project.Load(ctx, api.mustDB(), projectKey, project.LoadOptions.WithIntegrations)
			if err != nil {
				return sdk.WrapError(err, "cannot load project %s", projectKey)
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
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
						return sdk.WrapError(err, "cannot delete integration")
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
			event_v2.PublishProjectIntegrationEvent(ctx, api.Cache, sdk.EventIntegrationDeleted, projectKey, deletedIntegration, *u.AuthConsumerUser.AuthentifiedUser)
			return nil
		}
}

func (api *API) putProjectV2IntegrationHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManage),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			vars := mux.Vars(r)
			projectKey := vars["projectKey"]
			integrationName := vars["integrationName"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrUnauthorized)
			}

			var projectIntegration sdk.ProjectIntegration
			if err := service.UnmarshalBody(r, &projectIntegration); err != nil {
				return sdk.WrapError(err, "cannot read body")
			}

			p, err := project.Load(ctx, api.mustDB(), projectKey)
			if err != nil {
				return sdk.WrapError(err, "cannot load project %s", projectKey)
			}

			ppDB, err := integration.LoadProjectIntegrationByNameWithClearPassword(ctx, api.mustDB(), projectKey, integrationName)
			if err != nil {
				return sdk.WrapError(err, "cannot load integration %s for project %s", integrationName, projectKey)
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

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
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
				return sdk.WrapError(sdk.ErrWrongRequest, "model %d/%s not found", projectIntegration.Model.ID, projectIntegration.Model.Name)
			}

			if err := integration.UpdateIntegration(ctx, tx, projectIntegration); err != nil {
				return err
			}

			if projectIntegration.Model.Event {
				if err := event.ResetEventIntegration(ctx, tx, projectIntegration.ID); err != nil {
					return err
				}
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			event_v2.PublishProjectIntegrationEvent(ctx, api.Cache, sdk.EventIntegrationUpdated, projectKey, projectIntegration, *u.AuthConsumerUser.AuthentifiedUser)
			return service.WriteJSON(w, projectIntegration, http.StatusOK)
		}
}

func (api *API) getProjectV2IntegrationHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			vars := mux.Vars(r)
			projectKey := vars["projectKey"]
			integrationName := vars["integrationName"]

			integ, err := integration.LoadProjectIntegrationByName(ctx, api.mustDB(), projectKey, integrationName)
			if err != nil {
				return sdk.WrapError(err, "cannot load integration %s/%s", projectKey, integrationName)
			}

			plugins, err := plugin.LoadAllByIntegrationModelID(ctx, api.mustDB(), integ.IntegrationModelID)
			if err != nil {
				return sdk.WrapError(err, "cannot load integration plugin %s/%s", projectKey, integrationName)
			}
			integ.GRPCPlugins = plugins

			return service.WriteJSON(w, integ, http.StatusOK)
		}
}

func (api *API) postProjectV2IntegrationHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManage),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			vars := mux.Vars(r)
			projectKey := vars["projectKey"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrUnauthorized)
			}

			p, err := project.Load(ctx, api.mustDB(), projectKey, project.LoadOptions.WithIntegrations)
			if err != nil {
				return sdk.WrapError(err, "cannot load project %s", projectKey)
			}

			var pp sdk.ProjectIntegration
			if err := service.UnmarshalBody(r, &pp); err != nil {
				return sdk.WithStack(err)
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
				return sdk.WrapError(sdk.ErrWrongRequest, "model not found")
			}

			for _, pprojPlat := range p.Integrations {
				if pprojPlat.Name == pp.Name {
					if pprojPlat.Model.Public {
						return sdk.WithStack(sdk.ErrForbidden)
					}
					return sdk.WrapError(sdk.ErrWrongRequest, "integration %s already exist", pprojPlat.Name)
				}
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			if err := integration.InsertIntegration(tx, &pp); err != nil {
				return sdk.WrapError(err, "cannot insert integration")
			}

			if pp.Model.Event {
				if err := event.ResetEventIntegration(ctx, tx, pp.ID); err != nil {
					return sdk.WrapError(err, "cannot connect to event broker")
				}
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			event_v2.PublishProjectIntegrationEvent(ctx, api.Cache, sdk.EventIntegrationCreated, projectKey, pp, *u.AuthConsumerUser.AuthentifiedUser)
			return service.WriteJSON(w, pp, http.StatusOK)
		}
}

func (api *API) getProjectV2IntegrationsHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			vars := mux.Vars(r)
			projectKey := vars["projectKey"]

			p, errP := project.Load(ctx, api.mustDB(), projectKey, project.LoadOptions.WithIntegrations)
			if errP != nil {
				return sdk.WrapError(errP, "Cannot load project %s", projectKey)
			}
			return service.WriteJSON(w, p.Integrations, http.StatusOK)
		}
}
