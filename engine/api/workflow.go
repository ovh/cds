package api

import (
	"context"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

// getWorkflowsHandler returns ID and name of workflows for a given project/user
func (api *API) getWorkflowsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		ws, err := workflow.LoadAll(api.mustDB(), key)
		if err != nil {
			return err
		}

		return WriteJSON(w, r, ws, http.StatusOK)
	}
}

// getWorkflowHandler returns a full workflow
func (api *API) getWorkflowHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		withUsage := FormBool(r, "withUsage")

		w1, err := workflow.Load(api.mustDB(), api.Cache, key, name, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "getWorkflowHandler> Cannot load workflow %s", name)
		}

		if withUsage {
			usage, errU := loadWorkflowUsage(api.mustDB(), w1.ID)
			if errU != nil {
				return sdk.WrapError(errU, "getWorkflowHandler> Cannot load usage for workflow %s", name)
			}
			w1.Usage = &usage
		}

		w1.Permission = permission.WorkflowPermission(w1.ID, getUser(ctx))

		//We filter project and workflow configurtaion key, because they are always set on insertHooks
		w1.FilterHooksConfig("project", "workflow")

		return WriteJSON(w, r, w1, http.StatusOK)
	}
}

func loadWorkflowUsage(db gorp.SqlExecutor, workflowID int64) (sdk.Usage, error) {
	usage := sdk.Usage{}
	pips, errP := pipeline.LoadByWorkflowID(db, workflowID)
	if errP != nil {
		return usage, sdk.WrapError(errP, "loadWorkflowUsage> Cannot load pipelines linked to a workflow id %d", workflowID)
	}
	usage.Pipelines = pips

	envs, errE := environment.LoadByWorkflowID(db, workflowID)
	if errE != nil {
		return usage, sdk.WrapError(errE, "loadWorkflowUsage> Cannot load environments linked to a workflow id %d", workflowID)
	}
	usage.Environments = envs

	apps, errA := application.LoadByWorkflowID(db, workflowID)
	if errA != nil {
		return usage, sdk.WrapError(errA, "loadWorkflowUsage> Cannot load applications linked to a workflow id %d", workflowID)
	}
	usage.Applications = apps

	return usage, nil
}

// postWorkflowHandler create a new workflow
func (api *API) postWorkflowHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		p, errP := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)
		if errP != nil {
			return sdk.WrapError(errP, "Cannot load Project %s", key)
		}
		var wf sdk.Workflow
		if err := UnmarshalBody(r, &wf); err != nil {
			return sdk.WrapError(err, "Cannot read body")
		}
		wf.ProjectID = p.ID
		wf.ProjectKey = key

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "Cannot start transaction")
		}
		defer tx.Rollback()

		if err := workflow.Insert(tx, api.Cache, &wf, p, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "Cannot insert workflow")
		}

		// Add group
		for _, gp := range p.ProjectGroups {
			if gp.Permission == permission.PermissionReadWriteExecute {
				if err := workflow.AddGroup(tx, &wf, gp); err != nil {
					return sdk.WrapError(err, "Cannot add group %s", gp.Group.Name)
				}
			}
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectWorkflowLastModificationType); err != nil {
			return sdk.WrapError(err, "Cannot update project last modified date")
		}

		if err := workflow.HookRegistration(tx, api.Cache, nil, wf, p); err != nil {
			return sdk.WrapError(err, "postWorkflowHandler")
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectWorkflowLastModificationType); err != nil {
			return sdk.WrapError(err, "postWorkflowHandler> Cannot update project workflows last modified")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		wf1, errl := workflow.LoadByID(api.mustDB(), api.Cache, wf.ID, getUser(ctx))
		if errl != nil {
			return sdk.WrapError(errl, "Cannot load workflow")
		}

		//We filter project and workflow configurtaion key, because they are always set on insertHooks
		wf1.FilterHooksConfig("project", "workflow")

		return WriteJSON(w, r, wf1, http.StatusCreated)
	}
}

// putWorkflowHandler updates a workflow
func (api *API) putWorkflowHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		p, errP := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments)
		if errP != nil {
			return sdk.WrapError(errP, "putWorkflowHandler> Cannot load Project %s", key)
		}

		oldW, errW := workflow.Load(api.mustDB(), api.Cache, key, name, getUser(ctx))
		if errW != nil {
			return sdk.WrapError(errW, "putWorkflowHandler> Cannot load Workflow %s", key)
		}

		var wf sdk.Workflow
		if err := UnmarshalBody(r, &wf); err != nil {
			return sdk.WrapError(err, "Cannot read body")
		}
		wf.ID = oldW.ID
		wf.RootID = oldW.RootID
		wf.Root.ID = oldW.RootID
		wf.ProjectID = p.ID
		wf.ProjectKey = key

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "putWorkflowHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := workflow.Update(tx, api.Cache, &wf, oldW, p, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "putWorkflowHandler> Cannot update workflow")
		}

		if err := workflow.UpdateLastModifiedDate(tx, api.Cache, getUser(ctx), p.Key, oldW); err != nil {
			return sdk.WrapError(err, "putWorkflowHandler> Cannot update last modified date for workflow")
		}

		if oldW.Name != wf.Name {
			if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectWorkflowLastModificationType); err != nil {
				return sdk.WrapError(err, "putWorkflowHandler> Cannot update project last modified date")
			}
		}

		if err := workflow.HookRegistration(tx, api.Cache, oldW, wf, p); err != nil {
			return sdk.WrapError(err, "putWorkflowHandler")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "putWorkflowHandler> Cannot commit transaction")
		}

		wf1, errl := workflow.LoadByID(api.mustDB(), api.Cache, wf.ID, getUser(ctx))
		if errl != nil {
			return sdk.WrapError(errl, "putWorkflowHandler> Cannot load workflow")
		}

		usage, errU := loadWorkflowUsage(api.mustDB(), wf1.ID)
		if errU != nil {
			return sdk.WrapError(errU, "Cannot load usage")
		}
		wf1.Usage = &usage

		//We filter project and workflow configurtaion key, because they are always set on insertHooks
		wf1.FilterHooksConfig("project", "workflow")

		return WriteJSON(w, r, wf1, http.StatusOK)
	}
}

// putWorkflowHandler deletes a workflow
func (api *API) deleteWorkflowHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		p, errP := project.Load(api.mustDB(), api.Cache, key, getUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "Cannot load Project %s", key)
		}

		oldW, errW := workflow.Load(api.mustDB(), api.Cache, key, name, getUser(ctx))
		if errW != nil {
			return sdk.WrapError(errW, "Cannot load Workflow %s", key)
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "Cannot start transaction")
		}
		defer tx.Rollback()

		if err := workflow.Delete(tx, api.Cache, p, oldW, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "Cannot delete workflow")
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectWorkflowLastModificationType); err != nil {
			return sdk.WrapError(err, "Cannot update project last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(errT, "Cannot commit transaction")
		}
		return WriteJSON(w, r, nil, http.StatusOK)
	}
}
