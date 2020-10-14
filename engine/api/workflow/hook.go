package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/fsamin/go-dump"

	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

func computeHookToDelete(newWorkflow *sdk.Workflow, oldWorkflow *sdk.Workflow) map[string]sdk.NodeHook {
	hookToDelete := make(map[string]sdk.NodeHook)
	currentHooks := newWorkflow.WorkflowData.GetHooks()
	for k, h := range oldWorkflow.WorkflowData.GetHooks() {
		if _, has := currentHooks[k]; !has {
			hookToDelete[k] = *h
		}
	}
	return hookToDelete
}

func hookUnregistration(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, hookToDelete map[string]sdk.NodeHook) error {
	ctx, end := telemetry.Span(ctx, "workflow.hookUnregistration")
	defer end()

	if len(hookToDelete) == 0 {
		return nil
	}

	// Delete from vcs configuration if needed
	for _, h := range hookToDelete {
		if h.HookModelName == sdk.RepositoryWebHookModelName {
			// Call VCS to know if repository allows webhook and get the configuration fields
			projectVCSServer, err := repositoriesmanager.LoadProjectVCSServerLinkByProjectKeyAndVCSServerName(ctx, db, proj.Key, h.Config["vcsServer"].Value)
			if err == nil {
				client, errclient := repositoriesmanager.AuthorizedClient(ctx, db, store, proj.Key, projectVCSServer)
				if errclient != nil {
					return errclient
				}
				vcsHook := sdk.VCSHook{
					Method:   "POST",
					URL:      h.Config["webHookURL"].Value,
					Workflow: true,
					ID:       h.Config[sdk.HookConfigWebHookID].Value,
				}
				if err := client.DeleteHook(ctx, h.Config["repoFullName"].Value, vcsHook); err != nil {
					log.Error(ctx, "hookUnregistration> Cannot delete hook on repository %s", err)
				}
			}
		}
	}

	//Push the hook to hooks µService
	//Load service "hooks"
	srvs, err := services.LoadAllByType(ctx, db, sdk.TypeHooks)
	if err != nil {
		return err
	}
	_, code, errHooks := services.NewClient(db, srvs).DoJSONRequest(ctx, http.MethodDelete, "/task/bulk", hookToDelete, nil)
	if errHooks != nil || code >= 400 {
		// if we return an error, transaction will be rollbacked => hook will in database be not anymore on gitlab/bitbucket/github.
		// so, it's just a warn log
		log.Error(ctx, "hookUnregistration> unable to delete old hooks [%d]: %s", code, errHooks)
	}
	return nil
}

