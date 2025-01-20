package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/event_v2"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getProjectsV2Handler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			// For admin
			if isAdmin(ctx) {
				projects, err := project.LoadAll(ctx, api.mustDB(), api.Cache)
				if err != nil {
					return err
				}
				return service.WriteJSON(w, projects, http.StatusOK)
			}

			// Normal user
			keys, err := rbac.LoadAllProjectKeysAllowed(ctx, api.mustDB(), sdk.ProjectRoleRead, u.AuthConsumerUser.AuthentifiedUserID)
			if err != nil {
				return err
			}

			projects, err := project.LoadAllByKeys(ctx, api.mustDB(), keys)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, projects, http.StatusOK)
		}
}

func (api *API) deleteProjectV2Handler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManage),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			// Get project name in URL
			vars := mux.Vars(r)
			key := vars["projectKey"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			p, err := project.Load(ctx, api.mustDB(), key, project.LoadOptions.WithPipelines, project.LoadOptions.WithApplications)
			if err != nil {
				if !sdk.ErrorIs(err, sdk.ErrNoProject) {
					return sdk.WrapError(err, "deleteProject> load project '%s' from db", key)
				}
				return sdk.WrapError(err, "cannot load project %s", key)
			}

			// TODO Delete
			if len(p.Pipelines) > 0 {
				return sdk.WrapError(sdk.ErrProjectHasPipeline, "project '%s' still used by %d pipelines", key, len(p.Pipelines))
			}

			if len(p.Applications) > 0 {
				return sdk.WrapError(sdk.ErrProjectHasApplication, "project '%s' still used by %d applications", key, len(p.Applications))
			}
			//

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			if err := project.Delete(tx, p.Key); err != nil {
				return err
			}
			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			event_v2.PublishProjectEvent(ctx, api.Cache, sdk.EventProjectDeleted, *p, *u.AuthConsumerUser.AuthentifiedUser)
			return nil
		}
}

func (api *API) updateProjectV2Handler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			// Get project name in URL
			vars := mux.Vars(r)
			key := vars["projectKey"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			proj := &sdk.Project{}
			if err := service.UnmarshalBody(r, proj); err != nil {
				return sdk.WithStack(err)
			}

			if proj.Name == "" {
				return sdk.WrapError(sdk.ErrInvalidProjectName, "project name must no be empty")
			}

			// Check Request
			if key != proj.Key {
				return sdk.WrapError(sdk.ErrWrongRequest, "bad Project key %s/%s ", key, proj.Key)
			}

			if proj.WorkflowRetention <= 0 {
				proj.WorkflowRetention = api.Config.WorkflowV2.WorkflowRunRetention
			}

			// Check is project exist
			p, err := project.Load(ctx, api.mustDB(), key, project.LoadOptions.WithIcon)
			if err != nil {
				return err
			}
			// Update in DB is made given the primary key
			proj.ID = p.ID
			proj.VCSServers = p.VCSServers
			if proj.Icon == "" {
				p.Icon = proj.Icon
			}
			if err := project.Update(api.mustDB(), proj); err != nil {
				return sdk.WrapError(err, "cannot update project %s", key)
			}
			event_v2.PublishProjectEvent(ctx, api.Cache, sdk.EventProjectUpdated, *proj, *u.AuthConsumerUser.AuthentifiedUser)

			proj.Permissions.Writable = true

			return service.WriteJSON(w, proj, http.StatusOK)
		}
}

func (api *API) getProjectV2Handler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			// Get project name in URL
			vars := mux.Vars(r)
			key := vars["projectKey"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			p, errProj := project.Load(ctx, api.mustDB(), key)
			if errProj != nil {
				return sdk.WrapError(errProj, "getProjectHandler (%s)", key)
			}

			if isAdmin(ctx) {
				p.Permissions.Writable = true
			} else {
				var err error
				p.Permissions.Writable, err = rbac.HasRoleOnProjectAndUserID(ctx, api.mustDB(), sdk.ProjectRoleManage, u.AuthConsumerUser.AuthentifiedUser.ID, key)
				if err != nil {
					return err
				}
			}

			return service.WriteJSON(w, p, http.StatusOK)
		}
}

func (api *API) getProjectV2AccessHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.isCDNService),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)

			projectKey := vars["projectKey"]
			itemType := vars["type"]

			if sdk.CDNItemType(itemType) == sdk.CDNTypeItemWorkerCache {
				return sdk.WrapError(sdk.ErrForbidden, "cdn is not enabled for this type %s", itemType)
			}

			sessionID := req.Header.Get(sdk.CDSSessionID)
			if sessionID == "" {
				return sdk.WrapError(sdk.ErrForbidden, "missing session id header")
			}

			session, err := authentication.LoadSessionByID(ctx, api.mustDBWithCtx(ctx), sessionID)
			if err != nil {
				return err
			}
			consumer, err := authentication.LoadUserConsumerByID(ctx, api.mustDB(), session.ConsumerID,
				authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
			if err != nil {
				return sdk.NewErrorWithStack(err, sdk.ErrUnauthorized)
			}

			if consumer.Disabled {
				return sdk.WrapError(sdk.ErrUnauthorized, "consumer (%s) is disabled", consumer.ID)
			}

			maintainerOrAdmin := consumer.Maintainer() || consumer.Admin()
			canRead, err := rbac.HasRoleOnProjectAndUserID(ctx, api.mustDB(), sdk.ProjectRoleRead, consumer.AuthConsumerUser.AuthentifiedUserID, projectKey)
			if err != nil {
				return err
			}

			if maintainerOrAdmin || canRead {
				return service.WriteJSON(w, nil, http.StatusOK)
			}
			return service.WriteJSON(w, nil, http.StatusForbidden)
		}
}
