package api

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/rockbears/yaml"

	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getWorkerModelsV2Handler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(rbac.ProjectRead),
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

			branch := QueryString(req, "branch")

			vcsProject, err := api.getVCSByIdentifier(ctx, pKey, vcsIdentifier)
			if err != nil {
				return err
			}

			repo, err := api.getRepositoryByIdentifier(ctx, vcsProject.ID, repositoryIdentifier)
			if err != nil {
				return err
			}

			var entities []sdk.Entity
			if branch == "" {
				entities, err = entity.LoadByType(ctx, api.mustDB(), repo.ID, sdk.EntityTypeWorkerModel)
			} else {
				entities, err = entity.LoadByTypeAndBranch(ctx, api.mustDB(), repo.ID, sdk.EntityTypeWorkerModel, branch)
			}
			if err != nil {
				return err
			}
			modelTemplates := make([]sdk.V2WorkerModel, 0, len(entities))
			for _, e := range entities {
				var mt sdk.V2WorkerModel
				if err := yaml.Unmarshal([]byte(e.Data), &mt); err != nil {
					return sdk.WithStack(err)
				}
				modelTemplates = append(modelTemplates, mt)
			}
			return service.WriteJSON(w, modelTemplates, http.StatusOK)
		}
}
