package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

// getWorkflowsHandler returns ID and name of workflows for a given project/user
func (api *API) getWorkflowsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		ws, err := workflow.LoadAll(api.mustDB(), key)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, ws, http.StatusOK)
	}
}

// getWorkflowHandler returns a full workflow
func (api *API) getWorkflowHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		withUsage := FormBool(r, "withUsage")
		withAudits := FormBool(r, "withAudits")
		withLabels := FormBool(r, "withLabels")
		withDeepPipelines := FormBool(r, "withDeepPipelines")

		proj, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.WithPlatforms)
		if err != nil {
			return sdk.WrapError(err, "getWorkflowHandler> unable to load projet")
		}

		opts := workflow.LoadOptions{WithFavorites: true, DeepPipeline: withDeepPipelines, WithIcon: true, WithLabels: withLabels}
		w1, err := workflow.Load(ctx, api.mustDB(), api.Cache, proj, name, getUser(ctx), opts)
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

		if withAudits {
			audits, errA := workflow.LoadAudits(api.mustDB(), w1.ID)
			if errA != nil {
				return sdk.WrapError(errA, "getWorkflowHandler> Cannot load audits for workflow %s", name)
			}
			w1.Audits = audits
		}

		w1.Permission = permission.WorkflowPermission(key, w1.Name, getUser(ctx))

		//We filter project and workflow configurtaion key, because they are always set on insertHooks
		w1.FilterHooksConfig(sdk.HookConfigProject, sdk.HookConfigWorkflow)

		return service.WriteJSON(w, w1, http.StatusOK)
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

// postWorkflowRollbackHandler rollback to a specific audit id
func (api *API) postWorkflowRollbackHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		workflowName := vars["permWorkflowName"]
		auditID, errConv := strconv.ParseInt(vars["auditID"], 10, 64)
		if errConv != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "postWorkflowRollbackHandler> cannot convert auditID to int")
		}
		db := api.mustDB()
		u := getUser(ctx)

		proj, errP := project.Load(db, api.Cache, key, u,
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithApplications,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithPlatforms,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
		)
		if errP != nil {
			return sdk.WrapError(errP, "postWorkflowRollbackHandler> cannot load project %s", key)
		}

		wf, errW := workflow.Load(ctx, db, api.Cache, proj, workflowName, u, workflow.LoadOptions{WithoutNode: true})
		if errW != nil {
			return sdk.WrapError(errW, "postWorkflowRollbackHandler> cannot load workflow %s/%s", key, workflowName)
		}

		audit, errA := workflow.LoadAudit(db, auditID)
		if errA != nil {
			return sdk.WrapError(errA, "postWorkflowRollbackHandler> cannot load workflow audit %s/%s", key, workflowName)
		}

		var exportWf exportentities.Workflow
		if err := yaml.Unmarshal([]byte(audit.DataBefore), &exportWf); err != nil {
			return sdk.WrapError(err, "postWorkflowRollbackHandler> Cannot unmarshall data before")
		}

		tx, errTx := db.Begin()
		if errTx != nil {
			return sdk.WrapError(errTx, "postWorkflowRollbackHandler> Cannot begin transaction")
		}
		defer func() {
			_ = tx.Rollback()
		}()

		newWf, _, errP := workflow.ParseAndImport(ctx, tx, api.Cache, proj, &exportWf, u, workflow.ImportOptions{Force: true, WorkflowName: workflowName})
		if errP != nil {
			return sdk.WrapError(errP, "postWorkflowRollbackHandler> cannot parse and import previous workflow")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "postWorkflowRollbackHandler> cannot commit transaction")
		}

		event.PublishWorkflowUpdate(key, *wf, *newWf, getUser(ctx))

		return service.WriteJSON(w, *newWf, http.StatusOK)
	}
}

