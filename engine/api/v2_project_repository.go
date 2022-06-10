package api

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// deleteProjectRepositoryHandler Delete a repository from a project
func (api *API) deleteProjectRepositoryHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(rbac.ProjectManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			vcsName := vars["vcsProjectName"]
			repositoryName, err := url.PathUnescape(vars["repositoryName"])
			if err != nil {
				return sdk.WithStack(err)
			}

			vcsProject, err := vcs.LoadVCSByProject(ctx, api.mustDB(), pKey, vcsName)
			if err != nil {
				return err
			}

			repo, err := repository.LoadRepositoryByName(ctx, api.mustDB(), vcsProject.ID, repositoryName)
			if err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback() // nolint

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
			vcsName := vars["vcsProjectName"]

			vcsProject, err := vcs.LoadVCSByProject(ctx, api.mustDB(), pKey, vcsName)
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
			if err := repository.Insert(ctx, tx, &repo); err != nil {
				return err
			}

			// Check if repo exist
			vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, pKey, vcsName)
			if err != nil {
				return err
			}
			if _, err := vcsClient.RepoByFullname(ctx, repo.Name); err != nil {
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
			vcsName := vars["vcsProjectName"]

			vcsProject, err := vcs.LoadVCSByProject(ctx, api.mustDB(), pKey, vcsName)
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
