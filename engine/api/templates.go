package api

import (
	"bytes"
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/slug"
)

type contextKey int

const (
	contextWorkflowTemplate contextKey = iota
)

// TODO create real middleware
func (api *API) middlewareTemplate(needAdmin bool) func(ctx context.Context, w http.ResponseWriter, r *http.Request) (context.Context, error) {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) (context.Context, error) {
		// try to get template for given path that match user's groups with admin grants
		vars := mux.Vars(r)

		groupName := vars["groupName"]
		templateSlug := vars["templateSlug"]

		if groupName == "" || templateSlug == "" || !slug.Valid(templateSlug) {
			return nil, sdk.WrapError(sdk.ErrWrongRequest, "Invalid given group or template slug")
		}

		u := getUser(ctx)

		var group *sdk.Group
		for _, g := range u.Groups {
			if g.Name == groupName {
				group = &g
				break
			}
		}
		if group == nil {
			return nil, sdk.WrapError(sdk.ErrNotFound, "Invalid given group name")
		}

		if needAdmin {
			var isAdmin bool
			for _, a := range group.Admins {
				if a.ID == u.ID {
					isAdmin = true
					break
				}
			}
			if !isAdmin {
				return nil, sdk.WithStack(sdk.ErrInvalidGroupAdmin)
			}
		}

		wt, err := workflowtemplate.Get(api.mustDB(), workflowtemplate.NewCriteria().
			Slugs(templateSlug).GroupIDs(group.ID))
		if err != nil {
			return nil, err
		}
		if wt == nil {
			return nil, sdk.WithStack(sdk.ErrNotFound)
		}

		return context.WithValue(ctx, contextWorkflowTemplate, wt), nil
	}
}

func getWorkflowTemplate(c context.Context) *sdk.WorkflowTemplate {
	i := c.Value(contextWorkflowTemplate)
	if i == nil {
		return nil
	}
	wt, ok := i.(*sdk.WorkflowTemplate)
	if !ok {
		return nil
	}
	return wt
}

func (api *API) getTemplatesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		u := getUser(ctx)

		ts, err := workflowtemplate.GetAll(api.mustDB(), workflowtemplate.NewCriteria().
			GroupIDs(sdk.GroupsToIDs(u.Groups)...))
		if err != nil {
			return err
		}

		if err := group.AggregateOnWorkflowTemplate(api.mustDB(), ts...); err != nil {
			return err
		}
		if err := workflowtemplate.AggregateAuditsOnWorkflowTemplate(api.mustDB(), ts...); err != nil {
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

		t.Slug = slug.Convert(t.Name)

		if err := workflowtemplate.Insert(api.mustDB(), &t); err != nil {
			return err
		}

		event.PublishWorkflowTemplateAdd(t, u)

		if err := group.AggregateOnWorkflowTemplate(api.mustDB(), &t); err != nil {
			return err
		}
		if err := workflowtemplate.AggregateAuditsOnWorkflowTemplate(api.mustDB(), &t); err != nil {
			return err
		}

		return service.WriteJSON(w, t, http.StatusOK)
	}
}

func (api *API) getTemplateHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		ctx, err := api.middlewareTemplate(true)(ctx, w, r)
		if err != nil {
			return err
		}

		t := getWorkflowTemplate(ctx)

		if err := group.AggregateOnWorkflowTemplate(api.mustDB(), t); err != nil {
			return err
		}
		if err := workflowtemplate.AggregateAuditsOnWorkflowTemplate(api.mustDB(), t); err != nil {
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

		var err error
		ctx, err = api.middlewareTemplate(true)(ctx, w, r)
		if err != nil {
			return err
		}

		old := getWorkflowTemplate(ctx)
		u := getUser(ctx)

		// if group id has changed check that user is admin for new group id
		if old.GroupID != data.GroupID {
			var isAdminForGroup bool
			for _, g := range u.Groups {
				if g.ID == data.GroupID {
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
		}

		// update fields from request data
		new := sdk.WorkflowTemplate(*old)
		new.Name = data.Name
		new.Slug = slug.Convert(data.Name)
		new.GroupID = data.GroupID
		new.Description = data.Description
		new.Value = data.Value
		new.Parameters = data.Parameters
		new.Pipelines = data.Pipelines
		new.Applications = data.Applications
		new.Version = old.Version + 1

		if err := workflowtemplate.Update(api.mustDB(), &new); err != nil {
			return err
		}

		event.PublishWorkflowTemplateUpdate(*old, new, u)

		if err := group.AggregateOnWorkflowTemplate(api.mustDB(), &new); err != nil {
			return err
		}
		if err := workflowtemplate.AggregateAuditsOnWorkflowTemplate(api.mustDB(), &new); err != nil {
			return err
		}

		return service.WriteJSON(w, new, http.StatusOK)
	}
}

func (api *API) deleteTemplateHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		ctx, err := api.middlewareTemplate(true)(ctx, w, r)
		if err != nil {
			return err
		}

		wt := getWorkflowTemplate(ctx)

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Cannot start transaction")
		}
		defer func() { _ = tx.Rollback() }()

		if err := workflowtemplate.DeleteInstancesForWorkflowTemplateID(tx, wt.ID); err != nil {
			return err
		}

		if err := workflowtemplate.Delete(tx, wt); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishWorkflowTemplateDelete(*wt, getUser(ctx))

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) applyTemplateHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		ctx, err := api.middlewareTemplate(false)(ctx, w, r)
		if err != nil {
			return err
		}
		t := getWorkflowTemplate(ctx)

		// parse and check request
		var req sdk.WorkflowTemplateRequest
		if err := service.UnmarshalBody(r, &req); err != nil {
			return err
		}
		if err := t.CheckParams(req); err != nil {
			return sdk.NewError(sdk.ErrInvalidData, err)
		}

		// check right on project
		if !checkProjectReadPermission(ctx, req.ProjectKey) {
			return sdk.WithStack(sdk.ErrNoProject)
		}

		// load project with key
		p, err := project.Load(api.mustDB(), api.Cache, req.ProjectKey, getUser(ctx))
		if err != nil {
			return sdk.NewError(sdk.ErrNoProject, err)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Cannot start transaction")
		}
		defer func() { _ = tx.Rollback() }()

		i := &sdk.WorkflowTemplateInstance{
			ProjectID:               p.ID,
			WorkflowTemplateID:      t.ID,
			WorkflowTemplateVersion: t.Version,
			Request:                 req,
		}

		// create new instance for template
		if err := workflowtemplate.InsertInstance(tx, i); err != nil {
			return err
		}

		// execute template with request
		res, err := workflowtemplate.Execute(t, i)
		if err != nil {
			return err
		}

		buf := new(bytes.Buffer)
		if err := workflowtemplate.Tar(res, buf); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		return service.Write(w, buf.Bytes(), http.StatusOK, "application/tar")
	}
}

