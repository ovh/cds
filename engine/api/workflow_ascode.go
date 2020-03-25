package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/ascode"
	"github.com/ovh/cds/engine/api/ascode/sync"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/operation"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getWorkflowAsCodeHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		uuid := vars["uuid"]

		var ope sdk.Operation
		k := cache.Key(operation.CacheOperationKey, uuid)
		b, err := api.Cache.Get(k, &ope)
		if err != nil {
			log.Error(ctx, "cannot get from cache %s: %v", k, err)
		}
		if !b {
			return sdk.WithStack(sdk.ErrNotFound)
		}
		return service.WriteJSON(w, ope, http.StatusOK)
	}
}

// postWorkflowAsCodeHandler Update an as code workflow
// @title Make the workflow as code
// @title Update an as code workflow
func (api *API) postWorkflowAsCodeHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		workflowName := vars["permWorkflowName"]
		migrate := FormBool(r, "migrate")
		branch := FormString(r, "branch")
		message := FormString(r, "message")

		u := getAPIConsumer(ctx)
		p, err := project.Load(api.mustDB(), api.Cache, key,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithIntegrations,
			project.LoadOptions.WithClearKeys,
		)
		if err != nil {
			return err
		}

		wfDB, err := workflow.Load(ctx, api.mustDB(), api.Cache, *p, workflowName, workflow.LoadOptions{
			DeepPipeline:          migrate,
			WithAsCodeUpdateEvent: migrate,
		})
		if err != nil {
			return err
		}

		if wfDB.WorkflowData.Node.Context.ApplicationID == 0 {
			return sdk.WrapError(sdk.ErrApplicationNotFound, "root node does not have application context")
		}
		app := wfDB.Applications[wfDB.WorkflowData.Node.Context.ApplicationID]
		if app.VCSServer == "" || app.RepositoryFullname == "" {
			return sdk.WithStack(sdk.ErrRepoNotFound)
		}

		// MIGRATION TO AS CODE
		if migrate {
			return api.migrateWorkflowAsCode(ctx, w, p, wfDB, &app, branch, message)
		}

		// UPDATE EXISTING AS CODE WORKFLOW
		if wfDB.FromRepository == "" {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		// Get workflow from body
		var wk sdk.Workflow
		if err := service.UnmarshalBody(r, &wk); err != nil {
			return err
		}

		ope, err := workflow.UpdateWorkflowAsCode(ctx, api.Cache, api.mustDB(), *p, wk, app, branch, message, u.AuthentifiedUser)
		if err != nil {
			return err
		}

		sdk.GoRoutine(context.Background(), fmt.Sprintf("UpdateAsCodeResult-%s", ope.UUID), func(ctx context.Context) {
			ed := ascode.EntityData{
				Operation: ope,
				Name:      wk.Name,
				ID:        wk.ID,
				Type:      ascode.AsCodeWorkflow,
				FromRepo:  wk.FromRepository,
			}
			asCodeEvent := ascode.UpdateAsCodeResult(ctx, api.mustDB(), api.Cache, *p, &app, ed, u)
			if asCodeEvent != nil {
				event.PublishAsCodeEvent(ctx, p.Key, *asCodeEvent, u)
			}
			event.PublishWorkflowUpdate(ctx, p.Key, wk, wk, u)
		}, api.PanicDump())

		return service.WriteJSON(w, ope, http.StatusOK)
	}
}

func (api *API) migrateWorkflowAsCode(ctx context.Context, w http.ResponseWriter, proj *sdk.Project, wf *sdk.Workflow, app *sdk.Application, branch, message string) error {
	u := getAPIConsumer(ctx)

	// Sync as code event
	if len(wf.AsCodeEvent) > 0 {
		eventsLeft, _, err := sync.SyncAsCodeEvent(ctx, api.mustDB(), api.Cache, *proj, *app, getAPIConsumer(ctx).AuthentifiedUser)
		if err != nil {
			return err
		}
		wf.AsCodeEvent = eventsLeft
	}

	if wf.FromRepository != "" || (wf.FromRepository == "" && len(wf.AsCodeEvent) > 0) {
		return sdk.WithStack(sdk.ErrWorkflowAlreadyAsCode)
	}

	// Check if there is a repository web hook
	found := false
	for _, h := range wf.WorkflowData.GetHooks() {
		if h.HookModelName == sdk.RepositoryWebHookModelName {
			found = true
			break
		}
	}
	if !found {
		h := sdk.NodeHook{
			Config:        sdk.RepositoryWebHookModel.DefaultConfig.Clone(),
			HookModelName: sdk.RepositoryWebHookModel.Name,
		}
		wf.WorkflowData.Node.Hooks = append(wf.WorkflowData.Node.Hooks, h)

		if err := workflow.Update(ctx, api.mustDB(), api.Cache, *proj, wf, workflow.UpdateOptions{}); err != nil {
			return err
		}
	}

	// Export workflow + push + create pull request
	ope, err := workflow.MigrateAsCode(ctx, api.mustDB(), api.Cache, *proj, wf, *app, u, project.EncryptWithBuiltinKey, branch, message)
	if err != nil {
		return sdk.WrapError(err, "unable to migrate workflow as code")
	}

	sdk.GoRoutine(context.Background(), fmt.Sprintf("MigrateWorkflowAsCodeResult-%s", ope.UUID), func(ctx context.Context) {
		ed := ascode.EntityData{
			FromRepo:  ope.URL,
			Type:      ascode.AsCodeWorkflow,
			ID:        wf.ID,
			Name:      wf.Name,
			Operation: ope,
		}
		asCodeEvent := ascode.UpdateAsCodeResult(ctx, api.mustDB(), api.Cache, *proj, app, ed, u)
		if asCodeEvent != nil {
			event.PublishAsCodeEvent(ctx, proj.Key, *asCodeEvent, u)
		}
	}, api.PanicDump())

	return service.WriteJSON(w, ope, http.StatusOK)
}
