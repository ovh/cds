package api

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

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
		case "application/x-yaml", "text/x-yam":
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

		msgList, globalError := workflow.ParseAndImport(tx, api.Cache, proj, ew, force, getUser(ctx))
		msgListString := translate(r, msgList)

		if globalError != nil {
			myError, ok := globalError.(sdk.Error)
			if ok {
				return WriteJSON(w, r, msgListString, myError.Status)
			}
			return sdk.WrapError(globalError, "postWorkflowImportHandler> Unable import workflow %s", ew.Name)
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), proj, sdk.ProjectPipelineLastModificationType); err != nil {
			return sdk.WrapError(err, "postWorkflowImportHandler> Unable to update project")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "postWorkflowImportHandler> Cannot commit transaction")
		}

		return WriteJSON(w, r, msgListString, http.StatusOK)
	}
}

func (api *API) postWorkflowPushHandler() Handler {
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
		)
		if errp != nil {
			return sdk.WrapError(errp, "postWorkflowPushHandler>> Unable load project")
		}

		if r.Body == nil {
			return sdk.ErrWrongRequest
		}

		tr := tar.NewReader(r.Body)
		defer r.Body.Close()

		apps := make(map[string]exportentities.Application)
		pips := make(map[string]exportentities.PipelineV1)
		envs := make(map[string]exportentities.Environment)
		var wrkflw exportentities.Workflow

		mError := new(sdk.MultiError)
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				err = sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("Unable to read tar file"))
				return sdk.WrapError(err, "postWorkflowPushHandler>")
			}

			log.Debug("postWorkflowPushHandler> Reading %s", hdr.Name)

			buff := new(bytes.Buffer)
			if _, err := io.Copy(buff, tr); err != nil {
				err = sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("Unable to read tar file"))
				return sdk.WrapError(err, "postWorkflowPushHandler>")
			}

			switch {
			case strings.Contains(hdr.Name, ".app."):
				var app exportentities.Application
				if err := yaml.Unmarshal(buff.Bytes(), &app); err != nil {
					mError.Append(fmt.Errorf("Unable to load application %s: %v", hdr.Name, err))
					continue
				}
				apps[hdr.Name] = app
			case strings.Contains(hdr.Name, ".pip."):
				var pip exportentities.PipelineV1
				if err := yaml.Unmarshal(buff.Bytes(), &pip); err != nil {
					mError.Append(fmt.Errorf("Unable to load pipeline %s: %v", hdr.Name, err))
					continue
				}
				pips[hdr.Name] = pip
			case strings.Contains(hdr.Name, ".env."):
				var env exportentities.Environment
				if err := yaml.Unmarshal(buff.Bytes(), &w); err != nil {
					mError.Append(fmt.Errorf("Unable to load environment %s: %v", hdr.Name, err))
					continue
				}
				envs[hdr.Name] = env
			default:
				if err := yaml.Unmarshal(buff.Bytes(), &wrkflw); err != nil {
					mError.Append(fmt.Errorf("Unable to load workflow %s: %v", hdr.Name, err))
					continue
				}
			}
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "postWorkflowPushHandler> Unable to start tx")
		}

		allMsg := []sdk.Message{}
		for filename, app := range apps {
			log.Debug("postWorkflowPullHandler> Parsing %s", filename)
			msgList, err := application.ParseAndImport(tx, api.Cache, proj, &app, true, project.DecryptWithBuiltinKey, getUser(ctx))
			if err != nil {
				mError.Append(err)
			}
			allMsg = append(allMsg, msgList...)
			log.Debug("postWorkflowPushHandler> -- %s OK", filename)
		}

		for filename, app := range envs {
			log.Debug("postWorkflowPushHandler> Parsing %s", filename)
			msgList, err := environment.ParseAndImport(tx, api.Cache, proj, &app, true, project.DecryptWithBuiltinKey, getUser(ctx))
			if err != nil {
				mError.Append(err)
			}
			allMsg = append(allMsg, msgList...)
			log.Debug("postWorkflowPushHandler> -- %s OK", filename)
		}

		for filename, pip := range pips {
			log.Debug("postWorkflowPushHandler> Parsing %s", filename)
			msgList, err := pipeline.ParseAndImport(tx, api.Cache, proj, &pip, true, getUser(ctx))
			if err != nil {
				mError.Append(err)
			}
			allMsg = append(allMsg, msgList...)
			log.Debug("postWorkflowPushHandler> -- %s OK", filename)
		}

		msgList, err := workflow.ParseAndImport(tx, api.Cache, proj, &wrkflw, force, getUser(ctx))
		if err != nil {
			mError.Append(err)
		}
		allMsg = append(allMsg, msgList...)

		msgListString := translate(r, allMsg)

		if !mError.IsEmpty() {
			return mError
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), proj, sdk.ProjectPipelineLastModificationType); err != nil {
			return sdk.WrapError(err, "postWorkflowPushHandler> Unable to update project")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "postWorkflowPushHandler> Cannot commit transaction")
		}

		return WriteJSON(w, r, msgListString, http.StatusOK)
	}
}
