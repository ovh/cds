package api

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/telemetry"
)

func (api *API) postWorkflowPreviewHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]

		body, err := io.ReadAll(r.Body)
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

		proj, err := project.Load(ctx, api.mustDB(), key,
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

		// load the workflow from database if exists
		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "unable to start transaction")
		}
		defer tx.Rollback() // nolint

		workflowExists, err := workflow.Exists(ctx, tx, proj.Key, ew.GetName())
		if err != nil {
			return sdk.WrapError(err, "Cannot check if workflow exists")
		}
		var existingWorkflow *sdk.Workflow
		if workflowExists {
			existingWorkflow, err = workflow.Load(ctx, tx, api.Cache, *proj, ew.GetName(), workflow.LoadOptions{WithIcon: true})
			if err != nil {
				return sdk.WrapError(err, "unable to load existing workflow")
			}
		}

		wf, err := workflow.Parse(ctx, *proj, existingWorkflow, ew)
		if err != nil {
			return sdk.WrapError(err, "unable import workflow %s", ew.GetName())
		}

		if err := workflow.CompleteWorkflow(ctx, api.mustDB(), wf, *proj, workflow.LoadOptions{}); err != nil {
			return err
		}

		if err := workflow.CheckValidity(ctx, api.mustDB(), wf); err != nil {
			return err
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
		force := service.FormBool(r, "force")

		body, err := io.ReadAll(r.Body)
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

		proj, err := project.Load(ctx, api.mustDB(), key,
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

		u := getUserConsumer(ctx)

		// load the workflow from database if exists
		workflowExists, err := workflow.Exists(ctx, tx, proj.Key, ew.GetName())
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

		wrkflw, msgList, globalError := workflow.ParseAndImport(ctx, tx, api.Cache, *proj, wf, ew, getUserConsumer(ctx), workflow.ImportOptions{Force: force})
		msgListString := translate(msgList)
		if globalError != nil {
			if len(msgListString) != 0 {
				sdkErr := sdk.ExtractHTTPError(globalError)
				return service.WriteJSON(w, append(msgListString, sdkErr.Error()), sdkErr.Status)
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

		body, errr := io.ReadAll(r.Body)
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
		proj, err := project.Load(ctx, api.mustDB(), key,
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithIntegrations,
		)
		if err != nil {
			return sdk.WrapError(err, "unable load project")
		}

		u := getUserConsumer(ctx)

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
		msgListString := translate(msgList)
		if globalError != nil {
			if len(msgListString) != 0 {
				sdkErr := sdk.ExtractHTTPError(globalError)
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

		telemetry.Current(ctx,
			telemetry.Tag(telemetry.TagProjectKey, key),
		)

		if r.Body == nil {
			return sdk.WithStack(sdk.ErrWrongRequest)
		}

		btes, err := io.ReadAll(r.Body)
		if err != nil {
			return sdk.NewErrorWithStack(err, sdk.ErrWrongRequest)
		}
		defer r.Body.Close()

		log.Debug(ctx, "Read %d bytes from body", len(btes))
		tr := tar.NewReader(bytes.NewReader(btes))

		consumer := getUserConsumer(ctx)

		pushOptions := &workflow.PushOption{}
		if r.Header.Get(sdk.WorkflowAsCodeHeader) != "" {
			pushOptions.FromRepository = r.Header.Get(sdk.WorkflowAsCodeHeader)
			pushOptions.IsDefaultBranch = true
		}
		if service.FormBool(r, "force") {
			pushOptions.Force = true
		}

		//Load project
		proj, err := project.Load(ctx, db, key,
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

		mods := []workflowtemplate.TemplateRequestModifierFunc{
			workflowtemplate.TemplateRequestModifiers.DefaultKeys(*proj),
		}
		if pushOptions.FromRepository != "" {
			mods = append(mods, workflowtemplate.TemplateRequestModifiers.DefaultNameAndRepositories(*proj, pushOptions.FromRepository))
		}
		var allMsg []sdk.Message
		msgTemplate, wti, err := workflowtemplate.CheckAndExecuteTemplate(ctx, api.mustDB(), api.Cache, *consumer, *proj, &data, mods...)
		allMsg = append(allMsg, msgTemplate...)
		if err != nil {
			return err
		}
		msgPush, wrkflw, oldWrkflw, _, err := workflow.Push(ctx, db, api.Cache, proj, data, pushOptions, consumer, project.DecryptWithBuiltinKey, api.gpgKeyEmailAddress)
		allMsg = append(allMsg, msgPush...)
		if err != nil {
			return err
		}
		if err := workflowtemplate.UpdateTemplateInstanceWithWorkflow(ctx, api.mustDB(), *wrkflw, consumer, wti); err != nil {
			return err
		}

		msgListString := translate(allMsg)

		if wrkflw != nil {
			w.Header().Add(sdk.ResponseWorkflowIDHeader, fmt.Sprintf("%d", wrkflw.ID))
			w.Header().Add(sdk.ResponseWorkflowNameHeader, wrkflw.Name)
		}

		if oldWrkflw != nil {
			event.PublishWorkflowUpdate(ctx, proj.Key, *wrkflw, *oldWrkflw, consumer)
		} else {
			event.PublishWorkflowAdd(ctx, proj.Key, *wrkflw, consumer)
		}

		return service.WriteJSON(w, msgListString, http.StatusOK)
	}
}