func hookRegistration(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, wf *sdk.Workflow, oldWorkflow *sdk.Workflow) error {
	ctx, end := telemetry.Span(ctx, "workflow.hookRegistration")
	defer end()

	var oldHooks map[string]*sdk.NodeHook
	var oldHooksByRef map[string]sdk.NodeHook
	if oldWorkflow != nil {
		oldHooks = oldWorkflow.WorkflowData.GetHooks()
		oldHooksByRef = oldWorkflow.WorkflowData.GetHooksMapRef()
	}
	if len(wf.WorkflowData.Node.Hooks) <= 0 {
		return nil
	}

	srvs, err := services.LoadAllByType(ctx, db, sdk.TypeHooks)
	if err != nil {
		return sdk.WrapError(err, "unable to get services")
	}

	//Perform the request on one off the hooks service
	if len(srvs) < 1 {
		return sdk.WithStack(fmt.Errorf("no hooks service available, please try again"))
	}

	hookToUpdate := make(map[string]sdk.NodeHook)
	for i := range wf.WorkflowData.Node.Hooks {
		h := &wf.WorkflowData.Node.Hooks[i]

		h.Config[sdk.HookConfigProject] = sdk.WorkflowNodeHookConfigValue{
			Value:        wf.ProjectKey,
			Configurable: false,
		}
		h.Config[sdk.HookConfigWorkflow] = sdk.WorkflowNodeHookConfigValue{
			Value:        wf.Name,
			Configurable: false,
		}
		h.Config[sdk.HookConfigWorkflowID] = sdk.WorkflowNodeHookConfigValue{
			Value:        fmt.Sprint(wf.ID),
			Configurable: false,
		}

		if h.UUID == "" && oldHooksByRef != nil {
			// search previous hook configuration by ref
			previousHook, has := oldHooksByRef[h.Ref()]
			if has {
				h.UUID = previousHook.UUID
				// If previous hook is the same, we do nothing
				if h.Equals(previousHook) {
					continue
				}
			}
		} else if oldHooks != nil {
			// search previous hook configuration by uuid
			previousHook, has := oldHooks[h.UUID]
			// If previous hook is the same, we do nothing
			if has && h.Equals(*previousHook) {
				// If this a repowebhook with an empty eventFilter, let's keep the old one because vcs won't be called to get the default eventFilter
				eventFilter, has := h.GetConfigValue(sdk.HookConfigEventFilter)
				if previousHook.IsRepositoryWebHook() && h.IsRepositoryWebHook() &&
					(!has || eventFilter == "") {
					h.Config[sdk.HookConfigEventFilter] = previousHook.Config[sdk.HookConfigEventFilter]
				}
				continue
			}

		}
		// initialize a UUID is there no uuid
		if h.UUID == "" {
			h.UUID = sdk.UUID()
		}

		if h.IsRepositoryWebHook() || h.HookModelName == sdk.GitPollerModelName || h.HookModelName == sdk.GerritHookModelName {
			if wf.WorkflowData.Node.Context.ApplicationID == 0 || wf.Applications[wf.WorkflowData.Node.Context.ApplicationID].RepositoryFullname == "" || wf.Applications[wf.WorkflowData.Node.Context.ApplicationID].VCSServer == "" {
				return sdk.NewErrorFrom(sdk.ErrForbidden, "cannot create a git poller or repository webhook on an application without a repository")
			}
			h.Config[sdk.HookConfigVCSServer] = sdk.WorkflowNodeHookConfigValue{
				Value:        wf.Applications[wf.WorkflowData.Node.Context.ApplicationID].VCSServer,
				Configurable: false,
			}
			h.Config[sdk.HookConfigRepoFullName] = sdk.WorkflowNodeHookConfigValue{
				Value:        wf.Applications[wf.WorkflowData.Node.Context.ApplicationID].RepositoryFullname,
				Configurable: false,
			}
		}

		if err := updateSchedulerPayload(ctx, db, store, proj, wf, h); err != nil {
			return err
		}
		hookToUpdate[h.UUID] = *h
		log.Debug("workflow.hookrRegistration> following hook must be updated: %+v", h)
	}

	if len(hookToUpdate) > 0 {
		// Create hook on µservice
		_, code, errHooks := services.NewClient(db, srvs).DoJSONRequest(ctx, http.MethodPost, "/task/bulk", hookToUpdate, &hookToUpdate)
		if errHooks != nil || code >= 400 {
			return sdk.WrapError(errHooks, "unable to create hooks [%d]", code)
		}

		hooks := wf.WorkflowData.GetHooks()
		for i := range hookToUpdate {
			hooks[i].Config = hookToUpdate[i].Config
		}

		// Create vcs configuration ( always after hook creation to have webhook URL) + update hook in DB
		for i := range wf.WorkflowData.Node.Hooks {
			h := &wf.WorkflowData.Node.Hooks[i]
			// Manage VCSconfigation only for updated hooks
			if _, isUpdated := hookToUpdate[h.UUID]; !isUpdated {
				continue
			}
			v, ok := h.Config[sdk.HookConfigWebHookID]
			if h.IsRepositoryWebHook() {
				log.Debug("workflow.hookRegistration> managing vcs configuration: %+v", h)
			}
			if h.IsRepositoryWebHook() && h.Config["vcsServer"].Value != "" {
				if !ok || v.Value == "" {
					if err := createVCSConfiguration(ctx, db, store, proj, h); err != nil {
						return sdk.WithStack(err)
					}
				}
				if ok && v.Value != "" {
					if err := updateVCSConfiguration(ctx, db, store, proj, h); err != nil {
						// hook not found on VCS, perhaps manually deleted on vcs
						// we try to create a new hook
						if sdk.ErrorIs(err, sdk.ErrNotFound) {
							log.Warning(ctx, "hook %s not found on %s/%s", v.Value, h.Config["vcsServer"].Value, h.Config["repoFullName"].Value)
							if err := createVCSConfiguration(ctx, db, store, proj, h); err != nil {
								return err
							}
						} else {
							return sdk.WithStack(err)
						}
					}
				}
			}
		}
	}

	return nil
}

