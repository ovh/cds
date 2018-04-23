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
		proj, errp := project.Load(api.mustDB(ctx), api.Cache, key, getUser(ctx),
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithApplications,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithPipelines,
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
		proj, errp := project.Load(api.mustDB(ctx), api.Cache, key, getUser(ctx),
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithApplications,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithPipelines,
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

		tx, errtx := api.mustDB(ctx).Begin()
		if errtx != nil {
			return sdk.WrapError(errtx, "postWorkflowImportHandler> Unable to start tx")
		}
		defer tx.Rollback()

		wrkflw, msgList, globalError := workflow.ParseAndImport(tx, api.Cache, proj, ew, force, getUser(ctx), false)
		msgListString := translate(r, msgList)

		if globalError != nil {
			return sdk.WrapError(globalError, "postWorkflowImportHandler> Unable import workflow %s", ew.Name)
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), proj, sdk.ProjectWorkflowLastModificationType); err != nil {
			return sdk.WrapError(err, "postWorkflowImportHandler> Unable to update project")
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

func (api *API) postWorkflowPushHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
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
		proj, errp := project.Load(api.mustDB(ctx), api.Cache, key, getUser(ctx),
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithApplications,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithPipelines)
		if errp != nil {
			return sdk.WrapError(errp, "postWorkflowPushHandler> Cannot load project %s", key)
		}

		allMsg, wrkflw, err := workflow.Push(api.mustDB(ctx), api.Cache, proj, tr, pushOptions, getUser(ctx), project.DecryptWithBuiltinKey)
		if err != nil {
			return sdk.WrapError(err, "postWorkflowPushHandler> Cannot push workflow")
		}
		msgListString := translate(r, allMsg)

		if err := project.UpdateLastModified(api.mustDB(ctx), api.Cache, getUser(ctx), proj, sdk.ProjectPipelineLastModificationType); err != nil {
			return sdk.WrapError(err, "postWorkflowPushHandler> Unable to update project")
		}

		if wrkflw != nil {
			w.Header().Add(sdk.ResponseWorkflowIDHeader, fmt.Sprintf("%d", wrkflw.ID))
			w.Header().Add(sdk.ResponseWorkflowNameHeader, wrkflw.Name)
		}

		return WriteJSON(w, msgListString, http.StatusOK)
	}
}
