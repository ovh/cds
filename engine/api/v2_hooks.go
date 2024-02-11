package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/operation"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/repository"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/vcs"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/rockbears/log"
)

func (api *API) postRetrieveEventUserHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.isHookService),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			var r sdk.HookRetrieveUserRequest
			if err := service.UnmarshalBody(req, &r); err != nil {
				return err
			}
			ctx = context.WithValue(ctx, cdslog.HookEventID, r.HookEventUUID)

			vcsProjectWithSecret, err := vcs.LoadVCSByProject(ctx, api.mustDB(), r.ProjectKey, r.VCSServerName, gorpmapping.GetOptions.WithDecryption)
			if err != nil {
				return err
			}

			resp := sdk.HookRetrieveUserResponse{}
			u, _, _, err := findCommitter(ctx, api.Cache, api.mustDB(), r.Commit, r.SignKey, r.ProjectKey, *vcsProjectWithSecret, r.RepositoryName, api.Config.VCS.GPGKeys)
			if err != nil {
				return err
			}
			if u != nil {
				resp.UserID = u.ID
				resp.Username = u.Username
			}
			return service.WriteJSON(w, resp, http.StatusOK)
		}
}

func (api *API) getRetrieveSignKeyOperationHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.isHookService),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			vars := mux.Vars(req)
			uuid := vars["uuid"]

			ope, err := operation.GetRepositoryOperation(ctx, api.mustDB(), uuid)
			if err != nil {
				return err
			}
			return service.WriteJSON(w, ope, http.StatusOK)
		}
}

func (api *API) postHookEventRetrieveSignKeyHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.isHookService),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
			var hookRetrieveSignKey sdk.HookRetrieveSignKeyRequest
			if err := service.UnmarshalRequest(ctx, req, &hookRetrieveSignKey); err != nil {
				return err
			}

			ctx = context.WithValue(ctx, cdslog.HookEventID, hookRetrieveSignKey.HookEventUUID)
			ctx = context.WithValue(ctx, cdslog.Project, hookRetrieveSignKey.ProjectKey)

			proj, err := project.Load(ctx, api.mustDB(), hookRetrieveSignKey.ProjectKey, project.LoadOptions.WithClearKeys)
			if err != nil {
				return err
			}

			vcsProjectWithSecret, err := vcs.LoadVCSByProject(ctx, api.mustDB(), hookRetrieveSignKey.ProjectKey, hookRetrieveSignKey.VCSServerName, gorpmapping.GetOptions.WithDecryption)
			if err != nil {
				return err
			}

			vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, api.mustDB(), api.Cache, hookRetrieveSignKey.ProjectKey, hookRetrieveSignKey.VCSServerName)
			if err != nil {
				return err
			}
			repo, err := vcsClient.RepoByFullname(ctx, hookRetrieveSignKey.RepositoryName)
			if err != nil {
				log.Info(ctx, "unable to get repository %s/%s for project %s", hookRetrieveSignKey.VCSServerName, hookRetrieveSignKey.RepositoryName, hookRetrieveSignKey.ProjectKey)
				return err
			}

			cloneURL := repo.SSHCloneURL
			if vcsProjectWithSecret.Auth.SSHKeyName == "" {
				cloneURL = repo.HTTPCloneURL
			}
			ope, err := operation.CheckoutAndAnalyzeOperation(ctx, api.mustDB(), *proj, *vcsProjectWithSecret, repo.Fullname, cloneURL, hookRetrieveSignKey.Commit, hookRetrieveSignKey.Ref)
			if err != nil {
				return err
			}

			api.GoRoutines.Exec(context.Background(), "operation-polling-"+ope.UUID, func(ctx context.Context) {
				ope, err := operation.Poll(ctx, api.mustDB(), ope.UUID)
				if err != nil {
					log.ErrorWithStackTrace(ctx, err)
					ope.Status = sdk.OperationStatusError
					ope.Error = &sdk.OperationError{Message: fmt.Sprintf("%v", err)}
				}

				// Send result to hooks
				srvs, err := services.LoadAllByType(ctx, api.mustDB(), sdk.TypeHooks)
				if err != nil {
					log.ErrorWithStackTrace(ctx, err)
					return
				}
				if len(srvs) < 1 {
					log.ErrorWithStackTrace(ctx, sdk.NewErrorFrom(sdk.ErrNotFound, "unable to find hook uservice"))
					return
				}
				callback := sdk.HookEventCallback{
					VCSServerType:      hookRetrieveSignKey.VCSServerType,
					VCSServerName:      hookRetrieveSignKey.VCSServerName,
					RepositoryName:     hookRetrieveSignKey.RepositoryName,
					HookEventUUID:      hookRetrieveSignKey.HookEventUUID,
					SigningKeyCallback: &sdk.HookSigninKeyCallback{},
				}
				if ope.Status == sdk.OperationStatusDone {
					callback.SigningKeyCallback.SemverCurrent = ope.Setup.Checkout.Result.Semver.Current
					callback.SigningKeyCallback.SemverNext = ope.Setup.Checkout.Result.Semver.Next

					if ope.Setup.Checkout.Result.CommitVerified {
						callback.SigningKeyCallback.SignKey = ope.Setup.Checkout.Result.SignKeyID
					} else {
						callback.SigningKeyCallback.SignKey = ope.Setup.Checkout.Result.SignKeyID
						callback.SigningKeyCallback.Error = ope.Setup.Checkout.Result.Msg + fmt.Sprintf("(Operation ID: %s)", ope.UUID)
					}
				} else {
					callback.SigningKeyCallback.Error = ope.Error.Message + fmt.Sprintf("(Operation ID: %s)", ope.UUID)
				}

				if _, code, err := services.NewClient(srvs).DoJSONRequest(ctx, http.MethodPost, "/v2/repository/event/callback", callback, nil); err != nil {
					log.ErrorWithStackTrace(ctx, sdk.WrapError(err, "unable to send analysis call to hook [HTTP: %d]", code))
					return
				}
			})
			return service.WriteJSON(w, ope, http.StatusOK)
		}
}