// postWorkflowLabelHandler handler to link a label to a workflow
func (api *API) postWorkflowLabelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		workflowName := vars["permWorkflowName"]
		db := api.mustDB()
		u := getUser(ctx)

		var label sdk.Label
		if err := UnmarshalBody(r, &label); err != nil {
			return sdk.WrapError(err, "Cannot read body")
		}

		proj, errP := project.Load(db, api.Cache, key, u)
		if errP != nil {
			return sdk.WrapError(errP, "postWorkflowLabelHandler> cannot load project %s", key)
		}
		label.ProjectID = proj.ID

		tx, errTx := db.Begin()
		if errTx != nil {
			return sdk.WrapError(errTx, "postWorkflowLabelHandler> Cannot create new transaction")
		}
		defer tx.Rollback() //nolint

		if label.ID == 0 {
			if label.Name == "" {
				return service.WriteJSON(w, "Label ID or label name should not be empty", http.StatusBadRequest)
			}

			lbl, errL := project.LabelByName(db, proj.ID, label.Name)
			if errL != nil {
				if errL != sql.ErrNoRows {
					return sdk.WrapError(errL, "postWorkflowLabelHandler> cannot load label by name")
				}
				// If label doesn't exist create him
				if err := project.InsertLabel(tx, &label); err != nil {
					return sdk.WrapError(err, "postWorkflowLabelHandler> Cannot create new label")
				}
			} else {
				label.ID = lbl.ID
			}
		}

		wf, errW := workflow.Load(ctx, db, api.Cache, proj, workflowName, u, workflow.LoadOptions{WithoutNode: true, WithLabels: true})
		if errW != nil {
			return sdk.WrapError(errW, "postWorkflowLabelHandler> cannot load workflow %s/%s", key, workflowName)
		}

		if err := workflow.LabelWorkflow(tx, label.ID, wf.ID); err != nil {
			return sdk.WrapError(err, "postWorkflowLabelHandler> cannot link label %d to workflow %s", label.ID, wf.Name)
		}
		newWf := *wf
		label.WorkflowID = wf.ID
		newWf.Labels = append(newWf.Labels, label)

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "postWorkflowLabelHandler> Cannot commit transaction")
		}

		return service.WriteJSON(w, label, http.StatusOK)
	}
}

// deleteWorkflowLabelHandler handler to unlink a label to a workflow
func (api *API) deleteWorkflowLabelHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		workflowName := vars["permWorkflowName"]
		labelID, errV := requestVarInt(r, "labelID")
		if errV != nil {
			return sdk.WrapError(errV, "deleteWorkflowLabelHandler> Cannot convert to int labelID")
		}

		db := api.mustDB()
		u := getUser(ctx)

		proj, errP := project.Load(db, api.Cache, key, u)
		if errP != nil {
			return sdk.WrapError(errP, "deleteWorkflowLabelHandler> cannot load project %s", key)
		}

		wf, errW := workflow.Load(ctx, db, api.Cache, proj, workflowName, u, workflow.LoadOptions{WithoutNode: true})
		if errW != nil {
			return sdk.WrapError(errW, "deleteWorkflowLabelHandler> cannot load workflow %s/%s", key, workflowName)
		}

		if err := workflow.UnLabelWorkflow(db, labelID, wf.ID); err != nil {
			return sdk.WrapError(err, "deleteWorkflowLabelHandler> cannot unlink label %d to workflow %s", labelID, wf.Name)
		}

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

// postWorkflowHandler creates a new workflow
func (api *API) postWorkflowHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		p, errP := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.WithApplicationWithDeploymentStrategies, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups, project.LoadOptions.WithPlatforms)
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

		if wf.Root != nil && wf.Root.Context != nil && (wf.Root.Context.Application != nil || wf.Root.Context.ApplicationID != 0) {
			var err error
			if wf.Root.Context.DefaultPayload, err = workflow.DefaultPayload(ctx, tx, api.Cache, p, getUser(ctx), &wf); err != nil {
				log.Warning("putWorkflowHandler> Cannot set default payload : %v", err)
			}
		}

		if err := workflow.Insert(tx, api.Cache, &wf, p, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "Cannot insert workflow")
		}

		if errHr := workflow.HookRegistration(ctx, tx, api.Cache, nil, wf, p); errHr != nil {
			return sdk.WrapError(errHr, "postWorkflowHandler>Hook registration failed")
		}

		// Add group
		for _, gp := range p.ProjectGroups {
			if gp.Permission >= permission.PermissionReadExecute {
				if err := workflow.AddGroup(tx, &wf, gp); err != nil {
					return sdk.WrapError(err, "Cannot add group %s", gp.Group.Name)
				}
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		wf1, errl := workflow.LoadByID(api.mustDB(), api.Cache, p, wf.ID, getUser(ctx), workflow.LoadOptions{})
		if errl != nil {
			return sdk.WrapError(errl, "Cannot load workflow")
		}

		//We filter project and workflow configurtaion key, because they are always set on insertHooks
		wf1.FilterHooksConfig(sdk.HookConfigProject, sdk.HookConfigWorkflow)

		return service.WriteJSON(w, wf1, http.StatusCreated)
	}
}

