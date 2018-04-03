package workflow

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// HookRegistration ensures hooks registration on Hook µService
func HookRegistration(db gorp.SqlExecutor, store cache.Store, oldW *sdk.Workflow, wf sdk.Workflow, p *sdk.Project) (*sdk.WorkflowNodeContextDefaultPayloadVCS, error) {
	var hookToUpdate map[string]sdk.WorkflowNodeHook
	var hookToDelete map[string]sdk.WorkflowNodeHook

	if oldW != nil {
		hookToUpdate, hookToDelete = diffHook(oldW.GetHooks(), wf.GetHooks())
	} else {
		hookToUpdate = wf.GetHooks()
	}

	var defaultPayload *sdk.WorkflowNodeContextDefaultPayloadVCS

	if len(hookToUpdate) > 0 {
		//Push the hook to hooks µService
		dao := services.Querier(db, store)
		//Load service "hooks"
		srvs, err := dao.FindByType(services.TypeHooks)
		if err != nil {
			return nil, sdk.WrapError(err, "HookRegistration> Unable to get services dao")
		}

		// Update in VCS
		for i := range hookToUpdate {
			h := hookToUpdate[i]
			if oldW != nil && wf.Name != oldW.Name {
				configValue := h.Config[sdk.HookConfigWorkflow]
				configValue.Value = wf.Name
				h.Config[sdk.HookConfigWorkflow] = configValue
				hookToUpdate[i] = h
			}
		}

		//Perform the request on one off the hooks service
		if len(srvs) < 1 {
			return nil, sdk.WrapError(fmt.Errorf("HookRegistration> No hooks service available, please try again"), "Unable to get services dao")
		}

		for i := range hookToUpdate {
			h := hookToUpdate[i]
			v, ok := h.Config["webHookID"]
			if h.WorkflowHookModel.Name == sdk.SchedulerModelName {
				// Add git.branch in scheduler payload
				if wf.Root.Context != nil && wf.Root.Context.Application != nil && wf.Root.Context.Application.RepositoryFullname != "" && h.Config["payload"].Value != "" {
					payload := map[string]string{}
					if err := json.Unmarshal([]byte(h.Config["payload"].Value), &payload); err != nil {
						return nil, sdk.WrapError(err, "HookRegistration> Unable to unmarshall payload")
					}

					if payload["git.branch"] == "" {
						defaultBranch := "master"
						projectVCSServer := repositoriesmanager.GetProjectVCSServer(p, wf.Root.Context.Application.VCSServer)
						if projectVCSServer != nil {
							client, errclient := repositoriesmanager.AuthorizedClient(db, store, projectVCSServer)
							if errclient != nil {
								return nil, sdk.WrapError(errclient, "HookRegistration> Cannot get authorized client")
							}

							branches, errBr := client.Branches(wf.Root.Context.Application.RepositoryFullname)
							if errBr != nil {
								return nil, sdk.WrapError(errBr, "HookRegistration> Cannot get branches for %s", wf.Root.Context.Application.RepositoryFullname)
							}

							for _, branch := range branches {
								if branch.Default {
									defaultBranch = branch.DisplayID
									break
								}
							}
							payload["git.branch"] = defaultBranch
							if _, ok := payload[""]; ok {
								delete(payload, "")
							}
							payloadStr, errM := json.Marshal(&payload)
							if errM != nil {
								return nil, sdk.WrapError(errM, "HookRegistration> Cannot marshal hook config payload : %s", errM)
							}
							pl := h.Config["payload"]
							pl.Value = string(payloadStr)
							h.Config["payload"] = pl
						}
					}

				}
			} else if h.WorkflowHookModel.Name == sdk.RepositoryWebHookModelName && h.Config["vcsServer"].Value != "" && (!ok || v.Value == "") {
				if err := createVCSConfiguration(db, store, p, &h); err != nil {
					return nil, sdk.WrapError(err, "HookRegistration> Cannot update vcs configuration")
				}
				defaultBranch := "master"
				projectVCSServer := repositoriesmanager.GetProjectVCSServer(p, h.Config["vcsServer"].Value)
				if projectVCSServer != nil {
					client, errclient := repositoriesmanager.AuthorizedClient(db, store, projectVCSServer)
					if errclient != nil {
						return nil, sdk.WrapError(errclient, "HookRegistration> Cannot get authorized client from repository manager")
					}

					branches, errBr := client.Branches(h.Config["repoFullName"].Value)
					if errBr != nil {
						return nil, sdk.WrapError(errBr, "HookRegistration> Cannot get branches from repository manager %s", h.Config["repoFullName"].Value)
					}

					for _, branch := range branches {
						if branch.Default {
							defaultBranch = branch.DisplayID
							break
						}
					}
				}
				defaultPayload = &sdk.WorkflowNodeContextDefaultPayloadVCS{
					GitRepository: h.Config["repoFullName"].Value,
					GitBranch:     defaultBranch,
				}
			}

			if err := UpdateHook(db, &h); err != nil {
				return nil, sdk.WrapError(err, "HookRegistration> Cannot update hook")
			}
		}

		var hooksUpdated map[string]sdk.WorkflowNodeHook
		code, errHooks := services.DoJSONRequest(srvs, http.MethodPost, "/task/bulk", hookToUpdate, &hooksUpdated)
		if errHooks != nil || code >= 400 {
			return nil, sdk.WrapError(errHooks, "HookRegistration> Unable to create hooks [%d]", code)
		}
	}

	if len(hookToDelete) > 0 {
		if err := deleteHookConfiguration(db, store, p, hookToDelete); err != nil {
			return nil, sdk.WrapError(err, "HookRegistration> Cannot remove hook configuration")
		}
	}
	return defaultPayload, nil
}

