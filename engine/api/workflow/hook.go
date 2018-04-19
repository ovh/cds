package workflow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fsamin/go-dump"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// HookRegistration ensures hooks registration on Hook µService
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
		srvs, err := dao.FindByType(services.TypeHooks)
		if err != nil {
			return sdk.WrapError(err, "HookRegistration> Unable to get services dao")
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
			return sdk.WrapError(fmt.Errorf("HookRegistration> No hooks service available, please try again"), "Unable to get services dao")
		}

		// Update scheduler payload
		for i := range hookToUpdate {
			h := hookToUpdate[i]

			if h.WorkflowHookModel.Name == sdk.SchedulerModelName {
				// Add git.branch in scheduler payload
				if wf.Root.IsLinkedToRepo() && h.Config["payload"].Value != "" {
					var bodyJSON interface{}
					//Try to parse the body as an array
					bodyJSONArray := []interface{}{}
					if err := json.Unmarshal([]byte(h.Config["payload"].Value), &bodyJSONArray); err != nil {
						//Try to parse the body as a map
						bodyJSONMap := map[string]interface{}{}
						if err2 := json.Unmarshal([]byte(h.Config["payload"].Value), &bodyJSONMap); err2 == nil {
							bodyJSON = bodyJSONMap
						}
					} else {
						bodyJSON = bodyJSONArray
					}

					//Go Dump
					e := dump.NewDefaultEncoder(new(bytes.Buffer))
					e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
					e.ExtraFields.DetailedMap = false
					e.ExtraFields.DetailedStruct = false
					e.ExtraFields.DeepJSON = true
					e.ExtraFields.Len = false
					e.ExtraFields.Type = false
					payloadValues, errDump := e.ToStringMap(bodyJSON)
					if errDump != nil {
						return sdk.WrapError(errDump, "HookRegistration> Cannot dump payload %+v", h.Config["payload"].Value)
					}

					if payloadValues["git.branch"] == "" {
						defaultPayloadMap, errP := wf.Root.Context.DefaultPayloadToMap()
						if errP != nil {
							return sdk.WrapError(errP, "HookRegistration> Cannot read node default payload")
						}
						payloadValues["git.branch"] = defaultPayloadMap["WorkflowNodeContextDefaultPayloadVCS.GitBranch"]

						payloadStr, errM := json.MarshalIndent(&payloadValues, "", "  ")
						if errM != nil {
							return sdk.WrapError(errM, "HookRegistration> Cannot marshal hook config payload : %s", errM)
						}
						pl := h.Config["payload"]
						pl.Value = string(payloadStr)
						h.Config["payload"] = pl
						hookToUpdate[i] = h
					}
				}
			}
		}

		// Create hook on µservice
		code, errHooks := services.DoJSONRequest(srvs, http.MethodPost, "/task/bulk", hookToUpdate, &hookToUpdate)
		if errHooks != nil || code >= 400 {
			return sdk.WrapError(errHooks, "HookRegistration> Unable to create hooks [%d]", code)
		}

		// Create vcs configuration ( always after hook creation to have webhook URL) + update hook in DB
		for i := range hookToUpdate {
			h := hookToUpdate[i]
			v, ok := h.Config["webHookID"]
			if h.WorkflowHookModel.Name == sdk.RepositoryWebHookModelName && h.Config["vcsServer"].Value != "" && (!ok || v.Value == "") {
				if err := createVCSConfiguration(db, store, p, &h); err != nil {
					return sdk.WrapError(err, "HookRegistration> Cannot update vcs configuration")
				}
			}

			if err := UpdateHook(db, &h); err != nil {
				return sdk.WrapError(err, "HookRegistration> Cannot update hook")
			}
		}
	}

	if len(hookToDelete) > 0 {
		if err := deleteHookConfiguration(db, store, p, hookToDelete); err != nil {
			return sdk.WrapError(err, "HookRegistration> Cannot remove hook configuration")
		}
	}
	return nil
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
