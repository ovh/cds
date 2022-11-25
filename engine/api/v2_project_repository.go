package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getRepositoryByIdentifier(ctx context.Context, vcsID string, repositoryIdentifier string, opts ...gorpmapper.GetOptionFunc) (*sdk.ProjectRepository, error) {
	var repo *sdk.ProjectRepository
	var err error
	if sdk.IsValidUUID(repositoryIdentifier) {
		repo, err = repository.LoadRepositoryByVCSAndID(ctx, api.mustDB(), vcsID, repositoryIdentifier, opts...)
	} else {
		repo, err = repository.LoadRepositoryByName(ctx, api.mustDB(), vcsID, repositoryIdentifier, opts...)
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
			vcsProjectWithSecret, err := api.getVCSByIdentifier(ctx, pKey, vcsIdentifier, gorpmapper.GetOptions.WithDecryption)
			if err != nil {
				return err
			}

			var repoBody sdk.ProjectRepository
			if err := service.UnmarshalRequest(ctx, req, &repoBody); err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

			repoDB := repoBody
			repoDB.VCSProjectID = vcsProjectWithSecret.ID
			repoDB.CreatedBy = getUserConsumer(ctx).GetUsername()

			// Insert Repository
			if err := repository.Insert(ctx, tx, &repoDB); err != nil {
				return err
			}

			// Check if repo exist
			vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, pKey, vcsProjectWithSecret.Name)
			if err != nil {
				return err
			}
			vcsRepo, err := vcsClient.RepoByFullname(ctx, repoDB.Name)
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
			repositoryHookRegister, err := sdk.NewEntitiesHook(repoDB.ID, pKey, vcsProjectWithSecret.Type, vcsProjectWithSecret.Name, repoDB.Name)
			if err != nil {
				return err
			}

			_, code, errHooks := services.NewClient(tx, srvs).DoJSONRequest(ctx, http.MethodPost, "/v2/task", repositoryHookRegister, nil)
			if errHooks != nil || code >= 400 {
				return sdk.WrapError(errHooks, "unable to create hooks [HTTP: %d]", code)
			}

			if vcsProjectWithSecret.Auth.SSHKeyName != "" {
				if vcsRepo.SSHCloneURL == "" {
					return sdk.NewErrorFrom(sdk.ErrInvalidData, "this repo cannot be cloned using ssh.")
				}
				repoDB.CloneURL = vcsRepo.SSHCloneURL
			} else {
				if vcsRepo.HTTPCloneURL == "" {
					return sdk.NewErrorFrom(sdk.ErrInvalidData, "this repo cannot be cloned using https. Please provide a sshkey.")
				}
				repoDB.CloneURL = vcsRepo.HTTPCloneURL
			}
			repoDB.HookSignKey = repositoryHookRegister.HookSignKey

			// Update repository with Hook configuration
			repoDB.HookConfiguration = repositoryHookRegister.Configuration
			if err := repository.Update(ctx, tx, &repoDB); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}
			return service.WriteMarshal(w, req, repoDB, http.StatusCreated)
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

func (api *API) postRepositoryHookRegenKeyHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(rbac.ProjectManage),
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			vars := mux.Vars(r)
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

			repo, err := api.getRepositoryByIdentifier(ctx, vcsProject.ID, repositoryIdentifier, gorpmapper.GetOptions.WithDecryption)
			if err != nil {
				return err
			}
			newSecret, err := sdk.GenerateHookSecret()
			if err != nil {
				return err
			}
			repo.HookSignKey = newSecret

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			if err := repository.Update(ctx, tx, repo); err != nil {
				return err
			}
			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			srvs, err := services.LoadAllByType(ctx, api.mustDB(), sdk.TypeHooks)
			if err != nil {
				return err
			}
			if len(srvs) == 0 {
				return sdk.NewErrorFrom(sdk.ErrNotFound, "no hook service found")
			}
			hook := sdk.HookAccessData{
				URL:         fmt.Sprintf("%s/v2/webhook/repository/%s/%s", srvs[0].HTTPURL, vcsProject.Type, repo.ID),
				HookSignKey: newSecret,
			}
			return service.WriteJSON(w, hook, http.StatusOK)
		}
}
