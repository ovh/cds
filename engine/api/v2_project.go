package api

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// getAllRepositoriesHandler Get all repositories
func (api *API) getAllRepositoriesHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.isHookService),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			repos, err := repository.LoadAllRepositories(ctx, api.mustDB())
			if err != nil {
				return err
			}
			return service.WriteJSON(w, repos, http.StatusOK)
		}
}

func (api *API) getRepositoryHookHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.isHookService),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			repoIdentifier, err := url.PathUnescape(vars["repositoryIdentifier"])
			if err != nil {
				return sdk.NewErrorWithStack(
					err,
					sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given repository identifier"),
				)
			}
			if !sdk.IsValidUUID(repoIdentifier) {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "this handler needs the repository uuid")
			}
			repo, err := repository.LoadRepositoryByID(ctx, api.mustDB(), repoIdentifier, gorpmapping.GetOptions.WithDecryption)
			if err != nil {
				return err
			}
			h := sdk.Hook{
				HookSignKey:   repo.HookSignKey,
				UUID:          repo.ID,
				HookType:      sdk.RepositoryEntitiesHook,
				Configuration: repo.HookConfiguration,
			}
			return service.WriteJSON(w, h, http.StatusOK)
		}
}
