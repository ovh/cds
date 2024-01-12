package api

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/rockbears/yaml"

	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getWorkerModelV2Handler() ([]service.RbacChecker, service.Handler) {
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
			workerModelName := vars["workerModelName"]

			// Secret only available for the hatchery
			withCreds := QueryBool(req, "withSecrets")
			if withCreds && getHatcheryConsumer(ctx) == nil {
				return sdk.NewErrorFrom(sdk.ErrForbidden, "You cannot ask for secrets")
			}

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

			ref, err := api.getEntityRefFromQueryParams(ctx, req, pKey, vcsProject.Name, repo.Name)
			if err != nil {
				return err
			}

			var workerModel sdk.V2WorkerModel
			ent, err := entity.LoadByRefTypeName(ctx, api.mustDB(), repo.ID, ref, sdk.EntityTypeWorkerModel, workerModelName)
			if err != nil {
				return err
			}
			if err := yaml.Unmarshal([]byte(ent.Data), &workerModel); err != nil {
				return sdk.WrapError(err, "unable to read %s / %s @ %s", repo.ID, workerModelName, ref)
			}
			// Only hatchery can ask for cred
			if withCreds && getHatcheryConsumer(ctx) != nil {
				if err := entity.WorkerModelDecryptSecrets(ctx, api.mustDB(), proj.ID, &workerModel, project.DecryptWithBuiltinKey); err != nil {
					return err
				}
			}
			return service.WriteJSON(w, workerModel, http.StatusOK)
		}
}

func (api *API) getWorkerModelsV2Handler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectRead),
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
			tag := QueryString(req, "tag")

			if tag != "" && branch != "" {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "Query param tag and branch cannot be used together")
			}

			vcsProject, err := api.getVCSByIdentifier(ctx, pKey, vcsIdentifier)
			if err != nil {
				return err
			}

			repo, err := api.getRepositoryByIdentifier(ctx, vcsProject.ID, repositoryIdentifier)
			if err != nil {
				return err
			}

			var entities []sdk.Entity
			if branch == "" && tag == "" {
				entities, err = entity.LoadByRepositoryAndType(ctx, api.mustDB(), repo.ID, sdk.EntityTypeWorkerModel)
			} else {
				var ref string
				if tag != "" {
					ref = sdk.GitRefTagPrefix + tag
				} else {
					ref = sdk.GitRefBranchPrefix + branch
				}
				entities, err = entity.LoadByTypeAndRef(ctx, api.mustDB(), repo.ID, sdk.EntityTypeWorkerModel, ref)
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
