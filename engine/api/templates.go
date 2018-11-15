package api

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/slug"
)

type contextKey int

const (
	contextWorkflowTemplate contextKey = iota
)

func (api *API) middlewareTemplate(needAdmin bool) func(ctx context.Context, w http.ResponseWriter, r *http.Request) (context.Context, error) {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) (context.Context, error) {
		// try to get template for given id or path that match user's groups with/without admin grants
		vars := mux.Vars(r)

		id, _ := requestVarInt(r, "id") // ignore error, will check if not 0
		groupName := vars["groupName"]
		templateSlug := vars["templateSlug"]

		if id == 0 && (groupName == "" || templateSlug == "" || !slug.Valid(templateSlug)) {
			return nil, sdk.WrapError(sdk.ErrWrongRequest, "Invalid given id or group and template slug")
		}

		u := getUser(ctx)

		var gs []sdk.Group
		if needAdmin {
			for _, g := range u.Groups {
				for _, a := range g.Admins {
					if a.ID == u.ID {
						gs = append(gs, g)
					}
				}
			}
		} else {
			gs = u.Groups
		}

		if groupName != "" {
			for i := range gs {
				if gs[i].Name == groupName {
					gs = []sdk.Group{gs[i]}
					break
				}
			}
			if len(gs) == 0 {
				return nil, sdk.WrapError(sdk.ErrNotFound, "Invalid given group name")
			}
		}

		cr := workflowtemplate.NewCriteria()
		if id != 0 {
			cr = cr.IDs(id)
		} else {
			cr = cr.Slugs(templateSlug)
		}

		wt, err := workflowtemplate.Get(api.mustDB(), cr.GroupIDs(sdk.GroupsToIDs(gs)...))
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

		// duplicate couple of group id and slug will failed with sql constraint
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
		ctx, err := api.middlewareTemplate(false)(ctx, w, r)
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
		new.Update(data)

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

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) applyTemplateHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		ctx, err := api.middlewareTemplate(false)(ctx, w, r)
		if err != nil {
			return err
		}
		wt := getWorkflowTemplate(ctx)
		if err := group.AggregateOnWorkflowTemplate(api.mustDB(), wt); err != nil {
			return err
		}

		// parse and check request
		var req sdk.WorkflowTemplateRequest
		if err := service.UnmarshalBody(r, &req); err != nil {
			return err
		}
		if err := wt.CheckParams(req); err != nil {
			return err
		}

		// check right on project
		if !checkProjectReadPermission(ctx, req.ProjectKey) {
			return sdk.WithStack(sdk.ErrNoProject)
		}

		u := getUser(ctx)

		// load project with key
		p, err := project.Load(api.mustDB(), api.Cache, req.ProjectKey, u)
		if err != nil {
			return err
		}

		// check if a workflow exists with given slug
		wf, err := workflow.Load(ctx, api.mustDB(), api.Cache, p, req.WorkflowName, u, workflow.LoadOptions{})
		if err != nil && !sdk.ErrorIs(err, sdk.ErrWorkflowNotFound) {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Cannot start transaction")
		}
		defer func() { _ = tx.Rollback() }()

		var wti *sdk.WorkflowTemplateInstance
		if wf != nil {
			// check if workflow is a generated one for the current template
			wti, err = workflowtemplate.GetInstance(tx, workflowtemplate.NewCriteriaInstance().
				WorkflowIDs(wf.ID).WorkflowTemplateIDs(wt.ID))
			if err != nil {
				return err
			}
		}
		if wti == nil {
			// try to get a instance not assign to a workflow but with the same slug
			wtis, err := workflowtemplate.GetInstances(tx, workflowtemplate.NewCriteriaInstance().
				WorkflowIDs(0).WorkflowTemplateIDs(wt.ID).ProjectIDs(p.ID))
			if err != nil {
				return err
			}

			for _, res := range wtis {
				if res.Request.WorkflowName == req.WorkflowName {
					wti = res
					break
				}
			}
		}

		// if a previous instance exist for the same workflow update it, else create a new one
		var old *sdk.WorkflowTemplateInstance
		if wti != nil {
			clone := sdk.WorkflowTemplateInstance(*wti)
			old = &clone
			req.WorkflowName = wti.Request.WorkflowName
			wti.WorkflowTemplateVersion = wt.Version
			wti.Request = req
			if err := workflowtemplate.UpdateInstance(tx, wti); err != nil {
				return err
			}
		} else {
			wti = &sdk.WorkflowTemplateInstance{
				ProjectID:               p.ID,
				WorkflowTemplateID:      wt.ID,
				WorkflowTemplateVersion: wt.Version,
				Request:                 req,
			}
			if err := workflowtemplate.InsertInstance(tx, wti); err != nil {
				return err
			}
		}

		// execute template with request
		res, err := workflowtemplate.Execute(wt, wti)
		if err != nil {
			return err
		}

		buf := new(bytes.Buffer)
		if err := workflowtemplate.Tar(wt, res, buf); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		if old != nil {
			event.PublishWorkflowTemplateInstanceUpdate(*old, *wti, u)
		} else {
			event.PublishWorkflowTemplateInstanceAdd(*wti, u)
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

		u := getUser(ctx)

		ps, err := project.LoadAll(ctx, api.mustDB(), api.Cache, u)
		if err != nil {
			return err
		}

		is, err := workflowtemplate.GetInstances(api.mustDB(), workflowtemplate.NewCriteriaInstance().
			WorkflowTemplateIDs(t.ID).ProjectIDs(sdk.ProjectsToIDs(ps)...))
		if err != nil {
			return err
		}

		mProjects := make(map[int64]sdk.Project, len(ps))
		for i := range ps {
			mProjects[ps[i].ID] = ps[i]
		}
		for _, i := range is {
			p := mProjects[i.ProjectID]
			i.Project = &p
		}
		if err := workflowtemplate.AggregateAuditsOnWorkflowTemplateInstance(api.mustDB(), is...); err != nil {
			return err
		}
		if err := workflow.AggregateOnWorkflowTemplateInstance(api.mustDB(), is...); err != nil {
			return err
		}

		return service.WriteJSON(w, is, http.StatusOK)
	}
}

