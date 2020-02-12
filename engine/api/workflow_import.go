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
	"github.com/ovh/cds/engine/api/workflowtemplate"
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
		proj, errp := project.Load(api.mustDB(), api.Cache, key,
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

		wf, globalError := workflow.Parse(ctx, proj, ew)
		if globalError != nil {
			return sdk.WrapError(globalError, "postWorkflowPreviewHandler> Unable import workflow %s", ew.Name)
		}

		// Browse all node to find IDs
		if err := workflow.IsValid(ctx, api.Cache, api.mustDB(), wf, proj, workflow.LoadOptions{}); err != nil {
			return sdk.WrapError(err, "Workflow is not valid")
		}

		if err := workflow.RenameNode(ctx, api.mustDB(), wf); err != nil {
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
		proj, errp := project.Load(api.mustDB(), api.Cache, key,
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
		defer tx.Rollback() // nolint

		u := getAPIConsumer(ctx)

		// load the workflow from database if exists
		workflowExists, err := workflow.Exists(tx, proj.Key, ew.Name)
		if err != nil {
			return sdk.WrapError(err, "Cannot check if workflow exists")
		}
		var wf *sdk.Workflow
		if workflowExists {
			wf, err = workflow.Load(ctx, tx, api.Cache, proj, ew.Name, workflow.LoadOptions{WithIcon: true})
			if err != nil {
				return sdk.WrapError(err, "Unable to load existing workflow")
			}
		}

		wrkflw, msgList, globalError := workflow.ParseAndImport(ctx, tx, api.Cache, proj, wf, ew, getAPIConsumer(ctx), workflow.ImportOptions{Force: force})
		msgListString := translate(r, msgList)
		if globalError != nil {
			if len(msgListString) != 0 {
				sdkErr := sdk.ExtractHTTPError(globalError, r.Header.Get("Accept-Language"))
				return service.WriteJSON(w, append(msgListString, sdkErr.Message), sdkErr.Status)
			}
			if len(msgListString) != 0 {
				sdkErr := sdk.ExtractHTTPError(globalError, r.Header.Get("Accept-Language"))
				return service.WriteJSON(w, append(msgListString, sdkErr.Message), sdkErr.Status)
			}
			return sdk.WrapError(globalError, "Unable to import workflow %s", ew.Name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		if wf != nil {
			event.PublishWorkflowUpdate(ctx, proj.Key, *wrkflw, *wf, u)
		} else {
			event.PublishWorkflowAdd(ctx, proj.Key, *wrkflw, u)
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
		proj, errp := project.Load(api.mustDB(), api.Cache, key,
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

		u := getAPIConsumer(ctx)

		wf, err := workflow.Load(ctx, api.mustDB(), api.Cache, proj, wfName, workflow.LoadOptions{WithIcon: true})
		if err != nil {
			return sdk.WrapError(err, "Unable to load workflow")
		}

		// if workflow is as-code, we can't save it from edit as yml
		if wf.FromRepository != "" {
			return sdk.WithStack(sdk.ErrForbidden)
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

		wrkflw, msgList, globalError := workflow.ParseAndImport(ctx, tx, api.Cache, proj, wf, ew, u, workflow.ImportOptions{Force: true, WorkflowName: wfName})
		msgListString := translate(r, msgList)
		if globalError != nil {
			if len(msgListString) != 0 {
				sdkErr := sdk.ExtractHTTPError(globalError, r.Header.Get("Accept-Language"))
				return service.WriteJSON(w, append(msgListString, sdkErr.Message), sdkErr.Status)
			}
			return sdk.WrapError(globalError, "Unable to import workflow %s", ew.Name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		if wf != nil {
			event.PublishWorkflowUpdate(ctx, key, *wrkflw, *wf, u)
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
			return sdk.WithStack(sdk.ErrWrongRequest)
		}

		btes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Error(ctx, "postWorkflowPushHandler> Unable to read body: %v", err)
			return sdk.WithStack(sdk.ErrWrongRequest)
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

		u := getAPIConsumer(ctx)

		//Load project
		proj, errp := project.Load(db, api.Cache, key,
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithApplications,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithIntegrations)
		if errp != nil {
			return sdk.WrapError(errp, "postWorkflowPushHandler> Cannot load project %s", key)
		}

		allMsg, wrkflw, oldWrkflw, err := workflow.Push(ctx, db, api.Cache, proj, tr, pushOptions, u, project.DecryptWithBuiltinKey)
		if err != nil {
			return err
		}
		msgListString := translate(r, allMsg)

		if wrkflw != nil {
			w.Header().Add(sdk.ResponseWorkflowIDHeader, fmt.Sprintf("%d", wrkflw.ID))
			w.Header().Add(sdk.ResponseWorkflowNameHeader, wrkflw.Name)
		}

		if oldWrkflw != nil {
			event.PublishWorkflowUpdate(ctx, proj.Key, *wrkflw, *oldWrkflw, u)
		} else {
			event.PublishWorkflowAdd(ctx, proj.Key, *wrkflw, u)
		}

		if wrkflw.Template != nil {
			if err := workflowtemplate.SetTemplateData(ctx, api.mustDB(), proj, wrkflw, getAPIConsumer(ctx), wrkflw.Template); err != nil {
				log.Error(ctx, "postTemplateApplyHandler> unable to set template data: %v", err)
			}
		}

		return service.WriteJSON(w, msgListString, http.StatusOK)
	}
}
