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

func (api *API) getWorkerModelTemplatesHandler() ([]service.RbacChecker, service.Handler) {
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
				entities, err = entity.LoadByType(ctx, api.mustDB(), repo.ID, sdk.EntityTypeWorkerModelTemplate)
			} else {
				entities, err = entity.LoadByTypeAndBranch(ctx, api.mustDB(), repo.ID, sdk.EntityTypeWorkerModelTemplate, branch)
			}
			if err != nil {
				return err
			}
			modelTemplates := make([]sdk.WorkerModelTemplate, 0, len(entities))
			for _, e := range entities {
				var mt sdk.WorkerModelTemplate
				if err := yaml.Unmarshal([]byte(e.Data), &mt); err != nil {
					return sdk.WithStack(err)
				}
				modelTemplates = append(modelTemplates, mt)
			}
			return service.WriteJSON(w, modelTemplates, http.StatusOK)
		}
}
