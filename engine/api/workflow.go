package api

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/fsamin/go-dump"
	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
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

		return WriteJSON(w, ws, http.StatusOK)
	}
}

// getWorkflowHandler returns a full workflow
func (api *API) getWorkflowHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		withUsage := FormBool(r, "withUsage")

		w1, err := workflow.Load(api.mustDB(), api.Cache, key, name, getUser(ctx), workflow.LoadOptions{WithFavorite: true})
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

		w1.Permission = permission.WorkflowPermission(key, w1.Name, getUser(ctx))

		//We filter project and workflow configurtaion key, because they are always set on insertHooks
		w1.FilterHooksConfig(sdk.HookConfigProject, sdk.HookConfigWorkflow)

		return WriteJSON(w, w1, http.StatusOK)
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

// postWorkflowHandler creates a new workflow
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

		if wf.Root != nil && wf.Root.Context != nil && (wf.Root.Context.Application != nil || wf.Root.Context.ApplicationID != 0) {
			var err error
			if wf.Root.Context.DefaultPayload, err = getDefaultPayload(tx, api.Cache, p, getUser(ctx), &wf); err != nil {
				return sdk.WrapError(err, "putWorkflowHandler> Cannot set default payload")
			}
		}

		defaultPayload, errHr := workflow.HookRegistration(tx, api.Cache, nil, wf, p)
		if errHr != nil {
			return sdk.WrapError(errHr, "postWorkflowHandler")
		}
		if defaultPayload != nil && isDefaultPayloadEmpty(wf) {
			wf.Root.Context.DefaultPayload = *defaultPayload
		}

		if err := workflow.Insert(tx, api.Cache, &wf, p, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "Cannot insert workflow")
		}

		// Add group
		for _, gp := range p.ProjectGroups {
			if gp.Permission >= permission.PermissionReadExecute {
				if err := workflow.AddGroup(tx, &wf, gp); err != nil {
					return sdk.WrapError(err, "Cannot add group %s", gp.Group.Name)
				}
			}
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectWorkflowLastModificationType); err != nil {
			return sdk.WrapError(err, "Cannot update project last modified date")
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectWorkflowLastModificationType); err != nil {
			return sdk.WrapError(err, "postWorkflowHandler> Cannot update project workflows last modified")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		wf1, errl := workflow.LoadByID(api.mustDB(), api.Cache, wf.ID, getUser(ctx), workflow.LoadOptions{})
		if errl != nil {
			return sdk.WrapError(errl, "Cannot load workflow")
		}

		//We filter project and workflow configurtaion key, because they are always set on insertHooks
		wf1.FilterHooksConfig(sdk.HookConfigProject, sdk.HookConfigWorkflow)

		return WriteJSON(w, wf1, http.StatusCreated)
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

		oldW, errW := workflow.Load(api.mustDB(), api.Cache, key, name, getUser(ctx), workflow.LoadOptions{})
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
			if wf.Root.Context.DefaultPayload, err = getDefaultPayload(tx, api.Cache, p, getUser(ctx), &wf); err != nil {
				return sdk.WrapError(err, "putWorkflowHandler> Cannot set default payload")
			}
		}

		if err := workflow.Update(tx, api.Cache, &wf, oldW, p, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "putWorkflowHandler> Cannot update workflow")
		}

		// HookRegistration after workflow.Update.  It needs hooks to be created on DB
		defaultPayload, errHr := workflow.HookRegistration(tx, api.Cache, oldW, wf, p)
		if errHr != nil {
			return sdk.WrapError(errHr, "putWorkflowHandler")
		}

		if defaultPayload != nil && isDefaultPayloadEmpty(wf) {
			wf.Root.Context.DefaultPayload = *defaultPayload
			if err := workflow.UpdateNodeContext(tx, wf.Root.Context); err != nil {
				return sdk.WrapError(err, "putWorkflowHandler> updateNodeContext")
			}
		} else if defaultPayload != nil || (wf.Root.Context != nil && wf.Root.Context.Application != nil && wf.Root.Context.Application.RepositoryFullname != "") {
			wf.Metadata = getUpdatedMetadata(wf.Metadata)
			if err := workflow.UpdateMetadata(tx, wf.ID, wf.Metadata); err != nil {
				return sdk.WrapError(err, "putWorkflowHandler> cannot update metadata")
			}
		}

		if err := workflow.UpdateLastModifiedDate(tx, api.Cache, getUser(ctx), p.Key, oldW); err != nil {
			return sdk.WrapError(err, "putWorkflowHandler> Cannot update last modified date for workflow")
		}

		if oldW.Name != wf.Name {
			if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p, sdk.ProjectWorkflowLastModificationType); err != nil {
				return sdk.WrapError(err, "putWorkflowHandler> Cannot update project last modified date")
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "putWorkflowHandler> Cannot commit transaction")
		}

		wf1, errl := workflow.LoadByID(api.mustDB(), api.Cache, wf.ID, getUser(ctx), workflow.LoadOptions{})
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

		return WriteJSON(w, wf1, http.StatusOK)
	}
}

