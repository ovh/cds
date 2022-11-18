package api

import (
	"context"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getEntitiesHandler() ([]service.RbacChecker, service.Handler) {
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
