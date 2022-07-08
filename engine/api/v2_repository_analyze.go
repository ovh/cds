package api

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/service"
)

// postRepositoryAnalyzeHandler Trigger repository analysys
func (api *API) postRepositoryAnalyzeHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(rbac.IsHookService),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			repos, err := repository.LoadAllRepositories(ctx, api.mustDB())
			if err != nil {
				return err
			}
			return service.WriteJSON(w, repos, http.StatusOK)
		}
}
