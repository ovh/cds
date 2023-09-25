package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
)

func (api *API) postRetrieveWorkflowToTriggerHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.isHookService),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {

			var hookRequest sdk.HookListWorkflowRequest
			if err := service.UnmarshalRequest(ctx, req, &hookRequest); err != nil {
				return err
			}

			uniqueWorkflowMap := make(map[string]struct{})
			filteredWorkflowHooks := make([]sdk.V2WorkflowHook, 0)

			// Get repository web hooks
			workflowHooks, err := LoadWorkflowHooksWithRepositoryWebHooks(ctx, api.mustDB(), hookRequest)
			if err != nil {
				return err
			}
			log.Info(ctx, "found %d repository webhooks for event %+v", len(workflowHooks), hookRequest)
			for _, wk := range workflowHooks {
				if _, has := uniqueWorkflowMap[wk.EntityID]; !has {
					filteredWorkflowHooks = append(filteredWorkflowHooks, wk)
					uniqueWorkflowMap[wk.EntityID] = struct{}{}
				}
			}

			// Get workflow_update hooks
			workflowUpdateHooks, err := LoadWorkflowHooksWithWorkflowUpdate(ctx, api.mustDB(), hookRequest)
			if err != nil {
				return err
			}
			log.Info(ctx, "found %d workflow_update hook for event %+v", len(workflowUpdateHooks), hookRequest)
			for _, wk := range workflowUpdateHooks {
				if _, has := uniqueWorkflowMap[wk.EntityID]; !has {
					filteredWorkflowHooks = append(filteredWorkflowHooks, wk)
					uniqueWorkflowMap[wk.EntityID] = struct{}{}
				}
			}

			// Get model_update hooks
			modelUpdateHooks, err := LoadWorkflowHooksWithModelUpdate(ctx, api.mustDB(), hookRequest)
			if err != nil {
				return err
			}
			log.Info(ctx, "found %d workermodel_update hook for event %+v", len(modelUpdateHooks), hookRequest)
			for _, wk := range modelUpdateHooks {
				if _, has := uniqueWorkflowMap[wk.EntityID]; !has {
					filteredWorkflowHooks = append(filteredWorkflowHooks, wk)
					uniqueWorkflowMap[wk.EntityID] = struct{}{}
				}
			}

			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WithStack(err)
			}
			defer tx.Rollback()

			hooksWithReadRight := make([]sdk.V2WorkflowHook, 0)
			for _, h := range filteredWorkflowHooks {
				if !hookRequest.AnayzedProjectKeys.Contains(h.ProjectKey) {
					// Check project right
					vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, tx, api.Cache, h.ProjectKey, h.VCSName)
					if err != nil {
						return err
					}
					if _, err := vcsClient.RepoByFullname(ctx, hookRequest.RepositoryName); err != nil {
						log.Info(ctx, "hook %s/s on  %s/%s has no right on repository %s/%s", h.ID, h.Type, h.ProjectKey, h.WorkflowName, hookRequest.VCSName, hookRequest.RepositoryName)
						continue
					}
				}
				hooksWithReadRight = append(hooksWithReadRight, h)
			}

			if err := tx.Commit(); err != nil {
				return sdk.WithStack(err)
			}

			return service.WriteJSON(w, hooksWithReadRight, http.StatusOK)
		}
}

// LoadWorkflowHooksWithModelUpdate
// hookRequest contains all updated model from analysis
func LoadWorkflowHooksWithModelUpdate(ctx context.Context, db gorp.SqlExecutor, hookRequest sdk.HookListWorkflowRequest) ([]sdk.V2WorkflowHook, error) {
	filteredWorkflowHooks := make([]sdk.V2WorkflowHook, 0)

	models := make([]string, 0, len(hookRequest.Models))
	for _, m := range hookRequest.Models {
		models = append(models, fmt.Sprintf("%s/%s/%s/%s", m.ProjectKey, m.VCSName, m.RepoName, m.Name))
	}
	entitiesHooks, err := workflow_v2.LoadHooksByModelUpdated(ctx, db, models)
	if err != nil {
		return nil, err
	}
	for _, h := range entitiesHooks {
		if h.Branch == hookRequest.Branch {
			filteredWorkflowHooks = append(filteredWorkflowHooks, h)
		}
	}
	return filteredWorkflowHooks, nil
}

// LoadWorkflowHooksWithWorkflowUpdate
// hookRequest contains all updated workflow from analysis
func LoadWorkflowHooksWithWorkflowUpdate(ctx context.Context, db gorp.SqlExecutor, hookRequest sdk.HookListWorkflowRequest) ([]sdk.V2WorkflowHook, error) {
	filteredWorkflowHooks := make([]sdk.V2WorkflowHook, 0)

	for _, w := range hookRequest.Workflows {
		h, err := workflow_v2.LoadHooksByWorkflowUpdated(ctx, db, w.ProjectKey, w.VCSName, w.RepoName, w.Name)
		if err != nil {
			if sdk.ErrorIs(err, sdk.ErrNotFound) {
				continue
			}
			return nil, err
		}
		// check of event come from the right branch
		if hookRequest.Branch == h.Branch {
			filteredWorkflowHooks = append(filteredWorkflowHooks, *h)
		}
	}
	return filteredWorkflowHooks, nil
}

// LoadWorkflowHooksWithRepositoryWebHooks
// If event && workflow declaration are on the same repo : get only the hook defined on the current branch
// Else get all ( analyse process insert only 1 hook for the default branch
func LoadWorkflowHooksWithRepositoryWebHooks(ctx context.Context, db gorp.SqlExecutor, hookRequest sdk.HookListWorkflowRequest) ([]sdk.V2WorkflowHook, error) {
	// Repositories hooks
	workflowHooks, err := workflow_v2.LoadHooksByRepositoryEvent(ctx, db, hookRequest.VCSName, hookRequest.RepositoryName, hookRequest.RepositoryEventName)
	if err != nil {
		return nil, err
	}

	filteredWorkflowHooks := make([]sdk.V2WorkflowHook, 0)

	for _, w := range workflowHooks {
		// If event && workflow declaration are on the same repo
		if w.VCSName == hookRequest.VCSName && w.RepositoryName == hookRequest.RepositoryName {
			// Only get workflow configuration from current branch
			if w.Branch != hookRequest.Branch {
				continue
			}
		}

		// Check configuration : branch filter + path filter
		switch hookRequest.RepositoryEventName {
		case sdk.WorkflowHookEventPush:
			validBranch := sdk.IsValidHookBranch(ctx, w.Data.BranchFilter, hookRequest.Branch)
			validPath := sdk.IsValidHookPath(ctx, w.Data.PathFilter, hookRequest.Paths)
			if validBranch && validPath {
				filteredWorkflowHooks = append(filteredWorkflowHooks, w)
			}
			continue
		}
	}
	return filteredWorkflowHooks, nil
}

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