func (api *API) postRetrieveWorkflowToTriggerHandler() ([]service.RbacChecker, service.Handler) {
	return service.RBAC(api.isHookService),
		func(ctx context.Context, w http.ResponseWriter, req *http.Request) error {

			var hookRequest sdk.HookListWorkflowRequest
			if err := service.UnmarshalRequest(ctx, req, &hookRequest); err != nil {
				return err
			}

			ctx = context.WithValue(ctx, cdslog.HookEventID, hookRequest.HookEventUUID)

			db := api.mustDB()

			uniqueWorkflowMap := make(map[string]struct{})
			filteredWorkflowHooks := make([]sdk.V2WorkflowHook, 0)

			// Get repository web hooks
			workflowHooks, err := LoadWorkflowHooksWithRepositoryWebHooks(ctx, db, hookRequest)
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
			workflowUpdateHooks, err := LoadWorkflowHooksWithWorkflowUpdate(ctx, db, hookRequest)
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
			modelUpdateHooks, err := LoadWorkflowHooksWithModelUpdate(ctx, db, hookRequest)
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

			hooksWithReadRight := make([]sdk.V2WorkflowHook, 0)
			for _, h := range filteredWorkflowHooks {
				if !hookRequest.AnayzedProjectKeys.Contains(h.ProjectKey) {
					// Check project right
					vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, db, api.Cache, h.ProjectKey, hookRequest.VCSName)
					if err != nil {
						return err
					}
					if _, err := vcsClient.RepoByFullname(ctx, hookRequest.RepositoryName); err != nil {
						log.Info(ctx, "hook %s of type %s on project %s workflow %s has no right on repository %s/%s: %v", h.ID, h.Type, h.ProjectKey, h.WorkflowName, hookRequest.VCSName, hookRequest.RepositoryName, err)
						continue
					}
				}
				hooksWithReadRight = append(hooksWithReadRight, h)
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
		if h.Ref == hookRequest.Ref {
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
		if hookRequest.Ref == h.Ref {
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
			if w.Ref != hookRequest.Ref {
				continue
			}
		}

		// Check configuration : branch filter + path filter
		switch hookRequest.RepositoryEventName {
		case sdk.WorkflowHookEventPush:
			validBranch := sdk.IsValidHookRefs(ctx, w.Data.BranchFilter, strings.TrimPrefix(hookRequest.Ref, sdk.GitRefBranchPrefix))
			validTag := sdk.IsValidHookRefs(ctx, w.Data.TagFilter, strings.TrimPrefix(hookRequest.Ref, sdk.GitRefTagPrefix))
			validPath := sdk.IsValidHookPath(ctx, w.Data.PathFilter, hookRequest.Paths)
			if validBranch && validPath && validTag {
				filteredWorkflowHooks = append(filteredWorkflowHooks, w)
			}
			continue
		case sdk.WorkflowHookEventPullRequest, sdk.WorkflowHookEventPullRequestComment:
			validBranch := sdk.IsValidHookRefs(ctx, w.Data.BranchFilter, strings.TrimPrefix(hookRequest.Ref, sdk.GitRefBranchPrefix))
			validTag := sdk.IsValidHookRefs(ctx, w.Data.TagFilter, strings.TrimPrefix(hookRequest.Ref, sdk.GitRefTagPrefix))
			validPath := sdk.IsValidHookPath(ctx, w.Data.PathFilter, hookRequest.Paths)
			validType := true
			if len(w.Data.TypesFilter) > 0 {
				validType = sdk.IsInArray(hookRequest.RepositoryEventType, w.Data.TypesFilter)
			}
			if validBranch && validPath && validTag && validType {
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

			db := api.mustDB()

			// Check if project has read access
			vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, db, api.Cache, pKey, vcsName)
			if err != nil {
				return err
			}
			if _, err := vcsClient.RepoByFullname(ctx, repositoryName); err != nil {
				return err
			}

			srvs, err := services.LoadAllByType(ctx, db, sdk.TypeHooks)
			if err != nil {
				return err
			}
			if len(srvs) < 1 {
				return sdk.NewErrorFrom(sdk.ErrNotFound, "unable to find hook uservice")
			}
			path := fmt.Sprintf("/v2/repository/key/%s/%s", vcsName, url.PathEscape(repositoryName))

			var keyResp sdk.GenerateRepositoryWebhook
			_, code, err := services.NewClient(srvs).DoJSONRequest(ctx, http.MethodGet, path, nil, &keyResp)
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
