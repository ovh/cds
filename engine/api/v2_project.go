package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/service"
)

// getAllRepositoriesHandler Get all repositories
func (api *API) getAllRepositoriesHandler() ([]service.RbacChecker, service.Handler) {
	return nil,
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			repos, err := repository.LoadAllRepositories(ctx, api.mustDB())
			if err != nil {
				return err
			}
			return service.WriteJSON(w, repos, http.StatusOK)
		}
}
