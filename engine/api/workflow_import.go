package api

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/observability"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) postWorkflowPreviewHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]

		//Load project
		proj, errp := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx),
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithApplications,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithIntegrations,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
		)
		if errp != nil {
			return sdk.WrapError(errp, "postWorkflowPreviewHandler>> Unable load project")
		}

		body, errr := ioutil.ReadAll(r.Body)
		if errr != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errr)
		}
		defer r.Body.Close()

		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(body)
		}

		var ew = new(exportentities.Workflow)
		var errw error
		switch contentType {
		case "application/json":
			errw = json.Unmarshal(body, ew)
		case "application/x-yaml", "text/x-yaml":
			errw = yaml.Unmarshal(body, ew)
		default:
			return sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("unsupported content-type: %s", contentType))
		}

		if errw != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errw)
		}

		wf, globalError := workflow.Parse(proj, ew, deprecatedGetUser(ctx))
		if globalError != nil {
			return sdk.WrapError(globalError, "postWorkflowPreviewHandler> Unable import workflow %s", ew.Name)
		}

		// Browse all node to find IDs
		if err := workflow.IsValid(ctx, api.Cache, api.mustDB(), wf, proj, deprecatedGetUser(ctx), workflow.LoadOptions{}); err != nil {
			return sdk.WrapError(err, "Workflow is not valid")
		}

		if err := workflow.RenameNode(api.mustDB(), wf); err != nil {
			return sdk.WrapError(err, "Unable to rename node")
		}

		return service.WriteJSON(w, wf, http.StatusOK)
	}
}

func (api *API) postWorkflowImportHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		force := FormBool(r, "force")

		//Load project
		proj, errp := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx),
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithApplications,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithIntegrations,
		)
		if errp != nil {
			return sdk.WrapError(errp, "Unable load project")
		}

		body, errr := ioutil.ReadAll(r.Body)
		if errr != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errr)
		}
		defer r.Body.Close()

		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(body)
		}

		var ew = new(exportentities.Workflow)
		var errw error
		switch contentType {
		case "application/json":
			errw = json.Unmarshal(body, ew)
		case "application/x-yaml", "text/x-yaml":
			errw = yaml.Unmarshal(body, ew)
		default:
			return sdk.WrapError(sdk.ErrWrongRequest, "Unsupported content-type: %s", contentType)
		}

		if errw != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errw)
		}

		tx, errtx := api.mustDB().Begin()
		if errtx != nil {
			return sdk.WrapError(errtx, "Unable to start transaction")
		}
		defer tx.Rollback()

		u := deprecatedGetUser(ctx)

		// load the workflow from database if exists
		workflowExists, err := workflow.Exists(tx, proj.Key, ew.Name)
		if err != nil {
			return sdk.WrapError(err, "Cannot check if workflow exists")
		}
		var wf *sdk.Workflow
		if workflowExists {
			wf, err = workflow.Load(ctx, tx, api.Cache, proj, ew.Name, u, workflow.LoadOptions{WithIcon: true})
			if err != nil {
				return sdk.WrapError(err, "Unable to load existing workflow")
			}
		}

		wrkflw, msgList, globalError := workflow.ParseAndImport(ctx, tx, api.Cache, proj, wf, ew, deprecatedGetUser(ctx), workflow.ImportOptions{Force: force})
		msgListString := translate(r, msgList)
		if globalError != nil {
			return sdk.WrapError(globalError, "Unable to import workflow %s", ew.Name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		if wf != nil {
			event.PublishWorkflowUpdate(proj.Key, *wrkflw, *wf, u)
		} else {
			event.PublishWorkflowAdd(proj.Key, *wrkflw, u)
		}

		if wrkflw != nil {
			w.Header().Add(sdk.ResponseWorkflowIDHeader, fmt.Sprintf("%d", wrkflw.ID))
			w.Header().Add(sdk.ResponseWorkflowNameHeader, wrkflw.Name)
		}

		return service.WriteJSON(w, msgListString, http.StatusOK)
	}
}