func updateSchedulerPayload(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, wf *sdk.Workflow, h *sdk.NodeHook) error {
	ctx, end := telemetry.Span(ctx, "workflow.updateSchedulerPayload")
	defer end()

	if h.HookModelName != sdk.SchedulerModelName {
		return nil
	}
	// Add git.branch in scheduler payload
	if wf.WorkflowData.Node.IsLinkedToRepo(wf) {
		payloadValues := make(map[string]string)
		if h.Config["payload"].Value != "" {
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
			var errDump error
			payloadValues, errDump = dump.ToStringMap(bodyJSON)
			if errDump != nil {
				return sdk.WrapError(errDump, "cannot dump payload %+v", h.Config["payload"].Value)
			}
		}

		// try get git.branch on defaultPayload
		if payloadValues["git.branch"] == "" {
			defaultPayloadMap, errP := wf.WorkflowData.Node.Context.DefaultPayloadToMap()
			if errP != nil {
				return sdk.WrapError(errP, "cannot read node default payload")
			}
			if defaultPayloadMap["WorkflowNodeContextDefaultPayloadVCS.GitBranch"] != "" {
				payloadValues["git.branch"] = defaultPayloadMap["WorkflowNodeContextDefaultPayloadVCS.GitBranch"]
			}
			if defaultPayloadMap["WorkflowNodeContextDefaultPayloadVCS.GitRepository"] != "" {
				payloadValues["git.repository"] = defaultPayloadMap["WorkflowNodeContextDefaultPayloadVCS.GitRepository"]
			}
		}

		// try get git.branch on repo linked
		if payloadValues["git.branch"] == "" {
			defaultPayload, errDefault := DefaultPayload(ctx, db, store, proj, wf)
			if errDefault != nil {
				return sdk.WrapError(errDefault, "unable to get default payload")
			}
			var errDump error
			payloadValues, errDump = dump.ToStringMap(defaultPayload)
			if errDump != nil {
				return sdk.WrapError(errDump, "cannot dump payload %+v", h.Config["payload"].Value)
			}
		}

		payloadStr, errM := json.MarshalIndent(&payloadValues, "", "  ")
		if errM != nil {
			return sdk.WrapError(errM, "cannot marshal hook config payload : %s", errM)
		}
		pl := h.Config["payload"]
		pl.Value = string(payloadStr)
		h.Config["payload"] = pl
	}
	return nil
}

func createVCSConfiguration(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, h *sdk.NodeHook) error {
	ctx, end := telemetry.Span(ctx, "workflow.createVCSConfiguration", telemetry.Tag("UUID", h.UUID))
	defer end()
	// Call VCS to know if repository allows webhook and get the configuration fields
	projectVCSServer, err := repositoriesmanager.LoadProjectVCSServerLinkByProjectKeyAndVCSServerName(ctx, db, proj.Key, h.Config["vcsServer"].Value)
	if err != nil {
		log.Debug("createVCSConfiguration> No vcsServer found: %v", err)
		return nil
	}

	client, err := repositoriesmanager.AuthorizedClient(ctx, db, store, proj.Key, projectVCSServer)
	if err != nil {
		return sdk.WrapError(err, "cannot get vcs client")
	}
	// We have to check the repository to know if webhooks are supported and how (events)
	webHookInfo, err := repositoriesmanager.GetWebhooksInfos(ctx, client)
	if err != nil {
		return sdk.WrapError(err, "cannot get vcs web hook info")
	}
	if !webHookInfo.WebhooksSupported || webHookInfo.WebhooksDisabled {
		return sdk.NewErrorFrom(sdk.ErrForbidden, "hook creation are forbidden")
	}

	// Check hook config to avoid sending wrong hooks to VCS
	if h.Config["repoFullName"].Value == "" {
		return sdk.WrapError(sdk.ErrInvalidHookConfiguration, "missing repo fullname value for hook")
	}
	if !sdk.IsURL(h.Config["webHookURL"].Value) {
		return sdk.WrapError(sdk.ErrInvalidHookConfiguration, "given webhook url value %s is not a url", h.Config["webHookURL"].Value)
	}

	// Prepare the hook that will be send to VCS
	vcsHook := sdk.VCSHook{
		Method:   "POST",
		URL:      h.Config["webHookURL"].Value,
		Workflow: true,
	}

	// Set given event filters if exists, else default values will be set by CreateHook func.
	if c, ok := h.Config[sdk.HookConfigEventFilter]; ok && c.Value != "" {
		vcsHook.Events = strings.Split(c.Value, ";")
	}

	if err := client.CreateHook(ctx, h.Config["repoFullName"].Value, &vcsHook); err != nil {
		return sdk.WrapError(err, "Cannot create hook on repository: %+v", vcsHook)
	}
	telemetry.Current(ctx, telemetry.Tag("VCS_ID", vcsHook.ID))
	h.Config[sdk.HookConfigWebHookID] = sdk.WorkflowNodeHookConfigValue{
		Value:        vcsHook.ID,
		Configurable: false,
	}
	h.Config[sdk.HookConfigIcon] = sdk.WorkflowNodeHookConfigValue{
		Value:        webHookInfo.Icon,
		Configurable: false,
		Type:         sdk.HookConfigTypeString,
	}
	h.Config[sdk.HookConfigEventFilter] = sdk.WorkflowNodeHookConfigValue{
		Type:         sdk.HookConfigTypeMultiChoice,
		Configurable: true,
		Value:        strings.Join(vcsHook.Events, ";"),
	}
	return nil
}

