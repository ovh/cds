package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
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
		key := vars["permProjectKey"]
		name := vars["workflowName"]

		w1, err := workflow.Load(api.mustDB(), api.Cache, key, name, getUser(ctx))
		if err != nil {
			return err
		}
		//We filter project and workflow configurtaion key, because they are always set on insertHooks
		w1.FilterHooksConfig("project", "workflow")
		return WriteJSON(w, r, w1, http.StatusOK)
	}
}

// postWorkflowHandler create a new workflow
func (api *API) postWorkflowHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		p, errP := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments)
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

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p); err != nil {
			return sdk.WrapError(err, "Cannot update project last modified date")
		}

		//Push the hook to hooks µService
		dao := services.NewRepository(api.mustDB, api.Cache)
		//Load service "hooks"
		srvs, err := dao.FindByType("hooks")
		if err != nil {
			return sdk.WrapError(err, "putWorkflowHandler> Unable to get services dao")
		}

		//Perform the request on one off the hooks service
		hooks := wf.GetHooks()
		if len(hooks) > 0 {
			if len(srvs) < 1 {
				return sdk.WrapError(fmt.Errorf("postWorkflowHandler> No hooks service available, please try again"), "Unable to get services dao")
			}
			var errHooks error
			for _, s := range srvs {
				code, errBulk := services.DoJSONRequest(&s, http.MethodPost, "/task/bulk", hooks, nil)
				errHooks = errBulk
				if errBulk == nil {
					log.Debug("postWorkflowHandler> %d hooks created for workflow %s/%s (HTTP status code %d)", len(hooks), wf.ProjectKey, wf.Name, code)
					break
				}
			}
			if errHooks != nil {
				return sdk.WrapError(errHooks, "postWorkflowHandler> Unable to create hooks")
			}
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
		key := vars["permProjectKey"]
		name := vars["workflowName"]

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

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p); err != nil {
			return sdk.WrapError(err, "putWorkflowHandler> Cannot update project last modified date")
		}

		hooks := wf.GetHooks()
		if len(hooks) > 0 {

			//Push the hook to hooks µService
			dao := services.NewRepository(api.mustDB, api.Cache)
			//Load service "hooks"
			srvs, err := dao.FindByType("hooks")
			if err != nil {
				return sdk.WrapError(err, "putWorkflowHandler> Unable to get services dao")
			}

			if wf.Name != name {
				// update hook
				for i := range hooks {
					h := hooks[i]
					h.Config["workflow"] = wf.Name
					hooks[i] = h
				}
			}

			//Perform the request on one off the hooks service
			if len(srvs) < 1 {
				return sdk.WrapError(fmt.Errorf("putWorkflowHandler> No hooks service available, please try again"), "Unable to get services dao")
			}
			var errHooks error
			for _, s := range srvs {
				code, errBulk := services.DoJSONRequest(&s, http.MethodPost, "/task/bulk", hooks, nil)
				errHooks = errBulk
				if errBulk == nil {
					log.Debug("putWorkflowHandler> %d hooks created for workflow %s/%s (HTTP status code %d)", len(hooks), wf.ProjectKey, wf.Name, code)
					break
				}
			}
			if errHooks != nil {
				return sdk.WrapError(errHooks, "putWorkflowHandler> Unable to create hooks")
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "putWorkflowHandler> Cannot commit transaction")
		}

		wf1, errl := workflow.LoadByID(api.mustDB(), api.Cache, wf.ID, getUser(ctx))
		if errl != nil {
			return sdk.WrapError(errl, "putWorkflowHandler> Cannot load workflow")
		}

		//We filter project and workflow configurtaion key, because they are always set on insertHooks
		wf1.FilterHooksConfig("project", "workflow")

		return WriteJSON(w, r, wf1, http.StatusOK)
	}
}

// putWorkflowHandler deletes a workflow
func (api *API) deleteWorkflowHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["permProjectKey"]
		name := vars["workflowName"]

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

		if err := workflow.Delete(tx, oldW, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "Cannot delete workflow")
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p); err != nil {
			return sdk.WrapError(err, "Cannot update project last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(errT, "Cannot commit transaction")
		}
		return WriteJSON(w, r, nil, http.StatusOK)
	}
}
