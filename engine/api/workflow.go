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
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

// getWorkflowsHandler returns ID and name of workflows for a given project/user
func (api *API) getWorkflowsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]

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
		withTemplate := FormBool(r, "withTemplate")
		withAsCodeEvents := FormBool(r, "withAsCodeEvents")

		proj, err := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx), project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WrapError(err, "unable to load projet")
		}

		opts := workflow.LoadOptions{
			WithFavorites:         true,
			DeepPipeline:          withDeepPipelines,
			WithIcon:              true,
			WithLabels:            withLabels,
			WithAsCodeUpdateEvent: withAsCodeEvents,
		}
		w1, err := workflow.Load(ctx, api.mustDB(), api.Cache, proj, name, deprecatedGetUser(ctx), opts)
		if err != nil {
			return sdk.WrapError(err, "Cannot load workflow %s", name)
		}

		if withUsage {
			usage, errU := loadWorkflowUsage(api.mustDB(), w1.ID)
			if errU != nil {
				return sdk.WrapError(errU, "Cannot load usage for workflow %s", name)
			}
			w1.Usage = &usage
		}

		if withAudits {
			audits, errA := workflow.LoadAudits(api.mustDB(), w1.ID)
			if errA != nil {
				return sdk.WrapError(errA, "Cannot load audits for workflow %s", name)
			}
			w1.Audits = audits
		}

		if withTemplate {
			if err := workflowtemplate.AggregateTemplateInstanceOnWorkflow(api.mustDB(), w1); err != nil {
				return err
			}
			if w1.TemplateInstance != nil {
				if err := workflowtemplate.AggregateTemplateOnInstance(api.mustDB(), w1.TemplateInstance); err != nil {
					return err
				}
				if w1.TemplateInstance.Template != nil {
					if err := group.AggregateOnWorkflowTemplate(api.mustDB(), w1.TemplateInstance.Template); err != nil {
						return err
					}
					w1.FromTemplate = fmt.Sprintf("%s/%s", w1.TemplateInstance.Template.Group.Name, w1.TemplateInstance.Template.Slug)
					w1.TemplateUpToDate = w1.TemplateInstance.Template.Version == w1.TemplateInstance.WorkflowTemplateVersion
				}
			}
		}

		w1.Permission = permission.WorkflowPermission(key, w1.Name, deprecatedGetUser(ctx))

		// FIXME Sync hooks on new model. To delete when hooks will be compute on new model
		hooks := w1.GetHooks()
		w1.WorkflowData.Node.Hooks = make([]sdk.NodeHook, 0, len(hooks))
		for _, h := range hooks {
			w1.WorkflowData.Node.Hooks = append(w1.WorkflowData.Node.Hooks, sdk.NodeHook{
				Ref:           h.Ref,
				HookModelID:   h.WorkflowHookModelID,
				Config:        h.Config,
				UUID:          h.UUID,
				HookModelName: h.WorkflowHookModel.Name,
				NodeID:        w1.WorkflowData.Node.ID,
			})
		}

		//We filter project and workflow configurtaion key, because they are always set on insertHooks
		w1.FilterHooksConfig(sdk.HookConfigProject, sdk.HookConfigWorkflow)
		return service.WriteJSON(w, w1, http.StatusOK)
	}
}

