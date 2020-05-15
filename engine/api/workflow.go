package api

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/integration"
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
		filterByProject := r.FormValue(permProjectKey)
		filterByRepo := r.FormValue("repo")

		var opts = workflow.LoadAllWorkflowsOptions{}

		if filterByProject != "" {
			opts.Filters.ProjectKey = filterByProject
		}

		if filterByRepo != "" {
			opts.Filters.ApplicationRepository = filterByRepo
		}

		opts.Loaders.WithFavoritesForUserID = getAPIConsumer(ctx).AuthentifiedUserID

		groupIDS := getAPIConsumer(ctx).GetGroupIDs()
		opts.Filters.GroupIDs = groupIDS
		if isMaintainer(ctx) {
			opts.Filters.GroupIDs = nil
		}

		ws, err := workflow.LoadAllWorkflows(ctx, api.mustDBWithCtx(ctx), opts)
		if err != nil {
			return err
		}

		ids := ws.IDs()
		perms, err := permission.LoadWorkflowMaxLevelPermissionByWorkflowIDs(ctx, api.mustDB(), ids, groupIDS)
		if err != nil {
			return err
		}

		for i := range ws {
			if isAdmin(ctx) {
				ws[i].Permissions = sdk.Permissions{Readable: true, Writable: true, Executable: true}
			} else {
				idString := strconv.FormatInt(ws[i].ID, 10)
				ws[i].Permissions = perms.Permissions(idString)
				if isMaintainer(ctx) {
					ws[i].Permissions.Readable = true
				}
			}
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
		minimal := FormBool(r, "minimal")
		withoutIcons := FormBool(r, "withoutIcons")

		proj, err := project.Load(api.mustDB(), key, project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WrapError(err, "unable to load projet")
		}

		opts := workflow.LoadOptions{
			Minimal:                minimal, // if true, load only data from table workflow, not pipelines, app, env...
			DeepPipeline:           withDeepPipelines,
			WithIcon:               !withoutIcons,
			WithLabels:             withLabels,
			WithAsCodeUpdateEvent:  withAsCodeEvents,
			WithIntegrations:       true,
			WithTemplate:           withTemplate,
			WithFavoritesForUserID: getAPIConsumer(ctx).AuthentifiedUserID,
		}
		w1, err := workflow.Load(ctx, api.mustDB(), api.Cache, *proj, name, opts)
		if err != nil {
			return sdk.WrapError(err, "cannot load workflow %s", name)
		}

		if withUsage {
			usage, err := loadWorkflowUsage(api.mustDB(), w1.ID)
			if err != nil {
				return sdk.WrapError(err, "cannot load usage for workflow %s", name)
			}
			w1.Usage = &usage
		}

		if withAudits {
			audits, err := workflow.LoadAudits(api.mustDB(), w1.ID)
			if err != nil {
				return sdk.WrapError(err, "cannot load audits for workflow %s", name)
			}
			w1.Audits = audits
		}

		if isAdmin(ctx) {
			w1.Permissions = sdk.Permissions{Readable: true, Writable: true, Executable: true}
		} else {
			perms, err := permission.LoadWorkflowMaxLevelPermission(ctx, api.mustDB(), key, []string{w1.Name}, getAPIConsumer(ctx).GetGroupIDs())
			if err != nil {
				return err
			}
			w1.Permissions = perms.Permissions(w1.Name)
			if isMaintainer(ctx) {
				w1.Permissions.Readable = true
			}
		}

		w1.URLs.APIURL = api.Config.URL.API + api.Router.GetRoute("GET", api.getWorkflowHandler, map[string]string{"key": key, "permWorkflowName": w1.Name})
		w1.URLs.UIURL = api.Config.URL.UI + "/project/" + key + "/workflow/" + w1.Name

		//We filter project and workflow configuration key, because they are always set on insertHooks
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
		u := getAPIConsumer(ctx)

		proj, err := project.Load(db, key,
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithApplications,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithIntegrations,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
		)
		if err != nil {
			return sdk.WrapError(err, "cannot load project %s", key)
		}

		wf, err := workflow.Load(ctx, db, api.Cache, *proj, workflowName, workflow.LoadOptions{WithIcon: true})
		if err != nil {
			return sdk.WrapError(err, "cannot load workflow %s/%s", key, workflowName)
		}

		audit, err := workflow.LoadAudit(db, auditID, wf.ID)
		if err != nil {
			return sdk.WrapError(err, "cannot load workflow audit %s/%s", key, workflowName)
		}

		exportWf, err := exportentities.UnmarshalWorkflow([]byte(audit.DataBefore), exportentities.FormatYAML)
		if err != nil {
			return sdk.WrapError(err, "cannot unmarshal data before")
		}

		tx, err := db.Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot begin transaction")
		}
		defer func() {
			_ = tx.Rollback()
		}()

		newWf, _, errP := workflow.ParseAndImport(ctx, tx, api.Cache, *proj, wf, exportWf, u, workflow.ImportOptions{Force: true, WorkflowName: workflowName})
		if errP != nil {
			return sdk.WrapError(errP, "cannot parse and import previous workflow")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		newWf.Permissions.Readable = true
		newWf.Permissions.Executable = true
		newWf.Permissions.Writable = true

		event.PublishWorkflowUpdate(ctx, key, *wf, *newWf, getAPIConsumer(ctx))

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

		var label sdk.Label
		if err := service.UnmarshalBody(r, &label); err != nil {
			return sdk.WrapError(err, "cannot read body")
		}

		proj, err := project.Load(db, key,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithIntegrations,
		)
		if err != nil {
			return sdk.WrapError(err, "cannot load project %s", key)
		}
		label.ProjectID = proj.ID

		tx, errTx := db.Begin()
		if errTx != nil {
			return sdk.WrapError(errTx, "cannot create new transaction")
		}
		defer tx.Rollback() //nolint

		if label.ID == 0 {
			if label.Name == "" {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "label ID or label name should not be empty")
			}

			lbl, err := project.LabelByName(tx, proj.ID, label.Name)
			if err != nil {
				if sdk.Cause(err) != sql.ErrNoRows {
					return sdk.WrapError(err, "cannot load label by name")
				}
				// If label doesn't exist create him
				if err := project.InsertLabel(tx, &label); err != nil {
					return sdk.WrapError(err, "cannot create new label")
				}
			} else {
				label.ID = lbl.ID
			}
		}

		wf, err := workflow.Load(ctx, tx, api.Cache, *proj, workflowName, workflow.LoadOptions{WithLabels: true})
		if err != nil {
			return sdk.WrapError(err, "cannot load workflow %s/%s", key, workflowName)
		}

		if err := workflow.LabelWorkflow(tx, label.ID, wf.ID); err != nil {
			return sdk.WrapError(err, "cannot link label %d to workflow %s", label.ID, wf.Name)
		}
		newWf := *wf
		label.WorkflowID = wf.ID
		newWf.Labels = append(newWf.Labels, label)

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
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
			return sdk.WrapError(errV, "cannot convert to int labelID")
		}

		db := api.mustDB()

		proj, err := project.Load(db, key,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithIntegrations,
		)
		if err != nil {
			return sdk.WrapError(err, "cannot load project %s", key)
		}

		wf, err := workflow.Load(ctx, db, api.Cache, *proj, workflowName, workflow.LoadOptions{})
		if err != nil {
			return sdk.WrapError(err, "cannot load workflow %s/%s", key, workflowName)
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

		p, err := project.Load(api.mustDB(), key,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithIntegrations,
		)
		if err != nil {
			return err
		}
		var data sdk.Workflow
		if err := service.UnmarshalBody(r, &data); err != nil {
			return err
		}

		if err := workflow.RenameNode(ctx, api.mustDB(), &data); err != nil {
			return err
		}

		data.ProjectID = p.ID
		data.ProjectKey = key

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WithStack(errT)
		}
		defer tx.Rollback() // nolint

		if err := workflow.Insert(ctx, tx, api.Cache, *p, &data); err != nil {
			return sdk.WrapError(err, "cannot insert workflow")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		wf, err := workflow.LoadByID(ctx, api.mustDB(), api.Cache, *p, data.ID, workflow.LoadOptions{})
		if err != nil {
			return sdk.WrapError(err, "cannot load workflow")
		}

		event.PublishWorkflowAdd(ctx, p.Key, *wf, getAPIConsumer(ctx))

		wf.Permissions.Readable = true
		wf.Permissions.Writable = true
		wf.Permissions.Executable = true

		//We filter project and workflow configurtaion key, because they are always set on insertHooks
		wf.FilterHooksConfig(sdk.HookConfigProject, sdk.HookConfigWorkflow)

		return service.WriteJSON(w, wf, http.StatusCreated)
	}
}