func updateVCSConfiguration(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, h *sdk.NodeHook) error {
	ctx, end := telemetry.Span(ctx, "workflow.updateVCSConfiguration", telemetry.Tag("UUID", h.UUID))
	defer end()
	// Call VCS to know if repository allows webhook and get the configuration fields
	projectVCSServer, err := repositoriesmanager.LoadProjectVCSServerLinkByProjectKeyAndVCSServerName(ctx, db, proj.Key, h.Config["vcsServer"].Value)
	if err != nil {
		log.Debug("createVCSConfiguration> No vcsServer found: %v", err)
		return nil
	}

	client, err := repositoriesmanager.AuthorizedClient(ctx, db, store, proj.Key, projectVCSServer)
	if err != nil {
		return sdk.WrapError(err, "cannot get vcs client")
	}
	webHookInfo, errWH := repositoriesmanager.GetWebhooksInfos(ctx, client)
	if errWH != nil {
		return sdk.WrapError(errWH, "cannot get vcs web hook info")
	}

	vcsHook := sdk.VCSHook{
		ID:       h.Config[sdk.HookConfigWebHookID].Value,
		Method:   "POST",
		URL:      h.Config["webHookURL"].Value,
		Workflow: true,
	}

	// Set given event filters if exists, else default values will be set by CreateHook func.
	if c, ok := h.Config[sdk.HookConfigEventFilter]; ok && c.Value != "" {
		vcsHook.Events = strings.Split(c.Value, ";")
	}

	if err := client.UpdateHook(ctx, h.Config["repoFullName"].Value, &vcsHook); err != nil {
		return sdk.WrapError(err, "Cannot update hook on repository: %+v", vcsHook)
	}
	h.Config[sdk.HookConfigIcon] = sdk.WorkflowNodeHookConfigValue{
		Value:        webHookInfo.Icon,
		Configurable: false,
		Type:         sdk.HookConfigTypeString,
	}
	h.Config[sdk.HookConfigEventFilter] = sdk.WorkflowNodeHookConfigValue{
		Type:         sdk.HookConfigTypeMultiChoice,
		Configurable: true,
		Value:        strings.Join(vcsHook.Events, ";"),
	}
	return nil
}

// DefaultPayload returns the default payload for the workflow root
func DefaultPayload(ctx context.Context, db gorpmapper.SqlExecutorWithTx, store cache.Store, proj sdk.Project, wf *sdk.Workflow) (interface{}, error) {
	if wf.WorkflowData.Node.Context == nil || wf.WorkflowData.Node.Context.ApplicationID == 0 {
		return nil, nil
	}

	var defaultPayload interface{}

	app := wf.Applications[wf.WorkflowData.Node.Context.ApplicationID]

	if app.RepositoryFullname != "" {
		defaultBranch := "master"
		projectVCSServer, err := repositoriesmanager.LoadProjectVCSServerLinkByProjectKeyAndVCSServerName(ctx, db, proj.Key, app.VCSServer)
		if err == nil {
			client, err := repositoriesmanager.AuthorizedClient(ctx, db, store, proj.Key, projectVCSServer)
			if err != nil {
				return wf.WorkflowData.Node.Context.DefaultPayload, sdk.WrapError(err, "cannot get authorized client")
			}

			branches, err := client.Branches(ctx, app.RepositoryFullname)
			if err != nil {
				return wf.WorkflowData.Node.Context.DefaultPayload, err
			}

			for _, branch := range branches {
				if branch.Default {
					defaultBranch = branch.DisplayID
					break
				}
			}
		}

		defaultPayload = wf.WorkflowData.Node.Context.DefaultPayload
		if !wf.WorkflowData.Node.Context.HasDefaultPayload() {
			structuredDefaultPayload := sdk.WorkflowNodeContextDefaultPayloadVCS{
				GitBranch:     defaultBranch,
				GitRepository: app.RepositoryFullname,
			}
			defaultPayloadBtes, _ := json.Marshal(structuredDefaultPayload)
			if err := json.Unmarshal(defaultPayloadBtes, &defaultPayload); err != nil {
				return nil, err
			}
		} else if defaultPayloadMap, err := wf.WorkflowData.Node.Context.DefaultPayloadToMap(); err == nil && defaultPayloadMap["git.branch"] == "" {
			defaultPayloadMap["git.branch"] = defaultBranch
			defaultPayloadMap["git.repository"] = app.RepositoryFullname
			defaultPayload = defaultPayloadMap
		}
	} else {
		defaultPayload = wf.WorkflowData.Node.Context.DefaultPayload
	}

	return defaultPayload, nil
}
