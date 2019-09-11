package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/cache"
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
		k := cache.Key(workflow.CacheOperationKey, uuid)
		b, err := api.Cache.Get(k, &ope)
		if err != nil {
			log.Error("cannot get from cache %s: %v", k, err)
		}
		if !b {
			return sdk.ErrNotFound
		}
		return service.WriteJSON(w, ope, http.StatusOK)
	}
}

func (api *API) postResyncPRWorkflowAsCodeHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		workflowName := vars["permWorkflowName"]

		u := getAPIConsumer(ctx)
		proj, errP := project.Load(api.mustDB(), api.Cache, key,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithIntegrations,
			project.LoadOptions.WithClearKeys)
		if errP != nil {
			return sdk.WrapError(errP, "unable to load project")
		}
		wf, errW := workflow.Load(ctx, api.mustDB(), api.Cache, proj, workflowName, workflow.LoadOptions{
			DeepPipeline:          false,
			WithAsCodeUpdateEvent: true,
		})
		if errW != nil {
			return sdk.WrapError(errW, "unable to load workflow")
		}
		if err := workflow.SyncAsCodeEvent(ctx, api.mustDB(), api.Cache, proj, wf, u); err != nil {
			return err
		}
		return nil
	}
}

// postWorkflowAsCodeHandler Update an as code workflow
// @title Make the workflow as code	// @title Update an as code workflow
func (api *API) postWorkflowAsCodeHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		workflowName := vars["permWorkflowName"]
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

		wfDB, err := workflow.Load(ctx, api.mustDB(), api.Cache, p, workflowName, workflow.LoadOptions{})
		if err != nil {
			return err
		}
		if wfDB.FromRepository == "" {
			return sdk.ErrForbidden
		}

		var wk sdk.Workflow
		if err := service.UnmarshalBody(r, &wk); err != nil {
			return err
		}

		ope, err := workflow.UpdateWorkflowAsCode(ctx, api.Cache, api.mustDB(), p, wk, branch, message, u.AuthentifiedUser)
		if err != nil {
			return err
		}

		sdk.GoRoutine(context.Background(), fmt.Sprintf("UpdateWorkflowAsCodeResult-%s", ope.UUID), func(ctx context.Context) {
			workflow.UpdateWorkflowAsCodeResult(ctx, api.mustDB(), api.Cache, p, ope, &wk, u)
		}, api.PanicDump())

		return service.WriteJSON(w, ope, http.StatusOK)
	}
}

// postMigrateWorkflowAsCodeHandler Make the workflow as code
// @title Make the workflow as code
func (api *API) postMigrateWorkflowAsCodeHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		workflowName := vars["permWorkflowName"]

		u := getAPIConsumer(ctx)

		proj, errP := project.Load(api.mustDB(), api.Cache, key,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithIntegrations,
			project.LoadOptions.WithClearKeys)
		if errP != nil {
			return sdk.WrapError(errP, "unable to load project")
		}
		wf, errW := workflow.Load(ctx, api.mustDB(), api.Cache, proj, workflowName, workflow.LoadOptions{
			DeepPipeline:          true,
			WithAsCodeUpdateEvent: true,
		})
		if errW != nil {
			return sdk.WrapError(errW, "unable to load workflow")
		}

		// Sync as code event
		if len(wf.AsCodeEvent) > 0 {
			tx, err := api.mustDB().Begin()
			if err != nil {
				return sdk.WrapError(err, "unable to start transaction")
			}
			if err := workflow.SyncAsCodeEvent(ctx, tx, api.Cache, proj, wf, u); err != nil {
				tx.Rollback() // nolint
				return err
			}
			if err := tx.Commit(); err != nil {
				return sdk.WrapError(err, "unable to commit transaction")
			}
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

			oldW, errOld := workflow.LoadByID(ctx, api.mustDB(), api.Cache, proj, wf.ID, workflow.LoadOptions{})
			if errOld != nil {
				return errOld
			}

			if err := workflow.Update(ctx, api.mustDB(), api.Cache, wf, proj, workflow.UpdateOptions{OldWorkflow: oldW}); err != nil {
				return err
			}
		}

		// Export workflow + push + create pull request
		ope, err := workflow.MigrateAsCode(ctx, api.mustDB(), api.Cache, proj, wf, u, project.EncryptWithBuiltinKey)
		if err != nil {
			return sdk.WrapError(err, "unable to migrate workflow as code")
		}

		sdk.GoRoutine(context.Background(), fmt.Sprintf("MigrateWorkflowAsCodeResult-%s", ope.UUID), func(ctx context.Context) {
			workflow.UpdateWorkflowAsCodeResult(ctx, api.mustDB(), api.Cache, proj, ope, wf, u)
		}, api.PanicDump())

		return service.WriteJSON(w, ope, http.StatusOK)
	}
}