// putWorkflowHandler updates a workflow
func (api *API) putWorkflowHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		p, err := project.Load(api.mustDB(), key,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithIntegrations,
		)
		if err != nil {
			return sdk.WrapError(err, "cannot load Project %s", key)
		}

		oldW, err := workflow.Load(ctx, api.mustDB(), api.Cache, *p, name,
			workflow.LoadOptions{WithIcon: true, WithIntegrations: true})
		if err != nil {
			return sdk.WrapError(err, "cannot load Workflow %s", key)
		}

		if oldW.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		var wf sdk.Workflow
		if err := service.UnmarshalBody(r, &wf); err != nil {
			return sdk.WrapError(err, "cannot read body")
		}

		if err := workflow.RenameNode(ctx, api.mustDB(), &wf); err != nil {
			return sdk.WrapError(err, "cannot check pipeline name")
		}

		wf.ID = oldW.ID
		wf.ProjectID = p.ID
		wf.ProjectKey = key

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err := workflow.Update(ctx, tx, api.Cache, *p, &wf, workflow.UpdateOptions{}); err != nil {
			return sdk.WrapError(err, "cannot update workflow")
		}

		if defaultTags, ok := wf.Metadata["default_tags"]; wf.WorkflowData.Node.IsLinkedToRepo(&wf) && (!ok || defaultTags == "") {
			if wf.Metadata == nil {
				wf.Metadata = sdk.Metadata{}
			}
			wf.Metadata["default_tags"] = "git.branch,git.author"
			if err := workflow.UpdateMetadata(tx, wf.ID, wf.Metadata); err != nil {
				return sdk.WrapError(err, "cannot update metadata")
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		wf1, err := workflow.LoadByID(ctx, api.mustDB(), api.Cache, *p, wf.ID, workflow.LoadOptions{WithIntegrations: true})
		if err != nil {
			return sdk.WrapError(err, "cannot load workflow")
		}

		event.PublishWorkflowUpdate(ctx, p.Key, *wf1, *oldW, getAPIConsumer(ctx))

		wf1.Permissions.Readable = true
		wf1.Permissions.Writable = true
		wf1.Permissions.Executable = true

		usage, err := loadWorkflowUsage(api.mustDB(), wf1.ID)
		if err != nil {
			return sdk.WrapError(err, "cannot load usage")
		}
		wf1.Usage = &usage

		//We filter project and workflow configuration key, because they are always set on insertHooks
		wf1.FilterHooksConfig(sdk.HookConfigProject, sdk.HookConfigWorkflow)
		return service.WriteJSON(w, wf1, http.StatusOK)
	}
}