func deleteHookConfiguration(db gorp.SqlExecutor, store cache.Store, p *sdk.Project, hookToDelete map[string]sdk.WorkflowNodeHook) error {
	// Delete from vcs configuration if needed
	for _, h := range hookToDelete {
		if h.WorkflowHookModel.Name == sdk.RepositoryWebHookModelName {
			// Call VCS to know if repository allows webhook and get the configuration fields
			projectVCSServer := repositoriesmanager.GetProjectVCSServer(p, h.Config["vcsServer"].Value)
			if projectVCSServer != nil {
				client, errclient := repositoriesmanager.AuthorizedClient(db, store, projectVCSServer)
				if errclient != nil {
					return sdk.WrapError(errclient, "deleteHookConfiguration> Cannot get vcs client")
				}
				vcsHook := sdk.VCSHook{
					Method:   "POST",
					URL:      h.Config["webHookURL"].Value,
					Workflow: true,
					ID:       h.Config["webHookID"].Value,
				}
				if err := client.DeleteHook(h.Config["repoFullName"].Value, vcsHook); err != nil {
					log.Error("deleteHookConfiguration> Cannot delete hook on repository %s", err)
				}
				h.Config["webHookID"] = sdk.WorkflowNodeHookConfigValue{
					Value:        vcsHook.ID,
					Configurable: false,
				}
			}
		}
	}

	//Push the hook to hooks µService
	dao := services.Querier(db, store)
	//Load service "hooks"
	srvs, err := dao.FindByType(services.TypeHooks)
	if err != nil {
		return sdk.WrapError(err, "HookRegistration> Unable to get services dao")
	}
	code, errHooks := services.DoJSONRequest(srvs, http.MethodDelete, "/task/bulk", hookToDelete, nil)
	if errHooks != nil || code >= 400 {
		// if we return an error, transaction will be rollbacked => hook will in database be not anymore on gitlab/bitbucket/github.
		// so, it's just a warn log
		log.Warning("HookRegistration> Unable to delete old hooks [%d]: %s", code, errHooks)
	}
	return nil
}

func createVCSConfiguration(db gorp.SqlExecutor, store cache.Store, p *sdk.Project, h *sdk.WorkflowNodeHook) error {
	// Call VCS to know if repository allows webhook and get the configuration fields
	projectVCSServer := repositoriesmanager.GetProjectVCSServer(p, h.Config["vcsServer"].Value)
	if projectVCSServer == nil {
		return nil
	}

	client, errclient := repositoriesmanager.AuthorizedClient(db, store, projectVCSServer)
	if errclient != nil {
		return sdk.WrapError(errclient, "createVCSConfiguration> Cannot get vcs client")
	}
	webHookInfo, errWH := repositoriesmanager.GetWebhooksInfos(client)
	if errWH != nil {
		return sdk.WrapError(errWH, "createVCSConfiguration> Cannot get vcs web hook info")
	}
	if !webHookInfo.WebhooksSupported || webHookInfo.WebhooksDisabled {
		return sdk.WrapError(sdk.ErrForbidden, "createVCSConfiguration> hook creation are forbidden")
	}
	vcsHook := sdk.VCSHook{
		Method:   "POST",
		URL:      h.Config["webHookURL"].Value,
		Workflow: true,
	}
	if err := client.CreateHook(h.Config["repoFullName"].Value, &vcsHook); err != nil {
		return sdk.WrapError(err, "createVCSConfiguration> Cannot create hook on repository: %+v", vcsHook)
	}
	h.Config["webHookID"] = sdk.WorkflowNodeHookConfigValue{
		Value:        vcsHook.ID,
		Configurable: false,
	}

	return nil
}

func diffHook(oldHooks map[string]sdk.WorkflowNodeHook, newHooks map[string]sdk.WorkflowNodeHook) (hookToUpdate map[string]sdk.WorkflowNodeHook, hookToDelete map[string]sdk.WorkflowNodeHook) {
	hookToUpdate = make(map[string]sdk.WorkflowNodeHook)
	hookToDelete = make(map[string]sdk.WorkflowNodeHook)

	for key, hNew := range newHooks {
		hold, ok := oldHooks[key]
		// if new hook
		if !ok || !hNew.Equals(hold) {
			hookToUpdate[key] = newHooks[key]
			continue
		}
	}

	for key := range oldHooks {
		if _, ok := newHooks[key]; !ok {
			hookToDelete[key] = oldHooks[key]
		}
	}
	return
}