func (api *API) getTemplateInstancesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		ctx, err := api.middlewareTemplate(true)(ctx, w, r)
		if err != nil {
			return err
		}
		t := getWorkflowTemplate(ctx)

		is, err := workflowtemplate.GetInstances(api.mustDB(), workflowtemplate.NewCriteriaInstance().WorkflowTemplateIDs(t.ID))
		if err != nil {
			return err
		}

		return service.WriteJSON(w, is, http.StatusOK)
	}
}

// TODO refact merge with apply
/*func (api *API) updateWorkflowTemplateHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		key := vars["key"]
		workflowName := vars["permWorkflowName"]

		proj, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.WithPlatforms)
		if err != nil {
			return sdk.WrapError(err, "Unable to load projet")
		}

		wf, err := workflow.Load(ctx, api.mustDB(), api.Cache, proj, workflowName, getUser(ctx), workflow.LoadOptions{})
		if err != nil {
			return sdk.WrapError(err, "Cannot load workflow %s", workflowName)
		}

		// check if workflow is a generated one
		wti, err := workflowtemplate.GetInstance(api.mustDB(), workflowtemplate.NewCriteriaInstance().WorkflowIDs(wf.ID))
		if err != nil {
			return err
		}
		if wti == nil {
			return sdk.WithStack(sdk.ErrWorkflowNotGenerated)
		}

		// check if the workflow need an update
		wt, err := workflowtemplate.Get(api.mustDB(), workflowtemplate.NewCriteria().IDs(wti.WorkflowTemplateID))
		if err != nil {
			return err
		}

		// get new params but override name
		var req sdk.WorkflowTemplateRequest
		if err := service.UnmarshalBody(r, &req); err != nil {
			return err
		}
		req.Name = wti.Request.Name
		if err := wt.CheckParams(req); err != nil {
			return sdk.NewError(sdk.ErrInvalidData, err)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Cannot start transaction")
		}
		defer func() { _ = tx.Rollback() }()

		wti.WorkflowTemplateVersion = wt.Version
		wti.Request = req

		// create new instance for template
		if err := workflowtemplate.UpdateInstance(tx, wti); err != nil {
			return err
		}

		// execute template with request
		res, err := workflowtemplate.Execute(wt, wti)
		if err != nil {
			return err
		}

		buf := new(bytes.Buffer)
		if err := workflowtemplate.Tar(res, buf); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		return service.Write(w, buf.Bytes(), http.StatusOK, "application/tar")
	}
}*/

func (api *API) getTemplateInstanceHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		ctx, err := api.middlewareTemplate(false)(ctx, w, r)
		if err != nil {
			return err
		}
		t := getWorkflowTemplate(ctx)

		vars := mux.Vars(r)

		key := vars["key"]
		workflowName := vars["permWorkflowName"]

		proj, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.WithPlatforms)
		if err != nil {
			return sdk.WrapError(err, "Unable to load projet")
		}

		wf, err := workflow.Load(ctx, api.mustDB(), api.Cache, proj, workflowName, getUser(ctx), workflow.LoadOptions{})
		if err != nil {
			return sdk.WrapError(err, "Cannot load workflow %s", workflowName)
		}

		// return the template instance if workflow is a generated one
		wti, err := workflowtemplate.GetInstance(api.mustDB(), workflowtemplate.NewCriteriaInstance().
			WorkflowIDs(wf.ID).WorkflowTemplateIDs(t.ID))
		if err != nil {
			return err
		}
		if wti == nil {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		return service.WriteJSON(w, wti, http.StatusOK)
	}
}