// putWorkflowHandler updates a workflow
func (api *API) putWorkflowHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		p, errP := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.WithApplicationWithDeploymentStrategies, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithPlatforms)
		if errP != nil {
			return sdk.WrapError(errP, "putWorkflowHandler> Cannot load Project %s", key)
		}

		oldW, errW := workflow.Load(ctx, api.mustDB(), api.Cache, p, name, getUser(ctx), workflow.LoadOptions{WithIcon: true})
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

		if wf.Root != nil && wf.Root.Context != nil && (wf.Root.Context.Application != nil || wf.Root.Context.ApplicationID != 0) {
			var err error
			if wf.Root.Context.DefaultPayload, err = workflow.DefaultPayload(ctx, tx, api.Cache, p, getUser(ctx), &wf); err != nil {
				log.Warning("putWorkflowHandler> Cannot set default payload : %v", err)
			}
		}

		if err := workflow.Update(tx, api.Cache, &wf, oldW, p, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "putWorkflowHandler> Cannot update workflow")
		}

		// HookRegistration after workflow.Update.  It needs hooks to be created on DB
		if errHr := workflow.HookRegistration(ctx, tx, api.Cache, oldW, wf, p); errHr != nil {
			return sdk.WrapError(errHr, "putWorkflowHandler> HookRegistration")
		}

		if defaultTags, ok := wf.Metadata["default_tags"]; wf.Root.IsLinkedToRepo() && (!ok || defaultTags == "") {
			if wf.Metadata == nil {
				wf.Metadata = sdk.Metadata{}
			}
			wf.Metadata["default_tags"] = "git.branch,git.author"
			if err := workflow.UpdateMetadata(tx, wf.ID, wf.Metadata); err != nil {
				return sdk.WrapError(err, "putWorkflowHandler> cannot update metadata")
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "putWorkflowHandler> Cannot commit transaction")
		}

		wf1, errl := workflow.LoadByID(api.mustDB(), api.Cache, p, wf.ID, getUser(ctx), workflow.LoadOptions{})
		if errl != nil {
			return sdk.WrapError(errl, "putWorkflowHandler> Cannot load workflow")
		}

		usage, errU := loadWorkflowUsage(api.mustDB(), wf1.ID)
		if errU != nil {
			return sdk.WrapError(errU, "Cannot load usage")
		}
		wf1.Usage = &usage

		//We filter project and workflow configuration key, because they are always set on insertHooks
		wf1.FilterHooksConfig(sdk.HookConfigProject, sdk.HookConfigWorkflow)

		return service.WriteJSON(w, wf1, http.StatusOK)
	}
}

// putWorkflowHandler deletes a workflow
func (api *API) deleteWorkflowHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		p, errP := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.WithPlatforms)
		if errP != nil {
			return sdk.WrapError(errP, "Cannot load Project %s", key)
		}

		oldW, errW := workflow.Load(ctx, api.mustDB(), api.Cache, p, name, getUser(ctx), workflow.LoadOptions{})
		if errW != nil {
			return sdk.WrapError(errW, "Cannot load Workflow %s", key)
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "Cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err := workflow.MarkAsDelete(tx, oldW); err != nil {
			return sdk.WrapError(err, "Cannot delete workflow")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(errT, "Cannot commit transaction")
		}
		event.PublishWorkflowDelete(key, *oldW, getUser(ctx))

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) getWorkflowHookHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		uuid := vars["uuid"]

		proj, errP := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.WithPlatforms)
		if errP != nil {
			return sdk.WrapError(errP, "Cannot load Project %s", key)
		}

		wf, errW := workflow.Load(ctx, api.mustDB(), api.Cache, proj, name, getUser(ctx), workflow.LoadOptions{})
		if errW != nil {
			return sdk.WrapError(errW, "getWorkflowHookHandler> Cannot load Workflow %s/%s", key, name)
		}

		whooks := wf.GetHooks()
		_, has := whooks[uuid]
		if !has {
			return sdk.WrapError(sdk.ErrNotFound, "getWorkflowHookHandler> Cannot load Workflow %s/%s hook %s", key, name, uuid)
		}

		//Push the hook to hooks µService
		//Load service "hooks"
		srvs, errS := services.FindByType(api.mustDB(), services.TypeHooks)
		if errS != nil {
			return sdk.WrapError(errS, "getWorkflowHookHandler> Unable to load hooks services")
		}

		path := fmt.Sprintf("/task/%s/execution", uuid)
		task := sdk.Task{}
		if _, err := services.DoJSONRequest(ctx, srvs, "GET", path, nil, &task); err != nil {
			return sdk.WrapError(err, "getWorkflowHookHandler> Unable to get hook %s task and executions", uuid)
		}

		return service.WriteJSON(w, task, http.StatusOK)
	}
}