func (api *API) putWorkflowImportHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		wfName := vars["permWorkflowName"]

		// Load project
		proj, errp := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx),
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithApplications,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithIntegrations,
		)
		if errp != nil {
			return sdk.WrapError(errp, "Unable load project")
		}

		u := deprecatedGetUser(ctx)

		wf, err := workflow.Load(ctx, api.mustDB(), api.Cache, proj, wfName, u, workflow.LoadOptions{WithIcon: true})
		if err != nil {
			return sdk.WrapError(err, "Unable to load workflow")
		}

		body, errr := ioutil.ReadAll(r.Body)
		if errr != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errr)
		}
		defer r.Body.Close()

		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(body)
		}

		var ew = new(exportentities.Workflow)
		var errw error
		switch contentType {
		case "application/json":
			errw = json.Unmarshal(body, ew)
		case "application/x-yaml", "text/x-yaml":
			errw = yaml.Unmarshal(body, ew)
		default:
			return sdk.WrapError(sdk.ErrWrongRequest, "Unsupported content-type: %s", contentType)
		}

		if errw != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errw)
		}

		tx, errtx := api.mustDB().Begin()
		if errtx != nil {
			return sdk.WrapError(errtx, "Unable to start transaction")
		}
		defer func() {
			_ = tx.Rollback()
		}()

		wrkflw, msgList, globalError := workflow.ParseAndImport(ctx, tx, api.Cache, proj, wf, ew, deprecatedGetUser(ctx), workflow.ImportOptions{Force: true, WorkflowName: wfName})
		msgListString := translate(r, msgList)
		if globalError != nil {

			return sdk.WrapError(globalError, "Unable to import workflow %s", ew.Name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		oldW, errL := workflow.Load(ctx, api.mustDB(), api.Cache, proj, wfName, deprecatedGetUser(ctx), workflow.LoadOptions{})
		if errL == nil {
			event.PublishWorkflowUpdate(key, *wrkflw, *oldW, deprecatedGetUser(ctx))
		}

		if wrkflw != nil {
			w.Header().Add(sdk.ResponseWorkflowIDHeader, fmt.Sprintf("%d", wrkflw.ID))
			w.Header().Add(sdk.ResponseWorkflowNameHeader, wrkflw.Name)
		}

		return service.WriteJSON(w, msgListString, http.StatusOK)
	}
}

func (api *API) postWorkflowPushHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		db := api.mustDB()
		vars := mux.Vars(r)
		key := vars[permProjectKey]

		observability.Current(ctx,
			observability.Tag(observability.TagProjectKey, key),
		)

		if r.Body == nil {
			return sdk.ErrWrongRequest
		}

		btes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Error("postWorkflowPushHandler> Unable to read body: %v", err)
			return sdk.ErrWrongRequest
		}
		defer r.Body.Close()

		log.Debug("Read %d bytes from body", len(btes))
		tr := tar.NewReader(bytes.NewReader(btes))

		var pushOptions *workflow.PushOption
		if r.Header.Get(sdk.WorkflowAsCodeHeader) != "" {
			pushOptions = &workflow.PushOption{
				FromRepository:  r.Header.Get(sdk.WorkflowAsCodeHeader),
				IsDefaultBranch: true,
				Force:           FormBool(r, "force"),
			}
		}

		//Load project
		proj, errp := project.Load(db, api.Cache, key, deprecatedGetUser(ctx),
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithApplications,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithIntegrations)
		if errp != nil {
			return sdk.WrapError(errp, "postWorkflowPushHandler> Cannot load project %s", key)
		}

		allMsg, wrkflw, err := workflow.Push(ctx, db, api.Cache, proj, tr, pushOptions, deprecatedGetUser(ctx), project.DecryptWithBuiltinKey)
		if err != nil {
			return err
		}
		msgListString := translate(r, allMsg)

		if wrkflw != nil {
			w.Header().Add(sdk.ResponseWorkflowIDHeader, fmt.Sprintf("%d", wrkflw.ID))
			w.Header().Add(sdk.ResponseWorkflowNameHeader, wrkflw.Name)
		}

		return service.WriteJSON(w, msgListString, http.StatusOK)
	}
}
