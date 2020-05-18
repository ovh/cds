package api

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
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
		vars := mux.Vars(r)
		key := vars[permProjectKey]

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return sdk.NewError(sdk.ErrWrongRequest, err)
		}
		defer r.Body.Close()

		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(body)
		}
		format, err := exportentities.GetFormatFromContentType(contentType)
		if err != nil {
			return err
		}

		proj, err := project.Load(api.mustDB(), key,
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithApplications,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithIntegrations,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
		)
		if err != nil {
			return sdk.WrapError(err, "unable load project")
		}

		ew, errw := exportentities.UnmarshalWorkflow(body, format)
		if errw != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errw)
		}

		wf, err := workflow.Parse(ctx, *proj, ew)
		if err != nil {
			return sdk.WrapError(err, "unable import workflow %s", ew.GetName())
		}

		// Browse all node to find IDs
		if err := workflow.IsValid(ctx, api.Cache, api.mustDB(), wf, *proj, workflow.LoadOptions{}); err != nil {
			return sdk.WrapError(err, "workflow is not valid")
		}

		if err := workflow.RenameNode(ctx, api.mustDB(), wf); err != nil {
			return sdk.WrapError(err, "unable to rename node")
		}

		return service.WriteJSON(w, wf, http.StatusOK)
	}
}

func (api *API) postWorkflowImportHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		force := FormBool(r, "force")

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "can't "))
		}
		defer r.Body.Close()

		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(body)
		}
		format, err := exportentities.GetFormatFromContentType(contentType)
		if err != nil {
			return err
		}

		ew, err := exportentities.UnmarshalWorkflow(body, format)
		if err != nil {
			return err
		}

		proj, err := project.Load(api.mustDB(), key,
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithApplications,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithIntegrations,
		)
		if err != nil {
			return sdk.WrapError(err, "unable load project")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "unable to start transaction")
		}
		defer tx.Rollback() // nolint

		u := getAPIConsumer(ctx)

		// load the workflow from database if exists
		workflowExists, err := workflow.Exists(tx, proj.Key, ew.GetName())
		if err != nil {
			return sdk.WrapError(err, "Cannot check if workflow exists")
		}
		var wf *sdk.Workflow
		if workflowExists {
			wf, err = workflow.Load(ctx, tx, api.Cache, *proj, ew.GetName(), workflow.LoadOptions{WithIcon: true})
			if err != nil {
				return sdk.WrapError(err, "unable to load existing workflow")
			}
		}

		wrkflw, msgList, globalError := workflow.ParseAndImport(ctx, tx, api.Cache, *proj, wf, ew, getAPIConsumer(ctx), workflow.ImportOptions{Force: force})
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
			return sdk.WrapError(globalError, "Unable to import workflow %s", ew.GetName())
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
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

		body, errr := ioutil.ReadAll(r.Body)
		if errr != nil {
			return sdk.NewError(sdk.ErrWrongRequest, errr)
		}
		defer r.Body.Close()

		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			contentType = http.DetectContentType(body)
		}

		format, err := exportentities.GetFormatFromContentType(contentType)
		if err != nil {
			return err
		}

		ew, err := exportentities.UnmarshalWorkflow(body, format)
		if err != nil {
			return err
		}

		// Load project
		proj, err := project.Load(api.mustDB(), key,
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithApplications,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithIntegrations,
		)
		if err != nil {
			return sdk.WrapError(err, "unable load project")
		}

		u := getAPIConsumer(ctx)

		wf, err := workflow.Load(ctx, api.mustDB(), api.Cache, *proj, wfName, workflow.LoadOptions{WithIcon: true})
		if err != nil {
			return sdk.WrapError(err, "unable to load workflow")
		}

		// if workflow is as-code, we can't save it from edit as yml
		if wf.FromRepository != "" {
			return sdk.NewErrorFrom(sdk.ErrForbidden, "can't edit a workflow that is ascode")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "unable to start transaction")
		}
		defer tx.Rollback() //nolint

		wrkflw, msgList, globalError := workflow.ParseAndImport(ctx, tx, api.Cache, *proj, wf, ew, u, workflow.ImportOptions{Force: true, WorkflowName: wfName})
		msgListString := translate(r, msgList)
		if globalError != nil {
			if len(msgListString) != 0 {
				sdkErr := sdk.ExtractHTTPError(globalError, r.Header.Get("Accept-Language"))
				return service.WriteJSON(w, append(msgListString, sdkErr.Error()), sdkErr.Status)
			}
			return sdk.WrapError(globalError, "unable to import workflow %s", ew.GetName())
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
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
			return sdk.NewErrorWithStack(err, sdk.ErrWrongRequest)
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
		proj, err := project.Load(db, key,
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithApplications,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithIntegrations,
			project.LoadOptions.WithKeys,
		)
		if err != nil {
			return sdk.WrapError(err, "cannot load project %s", key)
		}

		data, err := exportentities.UntarWorkflowComponents(ctx, tr)
		if err != nil {
			return err
		}

		consumer := getAPIConsumer(ctx)

		mods := []workflowtemplate.TemplateRequestModifierFunc{
			workflowtemplate.TemplateRequestModifiers.DefaultKeys(*proj),
		}
		if pushOptions != nil && pushOptions.FromRepository != "" {
			mods = append(mods, workflowtemplate.TemplateRequestModifiers.DefaultNameAndRepositories(ctx, api.mustDB(), api.Cache, *proj, pushOptions.FromRepository))
		}
		var allMsg []sdk.Message
		msgTemplate, wti, err := workflowtemplate.CheckAndExecuteTemplate(ctx, api.mustDB(), *consumer, *proj, &data, mods...)
		allMsg = append(allMsg, msgTemplate...)
		if err != nil {
			return err
		}
		msgPush, wrkflw, oldWrkflw, err := workflow.Push(ctx, db, api.Cache, proj, data, pushOptions, u, project.DecryptWithBuiltinKey)
		allMsg = append(allMsg, msgPush...)
		if err != nil {
			return err
		}
		if err := workflowtemplate.UpdateTemplateInstanceWithWorkflow(ctx, api.mustDB(), *wrkflw, *consumer, wti); err != nil {
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

		return service.WriteJSON(w, msgListString, http.StatusOK)
	}
}
