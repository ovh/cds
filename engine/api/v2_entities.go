package api

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getEntitiesHandler() ([]service.RbacChecker, service.Handler) {
	return nil,
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			entityType := vars["entityType"]

			u := getUserConsumer(ctx)
			if u == nil {
				return sdk.ErrUnauthorized
			}

			var entities []sdk.EntityFullName
			if isAdmin(ctx) {
				var err error
				entities, err = entity.UnsafeLoadAllByType(ctx, api.mustDB(), entityType)
				if err != nil {
					return err
				}
			} else {
				projectKeys, err := rbac.LoadAllProjectKeysAllowed(ctx, api.mustDB(), sdk.ProjectRoleRead, u.AuthConsumerUser.AuthentifiedUserID)
				if err != nil {
					return err
				}
				entities, err = entity.UnsafeLoadAllByTypeAndProjectKeys(ctx, api.mustDB(), entityType, projectKeys)
				if err != nil {
					return err
				}
			}

			return service.WriteJSON(w, entities, http.StatusOK)
		}
}

// getProjectEntitiesHandler
func (api *API) getProjectEntitiesHandler() ([]service.RbacChecker, service.Handler) {
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

			if branch == "" {
				tx, err := api.mustDB().Begin()
				if err != nil {
					return err
				}
				vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, pKey, vcsProject.Name)
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

			entities, err := entity.LoadByRepositoryAndBranch(ctx, api.mustDB(), repo.ID, branch)
			if err != nil {
				return err
			}
			result := make([]sdk.ShortEntity, 0, len(entities))
			for _, e := range entities {
				result = append(result, sdk.ShortEntity{
					ID:     e.ID,
					Name:   e.Name,
					Type:   e.Type,
					Branch: e.Branch,
				})
			}
			return service.WriteJSON(w, result, http.StatusOK)
		}
}

func (api *API) getProjectEntityHandler() ([]service.RbacChecker, service.Handler) {
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
			entityType := vars["entityType"]
			entityName := vars["entityName"]

			branch := QueryString(req, "branch")

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
				vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, pKey, vcsProject.Name)
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

			entity, err := entity.LoadByBranchTypeName(ctx, api.mustDB(), repo.ID, branch, entityType, entityName)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, entity, http.StatusOK)
		}
}
