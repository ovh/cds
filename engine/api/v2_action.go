package api

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getActionV2Handler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.workerModelRead),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			repositoryIdentifier, err := url.PathUnescape(vars["repositoryIdentifier"])
			if err != nil {
				return sdk.WithStack(err)
			}
			actionName := vars["actionName"]

			proj, err := project.Load(ctx, api.mustDB(), pKey)
			if err != nil {
				return err
			}

			vcsProject, err := api.getVCSByIdentifier(ctx, pKey, vcsIdentifier)
			if err != nil {
				return err
			}

			repo, err := api.getRepositoryByIdentifier(ctx, vcsProject.ID, repositoryIdentifier)
			if err != nil {
				return err
			}

			ref, commit, err := api.getEntityRefFromQueryParams(ctx, req, proj.Key, vcsProject.Name, repo.Name)
			if err != nil {
				return err
			}

			var act sdk.V2Action
			if err := entity.LoadAndUnmarshalByRefTypeName(ctx, api.mustDB(), repo.ID, ref, commit, sdk.EntityTypeAction, actionName, &act); err != nil {
				return err
			}
			return service.WriteJSON(w, act, http.StatusOK)
		}
}