func loadWorkflowUsage(db gorp.SqlExecutor, workflowID int64) (sdk.Usage, error) {
	usage := sdk.Usage{}
	pips, errP := pipeline.LoadByWorkflowID(db, workflowID)
	if errP != nil {
		return usage, sdk.WrapError(errP, "Cannot load pipelines linked to a workflow id %d", workflowID)
	}
	usage.Pipelines = pips

	envs, errE := environment.LoadByWorkflowID(db, workflowID)
	if errE != nil {
		return usage, sdk.WrapError(errE, "Cannot load environments linked to a workflow id %d", workflowID)
	}
	usage.Environments = envs

	apps, errA := application.LoadByWorkflowID(db, workflowID)
	if errA != nil {
		return usage, sdk.WrapError(errA, "Cannot load applications linked to a workflow id %d", workflowID)
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
			return sdk.WrapError(sdk.ErrWrongRequest, "Cannot convert auditID to int")
		}
		db := api.mustDB()
		u := deprecatedGetUser(ctx)

		proj, errP := project.Load(db, api.Cache, key, u,
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithApplications,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithIntegrations,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
		)
		if errP != nil {
			return sdk.WrapError(errP, "Cannot load project %s", key)
		}

		wf, errW := workflow.Load(ctx, db, api.Cache, proj, workflowName, u, workflow.LoadOptions{WithIcon: true})
		if errW != nil {
			return sdk.WrapError(errW, "Cannot load workflow %s/%s", key, workflowName)
		}

		audit, errA := workflow.LoadAudit(db, auditID, wf.ID)
		if errA != nil {
			return sdk.WrapError(errA, "Cannot load workflow audit %s/%s", key, workflowName)
		}

		var exportWf exportentities.Workflow
		if err := yaml.Unmarshal([]byte(audit.DataBefore), &exportWf); err != nil {
			return sdk.WrapError(err, "Cannot unmarshal data before")
		}

		tx, errTx := db.Begin()
		if errTx != nil {
			return sdk.WrapError(errTx, "Cannot begin transaction")
		}
		defer func() {
			_ = tx.Rollback()
		}()

		newWf, _, errP := workflow.ParseAndImport(ctx, tx, api.Cache, proj, wf, &exportWf, u, workflow.ImportOptions{Force: true, WorkflowName: workflowName})
		if errP != nil {
			return sdk.WrapError(errP, "Cannot parse and import previous workflow")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishWorkflowUpdate(key, *wf, *newWf, deprecatedGetUser(ctx))

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
		u := deprecatedGetUser(ctx)

		var label sdk.Label
		if err := service.UnmarshalBody(r, &label); err != nil {
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
				if sdk.Cause(errL) != sql.ErrNoRows {
					return sdk.WrapError(errL, "postWorkflowLabelHandler> cannot load label by name")
				}
				// If label doesn't exist create him
				if err := project.InsertLabel(tx, &label); err != nil {
					return sdk.WrapError(err, "Cannot create new label")
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
			return sdk.WrapError(err, "cannot link label %d to workflow %s", label.ID, wf.Name)
		}
		newWf := *wf
		label.WorkflowID = wf.ID
		newWf.Labels = append(newWf.Labels, label)

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
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
		u := deprecatedGetUser(ctx)

		proj, errP := project.Load(db, api.Cache, key, u)
		if errP != nil {
			return sdk.WrapError(errP, "deleteWorkflowLabelHandler> cannot load project %s", key)
		}

		wf, errW := workflow.Load(ctx, db, api.Cache, proj, workflowName, u, workflow.LoadOptions{WithoutNode: true})
		if errW != nil {
			return sdk.WrapError(errW, "deleteWorkflowLabelHandler> cannot load workflow %s/%s", key, workflowName)
		}

		if err := workflow.UnLabelWorkflow(db, labelID, wf.ID); err != nil {
			return sdk.WrapError(err, "cannot unlink label %d to workflow %s", labelID, wf.Name)
		}

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

// postWorkflowHandler creates a new workflow
func (api *API) postWorkflowHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]

		p, errP := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx),
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithIntegrations,
		)
		if errP != nil {
			return sdk.WrapError(errP, "Cannot load Project %s", key)
		}
		var wf sdk.Workflow
		if err := service.UnmarshalBody(r, &wf); err != nil {
			return sdk.WrapError(err, "Cannot read body")
		}

		if wf.WorkflowData == nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "No node found")
		}

		if err := workflow.RenameNode(api.mustDB(), &wf); err != nil {
			return sdk.WrapError(err, "postWorkflowHandler> Cannot rename node")
		}

		(&wf).RetroMigrate()

		wf.ProjectID = p.ID
		wf.ProjectKey = key

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "Cannot start transaction")
		}
		defer tx.Rollback()

		if wf.Root != nil && wf.Root.Context != nil && (wf.Root.Context.Application != nil || wf.Root.Context.ApplicationID != 0) {
			var err error
			if wf.Root.Context.DefaultPayload, err = workflow.DefaultPayload(ctx, tx, api.Cache, p, &wf); err != nil {
				log.Warning("postWorkflowHandler> Cannot set default payload : %v", err)
			}
			wf.WorkflowData.Node.Context.DefaultPayload = wf.Root.Context.DefaultPayload
		}

		if err := workflow.Insert(tx, api.Cache, &wf, p, deprecatedGetUser(ctx)); err != nil {
			return sdk.WrapError(err, "Cannot insert workflow")
		}

		if errHr := workflow.HookRegistration(ctx, tx, api.Cache, nil, wf, p); errHr != nil {
			return sdk.WrapError(errHr, "postWorkflowHandler>Hook registration failed")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		wf1, errl := workflow.LoadByID(api.mustDB(), api.Cache, p, wf.ID, deprecatedGetUser(ctx), workflow.LoadOptions{})
		if errl != nil {
			return sdk.WrapError(errl, "Cannot load workflow")
		}

		//We filter project and workflow configurtaion key, because they are always set on insertHooks
		wf1.FilterHooksConfig(sdk.HookConfigProject, sdk.HookConfigWorkflow)

		// TODO REMOVE WHEN WE WILL DELETE OLD NODE STRUCT
		wf1.Root = nil
		wf1.Joins = nil
		return service.WriteJSON(w, wf1, http.StatusCreated)
	}
}

