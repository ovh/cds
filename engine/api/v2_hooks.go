package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getHooksRepositoriesHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.isHookService),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			vcsName := vars["vcsServer"]

			repoName, err := url.PathUnescape(vars["repositoryName"])
			if err != nil {
				return sdk.WithStack(err)
			}

			repos, err := repository.LoadByNameWithoutVCSServer(ctx, api.mustDB(), repoName)
			if err != nil {
				return err
			}

			repositories := make([]sdk.ProjectRepository, 0)
			for _, r := range repos {
				vcsserver, err := vcs.LoadVCSByIDAndProjectKey(ctx, api.mustDB(), r.ProjectKey, r.VCSProjectID)
				if err != nil {
					return err
				}
				if vcsserver.Name == vcsName {
					repositories = append(repositories, r)
				}
			}
			return service.WriteJSON(w, repositories, http.StatusOK)
		}
}

func (api *API) getRepositoryWebHookSecretHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.projectManage),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			pKey := vars["projectKey"]
			vcsType := vars["vcsType"]
			vcsName := vars["vcsServer"]
			repositoryName, err := url.PathUnescape(vars["repositoryName"])
			if err != nil {
				return err
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback()

			// Check if project has read access
			vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, pKey, vcsName)
			if err != nil {
				return err
			}
			if _, err := vcsClient.RepoByFullname(ctx, repositoryName); err != nil {
				return err
			}

			srvs, err := services.LoadAllByType(ctx, tx, sdk.TypeHooks)
			if err != nil {
				return err
			}
			if len(srvs) < 1 {
				return sdk.NewErrorFrom(sdk.ErrNotFound, "unable to find hook uservice")
			}
			path := fmt.Sprintf("/v2/repository/key/%s/%s", vcsName, url.PathEscape(repositoryName))

			var keyResp sdk.GenerateRepositoryWebhook
			_, code, err := services.NewClient(tx, srvs).DoJSONRequest(ctx, http.MethodGet, path, nil, &keyResp)
			if err != nil {
				return sdk.WrapError(err, "unable to delete hook [HTTP: %d]", code)
			}

			hookData := sdk.HookAccessData{
				HookSignKey: keyResp.Key,
				URL:         fmt.Sprintf("%s/v2/webhook/repository/%s/%s", srvs[0].HTTPURL, vcsType, vcsName),
			}
			return service.WriteJSON(w, hookData, http.StatusOK)
		}
}
