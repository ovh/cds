package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fsamin/go-dump"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func computeHookToDelete(newWorkflow *sdk.Workflow, oldWorkflow *sdk.Workflow) map[string]*sdk.NodeHook {
	hookToDelete := make(map[string]*sdk.NodeHook)
	currentHooks := newWorkflow.WorkflowData.GetHooks()
	for k, h := range oldWorkflow.WorkflowData.GetHooks() {
		if _, has := currentHooks[k]; !has {
			hookToDelete[k] = h
		}
	}
	return hookToDelete
}

func hookUnregistration(ctx context.Context, db gorp.SqlExecutor, store cache.Store, p *sdk.Project, hookToDelete map[string]*sdk.NodeHook) error {
	if len(hookToDelete) == 0 {
		return nil
	}

	// Delete from vcs configuration if needed
	for _, h := range hookToDelete {
		if h.HookModelName == sdk.RepositoryWebHookModelName {
			// Call VCS to know if repository allows webhook and get the configuration fields
			projectVCSServer := repositoriesmanager.GetProjectVCSServer(p, h.Config["vcsServer"].Value)
			if projectVCSServer != nil {
				client, errclient := repositoriesmanager.AuthorizedClient(ctx, db, store, p.Key, projectVCSServer)
				if errclient != nil {
					return errclient
				}
				vcsHook := sdk.VCSHook{
					Method:   "POST",
					URL:      h.Config["webHookURL"].Value,
					Workflow: true,
					ID:       h.Config["webHookID"].Value,
				}
				if err := client.DeleteHook(ctx, h.Config["repoFullName"].Value, vcsHook); err != nil {
					log.Error("deleteHookConfiguration> Cannot delete hook on repository %s", err)
				}
			}
		}
	}

	//Push the hook to hooks µService
	//Load service "hooks"
	srvs, err := services.FindByType(db, services.TypeHooks)
	if err != nil {
		return err
	}
	_, code, errHooks := services.DoJSONRequest(ctx, srvs, http.MethodDelete, "/task/bulk", hookToDelete, nil)
	if errHooks != nil || code >= 400 {
		// if we return an error, transaction will be rollbacked => hook will in database be not anymore on gitlab/bitbucket/github.
		// so, it's just a warn log
		log.Error("HookRegistration> unable to delete old hooks [%d]: %s", code, errHooks)
	}
	return nil
}

func hookRegistration(ctx context.Context, db gorp.SqlExecutor, store cache.Store, p *sdk.Project, wf *sdk.Workflow, oldWorkflow *sdk.Workflow) error {
	var oldHooks map[string]*sdk.NodeHook
	var oldHooksByRef map[string]sdk.NodeHook
	if oldWorkflow != nil {
		oldHooks = oldWorkflow.WorkflowData.GetHooks()
		oldHooksByRef = oldWorkflow.WorkflowData.GetHooksMapRef()
	}
	if len(wf.WorkflowData.Node.Hooks) <= 0 {
		return nil
	}

	srvs, err := services.FindByType(db, services.TypeHooks)
	if err != nil {
		return sdk.WrapError(err, "unable to get services dao")
	}

	//Perform the request on one off the hooks service
	if len(srvs) < 1 {
		return sdk.WrapError(fmt.Errorf("no hooks service available, please try again"), "Unable to get services dao")
	}

	hookToUpdate := make(map[string]sdk.NodeHook)
	for i := range wf.WorkflowData.Node.Hooks {
		h := &wf.WorkflowData.Node.Hooks[i]
		if h.UUID == "" && h.Ref == "" {
			h.Ref = fmt.Sprintf("%s.%d", wf.WorkflowData.Node.Name, i)
		} else if h.Ref != "" && oldHooksByRef != nil {
			// search previous hook configuration by ref
			previousHook, has := oldHooksByRef[h.Ref]
			h.UUID = previousHook.UUID
			// If previous hook is the same, we do nothing
			if has && h.Equals(previousHook) {
				continue
			}
		} else if oldHooks != nil {
			// search previous hook configuration by uuid
			previousHook, has := oldHooks[h.UUID]
			// If previous hook is the same, we do nothing
			if has && h.Equals(*previousHook) {
				continue
			}
		}
		// initialize a UUID is there no uuid
		if h.UUID == "" {
			h.UUID = sdk.UUID()
		}

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
		if h.HookModelName == sdk.RepositoryWebHookModelName || h.HookModelName == sdk.GitPollerModelName || h.HookModelName == sdk.GerritHookModelName {
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

		if err := updateSchedulerPayload(ctx, db, store, p, wf, h); err != nil {
			return err
		}
		hookToUpdate[h.UUID] = *h
	}

	if len(hookToUpdate) > 0 {
		// Create hook on µservice
		_, code, errHooks := services.DoJSONRequest(ctx, srvs, http.MethodPost, "/task/bulk", hookToUpdate, &hookToUpdate)
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
			v, ok := h.Config["webHookID"]
			if h.HookModelName == sdk.RepositoryWebHookModelName && h.Config["vcsServer"].Value != "" && (!ok || v.Value == "") {
				if err := createVCSConfiguration(ctx, db, store, p, h); err != nil {
					return sdk.WrapError(err, "Cannot update vcs configuration")
				}
			}
		}
	}

	return nil
}

