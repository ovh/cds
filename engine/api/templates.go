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
		u := getUser(ctx)

		ts, err := workflowtemplate.GetAll(api.mustDB(), workflowtemplate.NewCriteria().
			GroupIDs(sdk.GroupsToIDs(u.Groups)...))
		if err != nil {
			return err
		}

		return service.WriteJSON(w, ts, http.StatusOK)
	}
}

func (api *API) postTemplateHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var t sdk.WorkflowTemplate
		if err := service.UnmarshalBody(r, &t); err != nil {
			return err
		}
		if err := t.ValidateStruct(); err != nil {
			return err
		}
		t.Version = 0

		u := getUser(ctx)

		var isAdminForGroup bool
		for _, g := range u.Groups {
			if g.ID == t.GroupID {
				for _, a := range g.Admins {
					if a.ID == u.ID {
						isAdminForGroup = true
						break
					}
				}
				break
			}
		}
		if !isAdminForGroup {
			return sdk.WithStack(sdk.ErrInvalidGroupAdmin)
		}

		if err := workflowtemplate.Insert(api.mustDB(), &t); err != nil {
			return err
		}

		return service.WriteJSON(w, t, http.StatusOK)
	}
}

func (api *API) putTemplateHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		data := sdk.WorkflowTemplate{}
		if err := service.UnmarshalBody(r, &data); err != nil {
			return err
		}
		if err := data.ValidateStruct(); err != nil {
			return err
		}

		// try to get template for given id that match user's groups with admin grants
		id, err := requestVarInt(r, "id")
		if err != nil {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		u := getUser(ctx)
		var userGroupsWithAdmin []sdk.Group
		for _, g := range u.Groups {
			for _, a := range g.Admins {
				if a.ID == u.ID {
					userGroupsWithAdmin = append(userGroupsWithAdmin, g)
					break
				}
			}
		}

		t, err := workflowtemplate.Get(api.mustDB(), workflowtemplate.NewCriteria().
			IDs(id).GroupIDs(sdk.GroupsToIDs(userGroupsWithAdmin)...))
		if err != nil {
			return err
		}
		if t == nil {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		// update fields
		t.Value = data.Value
		t.Parameters = data.Parameters
		t.Pipelines = data.Pipelines
		t.Version = t.Version + 1

		if err := workflowtemplate.Update(api.mustDB(), t); err != nil {
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

		// try to get template for given id that match user's groups
		id, err := requestVarInt(r, "id")
		if err != nil {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		u := getUser(ctx)

		t, err := workflowtemplate.Get(api.mustDB(), workflowtemplate.NewCriteria().
			IDs(id).GroupIDs(sdk.GroupsToIDs(u.Groups)...))
		if err != nil {
			return err
		}
		if t == nil {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		// parse and check request
		var req sdk.WorkflowTemplateRequest
		if err := service.UnmarshalBody(r, &req); err != nil {
			return err
		}
		if err := t.CheckParams(req); err != nil {
			return err
		}

		// execute template with request
		res, err := workflowtemplate.Execute(t, req)
		if err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "importPipelineHandler: Cannot start transaction")
		}
		defer func() { _ = tx.Rollback() }()

		// import generated pipelines
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

		// import generated workflow
		var wor exportentities.Workflow
		if err := yaml.Unmarshal([]byte(res.Workflow), &wor); err != nil {
			return sdk.NewError(sdk.ErrWrongRequest, sdk.WrapError(err, "Cannot parse generated workflow template"))
		}

		workflow, msgs, err := workflow.ParseAndImport(ctx, tx, api.Cache, proj, &wor, getUser(ctx),
			workflow.ImportOptions{DryRun: false, Force: true})
		if err != nil {
			return sdk.WrapError(err, "Unable import generated workflow")
		}

		msgListString = append(msgListString, translate(r, msgs)...)

		// remove existing relations between workflow and template
		if err := workflowtemplate.DeleteRelationsForWorkflowID(tx, workflow.ID); err != nil {
			return err
		}

		// create new relation between workflow and template
		if err := workflowtemplate.InsertRelation(tx, &sdk.WorkflowTemplateWorkflow{
			WorkflowTemplateID:      t.ID,
			WorkflowID:              workflow.ID,
			WorkflowTemplateVersion: t.Version,
			Request:                 req,
		}); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		return service.WriteJSON(w, msgListString, http.StatusOK)
	}
}
