package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getRepositoryHooksHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]

			hooks, err := project.LoadAllWebHooks(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, hooks, http.StatusOK)
		}
}

func (api *API) deleteRepositoryHookHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			uuid := vars["uuid"]

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback()

			if err := project.DeleteWebHook(tx, pKey, uuid); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			return nil
		}
}

func (api *API) getRepositoryHookHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			uuid := vars["uuid"]

			hooks, err := project.LoadWebHookByID(ctx, api.mustDB(), pKey, uuid)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, hooks, http.StatusOK)

		}
}

func (api *API) postRepositoryHookHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.WithStack(sdk.ErrForbidden)
			}

			var r sdk.PostProjectWebHook
			if err := service.UnmarshalBody(req, &r); err != nil {
				return err
			}

			if err := r.Valid(); err != nil {
				return err
			}

			vcs, err := vcs.LoadVCSByProject(ctx, api.mustDB(), pKey, r.VCSServer)
			if err != nil {
				return err
			}

			// Check if project has read access
			vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, pKey, vcs.Name)
			if err != nil {
				return err
			}
			if _, err := vcsClient.RepoByFullname(ctx, r.Repository); err != nil {
				return err
			}

			srvs, err := services.LoadAllByType(ctx, api.mustDB(), sdk.TypeHooks)
			if err != nil {
				return err
			}
			if len(srvs) < 1 {
				return sdk.NewErrorFrom(sdk.ErrNotFound, "unable to find hook uservice")
			}
			path := fmt.Sprintf("/v2/repository/key/%s/%s/%s", pKey, url.PathEscape(r.VCSServer), url.PathEscape(r.Repository))

			var keyResp sdk.GenerateRepositoryWebhook
			_, code, err := services.NewClient(srvs).DoJSONRequest(ctx, http.MethodPost, path, nil, &keyResp)
			if err != nil {
				return sdk.WrapError(err, "unable to delete hook [HTTP: %d]", code)
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback()

			h := sdk.ProjectWebHook{
				ID:         keyResp.UUID,
				ProjectKey: pKey,
				VCSServer:  r.VCSServer,
				Repository: r.Repository,
				Workflow:   r.Workflow,
				Username:   u.GetUsername(),
				Type:       r.Type,
			}
			if err := project.InsertWebHook(ctx, tx, &h); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			hookData := sdk.HookAccessData{
				HookSignKey: keyResp.Key,
				URL:         fmt.Sprintf("%s/v2/webhook/repository/%s/%s/%s/%s", keyResp.HookPublicURL, pKey, vcs.Type, vcs.Name, keyResp.UUID),
			}
			return service.WriteJSON(w, hookData, http.StatusOK)
		}
}