// putWorkflowIconHandler updates a workflow
func (api *API) putWorkflowIconHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		p, errP := project.Load(api.mustDB(), key)
		if errP != nil {
			return errP
		}

		imageBts, errr := ioutil.ReadAll(r.Body)
		if errr != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errr)
		}
		defer r.Body.Close()

		icon := string(imageBts)
		if !strings.HasPrefix(icon, sdk.IconFormat) {
			return sdk.WithStack(sdk.ErrIconBadFormat)
		}
		if len(icon) > sdk.MaxIconSize {
			return sdk.WithStack(sdk.ErrIconBadSize)
		}

		wf, err := workflow.Load(ctx, api.mustDB(), api.Cache, *p, name, workflow.LoadOptions{
			Minimal: true,
		})
		if err != nil {
			return err
		}

		if err := workflow.UpdateIcon(api.mustDB(), wf.ID, icon); err != nil {
			return err
		}

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

// deleteWorkflowIconHandler updates a workflow
func (api *API) deleteWorkflowIconHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		p, errP := project.Load(api.mustDB(), key)
		if errP != nil {
			return errP
		}

		wf, err := workflow.Load(ctx, api.mustDB(), api.Cache, *p, name, workflow.LoadOptions{})
		if err != nil {
			return err
		}

		if err := workflow.UpdateIcon(api.mustDB(), wf.ID, ""); err != nil {
			return err
		}

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

// deleteWorkflowHandler deletes a workflow
func (api *API) deleteWorkflowHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		p, errP := project.Load(api.mustDB(), key, project.LoadOptions.WithIntegrations)
		if errP != nil {
			return sdk.WrapError(errP, "Cannot load Project %s", key)
		}

		b, errW := workflow.Exists(api.mustDB(), key, name)
		if errW != nil {
			return sdk.WrapError(errW, "Cannot check Workflow %s", key)
		}
		if !b {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "Cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err := workflow.MarkAsDelete(tx, key, name); err != nil {
			return sdk.WrapError(err, "Cannot delete workflow")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(errT, "Cannot commit transaction")
		}
		consumer := getAPIConsumer(ctx)
		sdk.GoRoutine(api.Router.Background, "deleteWorkflowHandler",
			func(ctx context.Context) {
				txg, errT := api.mustDB().Begin()
				if errT != nil {
					log.Error(ctx, "deleteWorkflowHandler> Cannot start transaction: %v", errT)
				}
				defer txg.Rollback() // nolint

				oldW, err := workflow.Load(ctx, txg, api.Cache, *p, name, workflow.LoadOptions{})
				if err != nil {
					log.Error(ctx, "deleteWorkflowHandler> unable to load workflow: %v", err)
					return
				}

				if err := workflow.Delete(ctx, txg, api.Cache, *p, oldW); err != nil {
					log.Error(ctx, "deleteWorkflowHandler> unable to delete workflow: %v", err)
					return
				}
				if err := txg.Commit(); err != nil {
					log.Error(ctx, "deleteWorkflowHandler> Cannot commit transaction: %v", err)
				}
				event.PublishWorkflowDelete(ctx, key, *oldW, consumer)
			}, api.PanicDump())

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