func (api *API) getTemplateInstanceHandler() service.Handler {
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
			if sdk.ErrorIs(err, sdk.ErrWorkflowNotFound) {
				return sdk.NewErrorFrom(sdk.ErrNotFound, "Cannot load workflow %s", workflowName)
			}
			return sdk.WithStack(err)
		}
		// return the template instance if workflow is a generated one
		wti, err := workflowtemplate.GetInstance(api.mustDB(), workflowtemplate.NewCriteriaInstance().
			WorkflowIDs(wf.ID))
		if err != nil {
			return err
		}
		if wti == nil {
			return sdk.NewErrorFrom(sdk.ErrNotFound, "No workflow template instance found")
		}
		return service.WriteJSON(w, wti, http.StatusOK)
	}
}

func (api *API) pullTemplateHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		ctx, err := api.middlewareTemplate(false)(ctx, w, r)
		if err != nil {
			return err
		}

		wt := getWorkflowTemplate(ctx)

		if err := group.AggregateOnWorkflowTemplate(api.mustDB(), wt); err != nil {
			return err
		}

		buf := new(bytes.Buffer)
		if err := workflowtemplate.Pull(wt, exportentities.FormatYAML, buf); err != nil {
			return err
		}

		w.Header().Add("Content-Type", "application/tar")
		w.WriteHeader(http.StatusOK)
		_, err = io.Copy(w, buf)
		return sdk.WrapError(err, "Unable to copy content buffer in the response writer")
	}
}

func (api *API) pushTemplateHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		btes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Error("%v", sdk.WrapError(err, "Unable to read body"))
			return sdk.ErrWrongRequest
		}
		defer r.Body.Close()

		tr := tar.NewReader(bytes.NewReader(btes))

		msgs, wt, err := workflowtemplate.Push(api.mustDB(), getUser(ctx), tr)
		if err != nil {
			return sdk.WrapError(err, "Cannot push template")
		}

		if wt != nil {
			if err := group.AggregateOnWorkflowTemplate(api.mustDB(), wt); err != nil {
				return err
			}
			w.Header().Add(sdk.ResponseTemplateGroupNameHeader, wt.Group.Name)
			w.Header().Add(sdk.ResponseTemplateSlugHeader, wt.Slug)
		}

		return service.WriteJSON(w, translate(r, msgs), http.StatusOK)
	}
}
