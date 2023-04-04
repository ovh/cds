package api

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/rockbears/yaml"

	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
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
			branch := QueryString(req, "branch")

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

			if branch == "" {
				tx, err := api.mustDB().Begin()
				if err != nil {
					return err
				}
				vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, proj.Key, vcsProject.Name)
				if err != nil {
					_ = tx.Rollback()
					return err
				}
				defaultBranch, err := vcsClient.Branch(ctx, repo.Name, sdk.VCSBranchFilters{Default: true})
				if err != nil {
					_ = tx.Rollback()
					return err
				}
				if err := tx.Commit(); err != nil {
					_ = tx.Rollback()
					return err
				}
				branch = defaultBranch.DisplayID
			}

			var workerModel sdk.V2WorkerModel
			if err := entity.LoadAndUnmarshalByBranchTypeName(ctx, api.mustDB(), repo.ID, branch, sdk.EntityTypeWorkerModel, workerModelName, &workerModel); err != nil {
				return err
			}
			if withCreds {
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
				entities, err = entity.LoadByRepositoryAndType(ctx, api.mustDB(), repo.ID, sdk.EntityTypeWorkerModel)
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