// putWorkflowHandler updates a workflow
func (api *API) putWorkflowHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		p, errP := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx),
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithIntegrations,
		)
		if errP != nil {
			return sdk.WrapError(errP, "putWorkflowHandler> Cannot load Project %s", key)
		}

		oldW, errW := workflow.Load(ctx, api.mustDB(), api.Cache, p, name, deprecatedGetUser(ctx), workflow.LoadOptions{WithIcon: true})
		if errW != nil {
			return sdk.WrapError(errW, "putWorkflowHandler> Cannot load Workflow %s", key)
		}

		var wf sdk.Workflow
		if err := service.UnmarshalBody(r, &wf); err != nil {
			return sdk.WrapError(err, "Cannot read body")
		}

		if err := workflow.RenameNode(api.mustDB(), &wf); err != nil {
			return sdk.WrapError(err, "Update> cannot check pipeline name")
		}

		// TODO : Delete in migration step 3
		// Retro migrate workflow
		(&wf).RetroMigrate()

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

		// TODO Remove old struct
		if wf.Root.Context != nil && (wf.Root.Context.Application != nil || wf.Root.Context.ApplicationID != 0) {
			var err error
			if wf.Root.Context.DefaultPayload, err = workflow.DefaultPayload(ctx, tx, api.Cache, p, &wf); err != nil {
				log.Warning("putWorkflowHandler> Cannot set default payload : %v", err)
			}
			wf.WorkflowData.Node.Context.DefaultPayload = wf.Root.Context.DefaultPayload
		}

		if err := workflow.Update(ctx, tx, api.Cache, &wf, oldW, p, deprecatedGetUser(ctx)); err != nil {
			return sdk.WrapError(err, "Cannot update workflow")
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
				return sdk.WrapError(err, "cannot update metadata")
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		wf1, errl := workflow.LoadByID(api.mustDB(), api.Cache, p, wf.ID, deprecatedGetUser(ctx), workflow.LoadOptions{})
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
		// TODO REMOVE
		wf1.Root = nil
		wf1.Joins = nil
		return service.WriteJSON(w, wf1, http.StatusOK)
	}
}

// putWorkflowHandler deletes a workflow
func (api *API) deleteWorkflowHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		p, errP := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx), project.LoadOptions.WithIntegrations)
		if errP != nil {
			return sdk.WrapError(errP, "Cannot load Project %s", key)
		}

		oldW, errW := workflow.Load(ctx, api.mustDB(), api.Cache, p, name, deprecatedGetUser(ctx), workflow.LoadOptions{})
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

		event.PublishWorkflowDelete(key, *oldW, deprecatedGetUser(ctx))

		sdk.GoRoutine(ctx, "deleteWorkflowHandler",
			func(ctx context.Context) {
				txg, errT := api.mustDB().Begin()
				if errT != nil {
					log.Error("deleteWorkflowHandler> Cannot start transaction: %v", errT)
				}
				defer txg.Rollback() // nolint
				if err := workflow.Delete(context.Background(), txg, api.Cache, p, oldW); err != nil {
					log.Error("deleteWorkflowHandler> unable to delete workflow: %v", err)
					return
				}
				if err := txg.Commit(); err != nil {
					log.Error("deleteWorkflowHandler> Cannot commit transaction: %v", err)
				}
			}, api.PanicDump())

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) getWorkflowHookHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		uuid := vars["uuid"]

		proj, errP := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx),
			project.LoadOptions.WithIntegrations,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithEnvironments)
		if errP != nil {
			return sdk.WrapError(errP, "Cannot load Project %s", key)
		}

		wf, errW := workflow.Load(ctx, api.mustDB(), api.Cache, proj, name, deprecatedGetUser(ctx), workflow.LoadOptions{})
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
			return sdk.WrapError(err, "Unable to get hook %s task and executions", uuid)
		}

		return service.WriteJSON(w, task, http.StatusOK)
	}
}
