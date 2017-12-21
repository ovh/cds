package workflow

import (
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
		srvs, err := dao.FindByType("hooks")
		if err != nil {
			return nil, sdk.WrapError(err, "HookRegistration> Unable to get services dao")
		}

		// Update in VCS
		for i := range hookToUpdate {
			h := hookToUpdate[i]
			if oldW != nil && wf.Name != oldW.Name {
				configValue := h.Config["workflow"]
				configValue.Value = wf.Name
				h.Config["workflow"] = configValue
				hookToUpdate[i] = h
			}
		}

		//Perform the request on one off the hooks service
		if len(srvs) < 1 {
			return nil, sdk.WrapError(fmt.Errorf("HookRegistration> No hooks service available, please try again"), "Unable to get services dao")
		}

		var hooksUpdated map[string]sdk.WorkflowNodeHook
		code, errHooks := services.DoJSONRequest(srvs, http.MethodPost, "/task/bulk", hookToUpdate, &hooksUpdated)
		if errHooks != nil || code >= 400 {
			return nil, sdk.WrapError(errHooks, "HookRegistration> Unable to create hooks [%d]", code)
		}

		for i := range hooksUpdated {
			h := hooksUpdated[i]
			if h.Config["vcsServer"].Value != "" {
				if err := createVCSConfiguration(db, store, p, &h); err != nil {
					return nil, sdk.WrapError(err, "HookRegistration> Cannot update vcs configuration")
				}
				defaultPayload = &sdk.WorkflowNodeContextDefaultPayloadVCS{
					GitRepository: h.Config["repoFullName"].Value,
				}
			}
			if err := UpdateHook(db, &h); err != nil {
				return nil, sdk.WrapError(err, "HookRegistration> Cannot update hook")
			}
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
		if h.WorkflowHookModel.Name == RepositoryWebHookModel.Name {
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
					return sdk.WrapError(err, "deleteHookConfiguration> Cannot delete hook on repository")
				}
				h.Config["webHookID"] = sdk.WorkflowNodeHookConfigValue{
					Value:        vcsHook.ID,
					Configurable: false,
				}
			}
		}
		return nil
	}

	//Push the hook to hooks µService
	dao := services.Querier(db, store)
	//Load service "hooks"
	srvs, err := dao.FindByType("hooks")
	if err != nil {
		return sdk.WrapError(err, "HookRegistration> Unable to get services dao")
	}
	code, errHooks := services.DoJSONRequest(srvs, http.MethodDelete, fmt.Sprintf("/task/bulk"), hookToDelete, nil)
	if errHooks != nil || code >= 400 {
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

	for kNew := range newHooks {
		hold, ok := oldHooks[kNew]
		// if new hook
		if !ok {
			hookToUpdate[kNew] = newHooks[kNew]
			continue
		}

	next:
		for k, v := range newHooks[kNew].Config {
			for kold, vold := range hold.Config {
				if kold == k && v != vold {
					hookToUpdate[kNew] = newHooks[kNew]
					break next
				}
			}
		}
	}

	for kHold := range oldHooks {
		if _, ok := newHooks[kHold]; !ok {
			hookToDelete[kHold] = oldHooks[kHold]
		}
	}
	return
}