func updateSchedulerPayload(ctx context.Context, db gorp.SqlExecutor, store cache.Store, p *sdk.Project, wf *sdk.Workflow, h *sdk.NodeHook) error {
	if h.HookModelName != sdk.SchedulerModelName {
		return nil
	}
	// Add git.branch in scheduler payload
	if wf.WorkflowData.Node.IsLinkedToRepo(wf) {
		var payloadValues map[string]string
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
			defaultPayload, errDefault := DefaultPayload(ctx, db, store, p, wf)
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

func createVCSConfiguration(ctx context.Context, db gorp.SqlExecutor, store cache.Store, p *sdk.Project, h *sdk.NodeHook) error {
	ctx, end := observability.Span(ctx, "workflow.createVCSConfiguration", observability.Tag("UUID", h.UUID))
	defer end()
	// Call VCS to know if repository allows webhook and get the configuration fields
	projectVCSServer := repositoriesmanager.GetProjectVCSServer(p, h.Config["vcsServer"].Value)
	if projectVCSServer == nil {
		return nil
	}

	client, errclient := repositoriesmanager.AuthorizedClient(ctx, db, store, p.Key, projectVCSServer)
	if errclient != nil {
		return sdk.WrapError(errclient, "createVCSConfiguration> Cannot get vcs client")
	}
	webHookInfo, errWH := repositoriesmanager.GetWebhooksInfos(ctx, client)
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
	if err := client.CreateHook(ctx, h.Config["repoFullName"].Value, &vcsHook); err != nil {
		return sdk.WrapError(err, "Cannot create hook on repository: %+v", vcsHook)
	}
	observability.Current(ctx, observability.Tag("VCS_ID", vcsHook.ID))
	h.Config["webHookID"] = sdk.WorkflowNodeHookConfigValue{
		Value:        vcsHook.ID,
		Configurable: false,
	}
	h.Config[sdk.HookConfigIcon] = sdk.WorkflowNodeHookConfigValue{
		Value:        webHookInfo.Icon,
		Configurable: false,
		Type:         sdk.HookConfigTypeString,
	}

	return nil
}

// DefaultPayload returns the default payload for the workflow root
func DefaultPayload(ctx context.Context, db gorp.SqlExecutor, store cache.Store, p *sdk.Project, wf *sdk.Workflow) (interface{}, error) {
	if wf.WorkflowData.Node.Context == nil || wf.WorkflowData.Node.Context.ApplicationID == 0 {
		return nil, nil
	}

	var defaultPayload interface{}

	app := wf.Applications[wf.WorkflowData.Node.Context.ApplicationID]

	if app.RepositoryFullname != "" {
		defaultBranch := "master"
		projectVCSServer := repositoriesmanager.GetProjectVCSServer(p, app.VCSServer)
		if projectVCSServer != nil {
			client, errclient := repositoriesmanager.AuthorizedClient(ctx, db, store, p.Key, projectVCSServer)
			if errclient != nil {
				return wf.WorkflowData.Node.Context.DefaultPayload, sdk.WrapError(errclient, "DefaultPayload> Cannot get authorized client")
			}

			branches, errBr := client.Branches(ctx, app.RepositoryFullname)
			if errBr != nil {
				return wf.WorkflowData.Node.Context.DefaultPayload, sdk.WrapError(errBr, "DefaultPayload> Cannot get branches for %s", app.RepositoryFullname)
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