// postWorkflowFavoriteHandler add or delete this workflow from favorite user
func (api *API) postWorkflowFavoriteHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		wf, errW := workflow.Load(api.mustDB(), api.Cache, key, name, getUser(ctx), workflow.LoadOptions{WithFavorite: true})
		if errW != nil {
			return sdk.WrapError(errW, "postWorkflowFavoriteHandler> Cannot load workflow %s/%s", key, name)
		}

		if err := workflow.UpdateFavorite(api.mustDB(), wf.ID, getUser(ctx), !wf.Favorite); err != nil {
			return sdk.WrapError(err, "postWorkflowFavoriteHandler> Cannot change workflow %s/%s favorite", key, name)
		}
		wf.Favorite = !wf.Favorite

		return WriteJSON(w, wf, http.StatusOK)
	}
}

func isDefaultPayloadEmpty(wf sdk.Workflow) bool {
	e := dump.NewDefaultEncoder(new(bytes.Buffer))
	e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
	e.ExtraFields.DetailedMap = false
	e.ExtraFields.DetailedStruct = false
	e.ExtraFields.Len = false
	e.ExtraFields.Type = false
	m, err := e.ToStringMap(wf.Root.Context.DefaultPayload)
	if err != nil {
		log.Warning("isDefaultPayloadEmpty>error while dump wf.Root.Context.DefaultPayload")
	}
	return len(m) == 0 // if empty, return true
}

func getUpdatedMetadata(metadata sdk.Metadata) sdk.Metadata {
	defaultTags, ok := metadata["default_tags"]
	if ok {
		var gitAuthor, gitBranch bool
		tagsList := strings.Split(defaultTags, ",")

		for _, tag := range tagsList {
			switch tag {
			case "git.branch":
				gitBranch = true
			case "git.author":
				gitAuthor = true
			}
		}

		if !gitAuthor {
			defaultTags = "git.author," + defaultTags
		}

		if !gitBranch {
			defaultTags = "git.branch," + defaultTags
		}
	} else {
		defaultTags = "git.branch,git.author"
	}

	if metadata == nil {
		metadata = sdk.Metadata{}
	}
	metadata["default_tags"] = defaultTags

	return metadata
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

		oldW, errW := workflow.Load(api.mustDB(), api.Cache, key, name, getUser(ctx), workflow.LoadOptions{})
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
		return WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) getWorkflowHookHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		uuid := vars["uuid"]

		wf, errW := workflow.Load(api.mustDB(), api.Cache, key, name, getUser(ctx), workflow.LoadOptions{})
		if errW != nil {
			return sdk.WrapError(errW, "getWorkflowHookHandler> Cannot load Workflow %s/%s", key, name)
		}

		whooks := wf.GetHooks()
		_, has := whooks[uuid]
		if !has {
			return sdk.WrapError(sdk.ErrNotFound, "getWorkflowHookHandler> Cannot load Workflow %s/%s hook %s", key, name, uuid)
		}

		//Push the hook to hooks µService
		dao := services.Querier(api.mustDB(), api.Cache)
		//Load service "hooks"
		srvs, errS := dao.FindByType("hooks")
		if errS != nil {
			return sdk.WrapError(errS, "getWorkflowHookHandler> Unable to load hooks services")
		}

		path := fmt.Sprintf("/task/%s/execution", uuid)
		task := sdk.Task{}
		if _, err := services.DoJSONRequest(srvs, "GET", path, nil, &task); err != nil {
			return sdk.WrapError(err, "getWorkflowHookHandler> Unable to get hook %s task and executions", uuid)
		}

		return WriteJSON(w, task, http.StatusOK)
	}
}

func getDefaultPayload(db gorp.SqlExecutor, store cache.Store, p *sdk.Project, u *sdk.User, wf *sdk.Workflow) (interface{}, error) {
	var defaultPayload interface{}
	appID := wf.Root.Context.ApplicationID
	if wf.Root.Context.Application != nil {
		appID = wf.Root.Context.Application.ID
	}
	app, errLa := application.LoadByID(db, store, appID, u)
	if errLa != nil {
		return wf.Root.Context.DefaultPayload, sdk.WrapError(errLa, "getDefaultPayload> unable to load application by id %d", appID)
	}
	wf.Root.Context.Application = app

	if app.RepositoryFullname != "" {
		defaultBranch := "master"
		projectVCSServer := repositoriesmanager.GetProjectVCSServer(p, app.VCSServer)
		if projectVCSServer != nil {
			client, errclient := repositoriesmanager.AuthorizedClient(db, store, projectVCSServer)
			if errclient != nil {
				return wf.Root.Context.DefaultPayload, sdk.WrapError(errclient, "getDefaultPayload> Cannot get authorized client")
			}

			branches, errBr := client.Branches(app.RepositoryFullname)
			if errBr != nil {
				return wf.Root.Context.DefaultPayload, sdk.WrapError(errBr, "getDefaultPayload> Cannot get branches for %s", app.RepositoryFullname)
			}

			for _, branch := range branches {
				if branch.Default {
					defaultBranch = branch.DisplayID
					break
				}
			}
		}

		if wf.Root.Context.HasDefaultPayload() {
			defaultPayload = sdk.WorkflowNodeContextDefaultPayloadVCS{
				GitBranch: defaultBranch,
			}
		} else if defaultPayloadMap, err := wf.Root.Context.DefaultPayloadToMap(); err == nil && defaultPayloadMap["git.branch"] == "" {
			defaultPayloadMap["git.branch"] = defaultBranch
			defaultPayload = defaultPayloadMap
		}
	}

	return defaultPayload, nil
}
