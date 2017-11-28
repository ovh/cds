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

// HookRegistration
func HookRegistration(db gorp.SqlExecutor, store cache.Store, oldW *sdk.Workflow, wf sdk.Workflow, p *sdk.Project) error {
	var hookToUpdate map[string]sdk.WorkflowNodeHook
	var hookToDelete map[string]sdk.WorkflowNodeHook

	if oldW != nil {
		hookToUpdate, hookToDelete = diffHook(oldW.GetHooks(), wf.GetHooks())
	} else {
		hookToUpdate = wf.GetHooks()
	}

	if len(hookToUpdate) > 0 {
		//Push the hook to hooks µService
		dao := services.Querier(db, store)
		//Load service "hooks"
		srvs, err := dao.FindByType("hooks")
		if err != nil {
			return sdk.WrapError(err, "HookRegistration> Unable to get services dao")
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
			return sdk.WrapError(fmt.Errorf("HookRegistration> No hooks service available, please try again"), "Unable to get services dao")
		}

		var hooksUpdated map[string]sdk.WorkflowNodeHook
		code, errHooks := services.DoJSONRequest(srvs, http.MethodPost, "/task/bulk", hookToUpdate, &hooksUpdated)
		if errHooks == nil {
			for _, h := range hooksUpdated {
				if err := UpdateHook(db, &h); err != nil {
					return sdk.WrapError(errHooks, "HookRegistration> Cannot update hook")
				}
			}
			log.Debug("HookRegistration> %d hooks created for workflow %s/%s (HTTP status code %d)", len(hookToUpdate), wf.ProjectKey, wf.Name, code)
		} else {
			return sdk.WrapError(errHooks, "HookRegistration> Unable to create hooks")
		}

		for i := range hooksUpdated {
			h := hooksUpdated[i]
			if h.Config["vcsServer"].Value != "" {
				if err := updateVCSConfiguration(db, store, p, h); err != nil {
					return sdk.WrapError(err, "Cannot update vcs configuration")
				}
			}
		}

	}

	if len(hookToDelete) > 0 {
		//Push the hook to hooks µService
		dao := services.Querier(db, store)
		//Load service "hooks"
		srvs, err := dao.FindByType("hooks")
		if err != nil {
			return sdk.WrapError(err, "HookRegistration> Unable to get services dao")
		}
		code, errHooks := services.DoJSONRequest(srvs, http.MethodDelete, fmt.Sprintf("/task/bulk"), hookToDelete, nil)
		if errHooks != nil || code >= 400 {
			log.Warning("HookRegistration> Unable to delete old hooks")
		}
	}
	return nil
}

func updateVCSConfiguration(db gorp.SqlExecutor, store cache.Store, p *sdk.Project, h sdk.WorkflowNodeHook) error {
	// Call VCS to know if repository allows webhook and get the configuration fields
	projectVCSServer := repositoriesmanager.GetProjectVCSServer(p, h.Config["vcsServer"].Value)
	if projectVCSServer != nil {
		client, errclient := repositoriesmanager.AuthorizedClient(db, store, projectVCSServer)
		if errclient != nil {
			return sdk.WrapError(errclient, "getWorkflowHookModelsHandler> Cannot get vcs client")
		}
		webHookInfo, errWH := repositoriesmanager.GetWebhooksInfos(client)
		if errWH != nil {
			return sdk.WrapError(errWH, "getWorkflowHookModelsHandler> Cannot get vcs web hook info")
		}
		if !webHookInfo.WebhooksSupported || webHookInfo.WebhooksDisabled {
			return sdk.WrapError(sdk.ErrForbidden, "updateVCSConfiguration> hook creation are forbidden")
		}
		vcsHook := sdk.VCSHook{
			Method: "POST",
			URL:    h.Config["webHookURL"].Value,
		}
		if err := client.CreateHook(h.Config["repoFullName"].Value, vcsHook); err != nil {
			return sdk.WrapError(err, "updateVCSConfiguration> Cannot create hook on repository")
		}
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