// deleteWorkflowEventsIntegrationHandler deletes a workflow event integration
func (api *API) deleteWorkflowEventsIntegrationHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		prjIntegrationIDStr := vars["integrationID"]
		db := api.mustDB()

		prjIntegrationID, err := strconv.ParseInt(prjIntegrationIDStr, 10, 64)
		if err != nil {
			return sdk.WrapError(sdk.ErrInvalidID, "integration id is not correct (%s) : %v", prjIntegrationIDStr, err)
		}

		p, err := project.Load(db, key, project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WrapError(err, "cannot load Project %s", key)
		}

		wf, err := workflow.Load(ctx, db, api.Cache, *p, name, workflow.LoadOptions{WithIntegrations: true})
		if err != nil {
			return sdk.WrapError(err, "cannot load Workflow %s", key)
		}

		if err := integration.RemoveFromWorkflow(db, wf.ID, prjIntegrationID); err != nil {
			return sdk.WrapError(err, "cannot remove integration id %d from workflow %s (id: %d)", prjIntegrationID, wf.Name, wf.ID)
		}

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) getWorkflowHookHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		uuid := vars["uuid"]

		proj, err := project.Load(api.mustDB(), key,
			project.LoadOptions.WithIntegrations,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithEnvironments)
		if err != nil {
			return sdk.WrapError(err, "cannot load Project %s", key)
		}

		wf, err := workflow.Load(ctx, api.mustDB(), api.Cache, *proj, name, workflow.LoadOptions{})
		if err != nil {
			return sdk.WrapError(err, "cannot load Workflow %s/%s", key, name)
		}

		whooks := wf.WorkflowData.GetHooks()
		_, has := whooks[uuid]
		if !has {
			return sdk.WrapError(sdk.ErrNotFound, "cannot load Workflow %s/%s hook %s", key, name, uuid)
		}

		//Push the hook to hooks µService
		//Load service "hooks"
		srvs, err := services.LoadAllByType(ctx, api.mustDB(), services.TypeHooks)
		if err != nil {
			return sdk.WrapError(err, "unable to load hooks services")
		}

		path := fmt.Sprintf("/task/%s/execution", uuid)
		task := sdk.Task{}
		if _, _, err := services.NewClient(api.mustDB(), srvs).DoJSONRequest(ctx, "GET", path, nil, &task); err != nil {
			return sdk.WrapError(err, "unable to get hook %s task and executions", uuid)
		}

		return service.WriteJSON(w, task, http.StatusOK)
	}
}

func (api *API) getWorkflowNotificationsConditionsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		data := struct {
			Operators      map[string]string `json:"operators"`
			ConditionNames []string          `json:"names"`
		}{
			Operators: sdk.WorkflowConditionsOperators,
		}

		wr, errr := workflow.LoadLastRun(api.mustDB(), key, name, workflow.LoadRunOptions{})
		if errr != nil {
			if !sdk.ErrorIs(errr, sdk.ErrNotFound) {
				return sdk.WrapError(errr, "getWorkflowTriggerConditionHandler> Unable to load last run workflow")
			}
		}

		params := []sdk.Parameter{}
		var refNode *sdk.Node
		if wr != nil {
			refNode = &wr.Workflow.WorkflowData.Node
			var errp error
			params, errp = workflow.NodeBuildParametersFromRun(*wr, refNode.ID)
			if errp != nil {
				return sdk.WrapError(errp, "getWorkflowTriggerConditionHandler> Unable to load build parameters from workflow run")
			}

			if len(params) == 0 {
				refNode = nil
			}
		} else {
			data.ConditionNames = append(data.ConditionNames, sdk.BasicVariableNames...)
		}

		if sdk.ParameterFind(params, "git.repository") == nil {
			data.ConditionNames = append(data.ConditionNames, sdk.BasicGitVariableNames...)
		}
		if sdk.ParameterFind(params, "git.tag") == nil {
			data.ConditionNames = append(data.ConditionNames, "git.tag")
		}

		for _, p := range params {
			data.ConditionNames = append(data.ConditionNames, p.Name)
		}

		sort.Strings(data.ConditionNames)
		return service.WriteJSON(w, data, http.StatusOK)
	}
}

func (api *API) getAllWorkflowsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		// TODO

		return nil
	}
}
