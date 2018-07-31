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
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) postWorkflowPreviewHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

		//Load project
		proj, errp := project.Load(api.mustDB(), api.Cache, key, getUser(ctx),
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithApplications,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithPlatforms,
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

		wf, globalError := workflow.Parse(proj, ew)
		if globalError != nil {
			return sdk.WrapError(globalError, "postWorkflowPreviewHandler> Unable import workflow %s", ew.Name)
		}

		return WriteJSON(w, wf, http.StatusOK)
	}
}

func (api *API) postWorkflowImportHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["permProjectKey"]
		force := FormBool(r, "force")

		//Load project
		proj, errp := project.Load(api.mustDB(), api.Cache, key, getUser(ctx),
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithApplications,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithPlatforms,
		)
		if errp != nil {
			return sdk.WrapError(errp, "postWorkflowImportHandler>> Unable load project")
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

		tx, errtx := api.mustDB().Begin()
		if errtx != nil {
			return sdk.WrapError(errtx, "postWorkflowImportHandler> Unable to start tx")
		}
		defer tx.Rollback()

		wrkflw, msgList, globalError := workflow.ParseAndImport(ctx, tx, api.Cache, proj, ew, getUser(ctx), workflow.ImportOptions{DryRun: false, Force: force})
		msgListString := translate(r, msgList)

		if globalError != nil {
			return sdk.WrapError(globalError, "postWorkflowImportHandler> Unable import workflow %s", ew.Name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "postWorkflowImportHandler> Cannot commit transaction")
		}

		if wrkflw != nil {
			w.Header().Add(sdk.ResponseWorkflowIDHeader, fmt.Sprintf("%d", wrkflw.ID))
			w.Header().Add(sdk.ResponseWorkflowNameHeader, wrkflw.Name)
		}

		return WriteJSON(w, msgListString, http.StatusOK)
	}
}

func (api *API) putWorkflowImportHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		wfName := vars["permWorkflowName"]

		//Load project
		proj, errp := project.Load(api.mustDB(), api.Cache, key, getUser(ctx),
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithApplications,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithPlatforms,
		)
		if errp != nil {
			return sdk.WrapError(errp, "postWorkflowImportHandler>> Unable load project")
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

		tx, errtx := api.mustDB().Begin()
		if errtx != nil {
			return sdk.WrapError(errtx, "postWorkflowImportHandler> Unable to start tx")
		}
		defer func() {
			_ = tx.Rollback()
		}()

		wrkflw, msgList, globalError := workflow.ParseAndImport(ctx, tx, api.Cache, proj, ew, getUser(ctx), workflow.ImportOptions{DryRun: false, Force: true, WorkflowName: wfName})
		msgListString := translate(r, msgList)

		if globalError != nil {
			return sdk.WrapError(globalError, "postWorkflowImportHandler> Unable import workflow %s", ew.Name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "postWorkflowImportHandler> Cannot commit transaction")
		}

		oldW, errL := workflow.Load(ctx, api.mustDB(), api.Cache, proj, wfName, getUser(ctx), workflow.LoadOptions{WithoutNode: true})
		if errL == nil {
			event.PublishWorkflowUpdate(key, *wrkflw, *oldW, getUser(ctx))
		}

		if wrkflw != nil {
			w.Header().Add(sdk.ResponseWorkflowIDHeader, fmt.Sprintf("%d", wrkflw.ID))
			w.Header().Add(sdk.ResponseWorkflowNameHeader, wrkflw.Name)
		}

		return WriteJSON(w, msgListString, http.StatusOK)
	}
}

func (api *API) postWorkflowPushHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		db := api.mustDB()
		vars := mux.Vars(r)
		key := vars["permProjectKey"]

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
				FromRepository: r.Header.Get(sdk.WorkflowAsCodeHeader),
			}
		}

		//Load project
		proj, errp := project.Load(db, api.Cache, key, getUser(ctx),
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithApplications,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithPlatforms)
		if errp != nil {
			return sdk.WrapError(errp, "postWorkflowPushHandler> Cannot load project %s", key)
		}

		allMsg, wrkflw, err := workflow.Push(ctx, db, api.Cache, proj, tr, pushOptions, getUser(ctx), project.DecryptWithBuiltinKey)
		if err != nil {
			return sdk.WrapError(err, "postWorkflowPushHandler> Cannot push workflow")
		}
		msgListString := translate(r, allMsg)

		if wrkflw != nil {
			w.Header().Add(sdk.ResponseWorkflowIDHeader, fmt.Sprintf("%d", wrkflw.ID))
			w.Header().Add(sdk.ResponseWorkflowNameHeader, wrkflw.Name)
		}

		return WriteJSON(w, msgListString, http.StatusOK)
	}
}
