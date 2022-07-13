package api

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getRepositoryByIdentifier(ctx context.Context, vcsID string, repositoryIdentifier string) (*sdk.ProjectRepository, error) {
	var repo *sdk.ProjectRepository
	var err error
	if sdk.IsValidUUID(repositoryIdentifier) {
		repo, err = repository.LoadRepositoryByVCSAndID(ctx, api.mustDB(), vcsID, repositoryIdentifier)
	} else {
		repo, err = repository.LoadRepositoryByName(ctx, api.mustDB(), vcsID, repositoryIdentifier)
	}
	if err != nil {
		return nil, err
	}
	return repo, nil
}

// deleteProjectRepositoryHandler Delete a repository from a project
func (api *API) deleteProjectRepositoryHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(rbac.ProjectManage),
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

			vcsProject, err := api.getVCSByIdentifier(ctx, pKey, vcsIdentifier)
			if err != nil {
				return err
			}

			repo, err := api.getRepositoryByIdentifier(ctx, vcsProject.ID, repositoryIdentifier)
			if err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			// Remove hooks
			srvs, err := services.LoadAllByType(ctx, tx, sdk.TypeHooks)
			if err != nil {
				return err
			}
			if len(srvs) < 1 {
				return sdk.NewErrorFrom(sdk.ErrNotFound, "unable to find hook uservice")
			}
			_, code, errHooks := services.NewClient(tx, srvs).DoJSONRequest(ctx, http.MethodDelete, "/task/"+repo.ID, nil, nil)
			if (errHooks != nil || code >= 400) && code != 404 {
				return sdk.WrapError(errHooks, "unable to delete hook [HTTP: %d]", code)
			}

			if err := repository.Delete(tx, repo.VCSProjectID, repo.Name); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}
			return service.WriteMarshal(w, req, vcsProject, http.StatusOK)
		}
}

// postProjectRepositoryHandler Attach a new repository to the given project
func (api *API) postProjectRepositoryHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(rbac.ProjectManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]

			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			vcsProject, err := api.getVCSByIdentifier(ctx, pKey, vcsIdentifier)
			if err != nil {
				return err
			}

			var repo sdk.ProjectRepository
			if err := service.UnmarshalRequest(ctx, req, &repo); err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			repo.VCSProjectID = vcsProject.ID
			repo.CreatedBy = getAPIConsumer(ctx).GetUsername()

			// Insert Repository
			if err := repository.Insert(ctx, tx, &repo); err != nil {
				return err
			}

			// Check if repo exist
			vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, pKey, vcsProject.Name)
			if err != nil {
				return err
			}
			vcsRepo, err := vcsClient.RepoByFullname(ctx, repo.Name)
			if err != nil {
				return err
			}

			// Create hook
			srvs, err := services.LoadAllByType(ctx, tx, sdk.TypeHooks)
			if err != nil {
				return err
			}
			if len(srvs) < 1 {
				return sdk.NewErrorFrom(sdk.ErrNotFound, "unable to find hook uservice")
			}
			repositoryHookRegister := sdk.NewEntitiesHook(repo.ID, pKey, vcsProject.Type, vcsProject.Name, repo.Name)
			_, code, errHooks := services.NewClient(tx, srvs).DoJSONRequest(ctx, http.MethodPost, "/v2/task", repositoryHookRegister, nil)
			if errHooks != nil || code >= 400 {
				return sdk.WrapError(errHooks, "unable to create hooks [HTTP: %d]", code)
			}

			if repo.Auth.SSHKeyName != "" {
				repo.CloneURL = vcsRepo.SSHCloneURL
			} else {
				repo.CloneURL = vcsRepo.HTTPCloneURL
			}

			// Update repository with Hook configuration
			repo.HookConfiguration = repositoryHookRegister.Configuration
			if err := repository.Update(ctx, tx, &repo); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}
			return service.WriteMarshal(w, req, vcsProject, http.StatusCreated)
		}
}

// getVCSProjectRepositoryAllHandler returns the list of repositories linked to the given vcs/project
func (api *API) getVCSProjectRepositoryAllHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(rbac.ProjectRead),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			vars := mux.Vars(r)
			pKey := vars["projectKey"]

			vcsIdentifier, err := url.PathUnescape(vars["vcsIdentifier"])
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			vcsProject, err := api.getVCSByIdentifier(ctx, pKey, vcsIdentifier)
			if err != nil {
				return err
			}

			repositories, err := repository.LoadAllRepositoriesByVCSProjectID(ctx, api.mustDB(), vcsProject.ID)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, repositories, http.StatusOK)
		}
}
