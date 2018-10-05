package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func (api *API) getTemplatesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// TODO implement group owner for templates
		ts, err := workflowtemplate.GetAll(api.mustDB())
		if err != nil {
			return err
		}

		return service.WriteJSON(w, ts, http.StatusOK)
	}
}

func (api *API) postTemplateHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		t := &sdk.WorkflowTemplate{}
		if err := UnmarshalBody(r, t); err != nil {
			return sdk.WrapError(err, "Unmarshall body error")
		}

		if err := t.ValidateStruct(); err != nil {
			return err
		}

		// TODO implement group owner for templates

		if err := workflowtemplate.InsertWorkflow(api.mustDB(), t); err != nil {
			return err
		}

		return service.WriteJSON(w, t, http.StatusOK)
	}
}

func (api *API) executeTemplateHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]

		// load project for given key
		proj, err := project.Load(api.mustDB(), api.Cache, projectKey, getUser(ctx),
			project.LoadOptions.Default, project.LoadOptions.WithGroups)
		if err != nil {
			return sdk.WrapError(err, "Unable to load project %s", projectKey)
		}

		id, err := requestVarInt(r, "id")
		if err != nil {
			return sdk.ErrNotFound
		}

		// TODO implement group owner for templates

		t, err := workflowtemplate.GetByID(api.mustDB(), id)
		if err != nil {
			return err
		}
		if t == nil {
			return sdk.ErrNotFound
		}

		var req sdk.WorkflowTemplateRequest
		if err := UnmarshalBody(r, &req); err != nil {
			return sdk.WrapError(err, "Unmarshall body error")
		}

		if err := t.CheckParams(req); err != nil {
			return err
		}

		res, err := workflowtemplate.Execute(t, req)
		if err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "importPipelineHandler: Cannot start transaction")
		}
		defer tx.Rollback()

		var msgListString []string

		for _, p := range res.Pipelines {
			var pip exportentities.PipelineV1
			if err := yaml.Unmarshal([]byte(p), &pip); err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, sdk.WrapError(err, "Cannot parse generated pipeline template"))
			}

			_, msgs, err := pipeline.ParseAndImport(tx, api.Cache, proj, pip, getUser(ctx),
				pipeline.ImportOptions{Force: true})
			msgsT := translate(r, msgs)
			if err != nil {
				return sdk.WrapError(err, "Unable import generated pipeline")
			}

			msgListString = append(msgListString, msgsT...)
		}

		var wor exportentities.Workflow
		if err := yaml.Unmarshal([]byte(res.Workflow), &wor); err != nil {
			return sdk.NewError(sdk.ErrWrongRequest, sdk.WrapError(err, "Cannot parse generated workflow template"))
		}

		_, msgs, err := workflow.ParseAndImport(ctx, tx, api.Cache, proj, &wor, getUser(ctx),
			workflow.ImportOptions{DryRun: false, Force: true})
		if err != nil {
			return sdk.WrapError(err, "Unable import generated workflow")
		}

		msgListString = append(msgListString, translate(r, msgs)...)

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "importPipelineHandler> Cannot commit transaction")
		}

		return service.WriteJSON(w, msgListString, http.StatusOK)
	}
}
