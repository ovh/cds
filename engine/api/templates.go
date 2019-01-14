package api

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
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

		var g *sdk.Group
		var err error
		if groupName != "" {
			// check that group exists
			g, err = group.LoadGroup(api.mustDB(), groupName)
			if err != nil {
				return nil, err
			}

			if needAdmin {
				if err := group.CheckUserIsGroupAdmin(g, u); err != nil {
					return nil, err
				}
			} else {
				if err := group.CheckUserIsGroupMember(g, u); err != nil {
					return nil, err
				}
			}
		}
		gs := append(u.Groups, *group.SharedInfraGroup)

		var wt *sdk.WorkflowTemplate
		if id != 0 {
			if u.Admin {
				wt, err = workflowtemplate.GetByID(api.mustDB(), id)
			} else {
				wt, err = workflowtemplate.GetByIDAndGroupIDs(api.mustDB(), id, sdk.GroupsToIDs(gs))
			}
		} else {
			wt, err = workflowtemplate.GetBySlugAndGroupIDs(api.mustDB(), templateSlug, []int64{g.ID})
		}
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

		var ts []sdk.WorkflowTemplate
		var err error
		if u.Admin {
			ts, err = workflowtemplate.GetAll(api.mustDB())
		} else {
			ts, err = workflowtemplate.GetAllByGroupIDs(api.mustDB(), append(sdk.GroupsToIDs(u.Groups), group.SharedInfraGroup.ID))
		}
		if err != nil {
			return err
		}

		tsPointers := make([]*sdk.WorkflowTemplate, len(ts))
		for i := range ts {
			tsPointers[i] = &ts[i]
		}

		if err := group.AggregateOnWorkflowTemplate(api.mustDB(), tsPointers...); err != nil {
			return err
		}
		if err := workflowtemplate.AggregateAuditsOnWorkflowTemplate(api.mustDB(), tsPointers...); err != nil {
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
		if err := t.IsValid(); err != nil {
			return err
		}
		t.Version = 0

		u := getUser(ctx)

		// check that group exists
		g, err := group.LoadGroupByID(api.mustDB(), t.GroupID)
		if err != nil {
			return err
		}

		if err := group.CheckUserIsGroupAdmin(g, u); err != nil {
			return err
		}

		// execute template with no instance only to check if parsing is ok
		if _, err := workflowtemplate.Execute(&t, nil); err != nil {
			return err
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
		t.Editable = true

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
		if err := group.CheckUserIsGroupAdmin(t.Group, getUser(ctx)); err == nil {
			t.Editable = true
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
		if err := data.IsValid(); err != nil {
			return err
		}

		var err error
		ctx, err = api.middlewareTemplate(true)(ctx, w, r)
		if err != nil {
			return err
		}

		old := getWorkflowTemplate(ctx)
		u := getUser(ctx)

		// if group id has changed check that the group exists and user is admin for new group id
		if old.GroupID != data.GroupID {
			newGroup, err := group.LoadGroupByID(api.mustDB(), data.GroupID)
			if err != nil {
				return err
			}

			if err := group.CheckUserIsGroupAdmin(newGroup, u); err != nil {
				return err
			}
		}

		// update fields from request data
		new := sdk.WorkflowTemplate(*old)
		new.Update(data)

		// execute template with no instance only to check if parsing is ok
		if _, err := workflowtemplate.Execute(&new, nil); err != nil {
			return err
		}

		if err := workflowtemplate.Update(api.mustDB(), &new); err != nil {
			return err
		}

		event.PublishWorkflowTemplateUpdate(*old, new, data.ChangeMessage, u)

		if err := group.AggregateOnWorkflowTemplate(api.mustDB(), &new); err != nil {
			return err
		}
		if err := workflowtemplate.AggregateAuditsOnWorkflowTemplate(api.mustDB(), &new); err != nil {
			return err
		}
		new.Editable = true

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

		if err := workflowtemplate.Delete(api.mustDB(), wt); err != nil {
			return err
		}

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) applyTemplateHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		withImport := FormBool(r, "import")

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

		u := getUser(ctx)

		// check permission on project
		if !u.Admin {
			if !withImport && !checkProjectReadPermission(ctx, req.ProjectKey) {
				return sdk.WithStack(sdk.ErrNoProject)
			}
			if withImport && !api.checkProjectPermissions(ctx, req.ProjectKey, permission.PermissionReadWriteExecute, nil) {
				return sdk.NewErrorFrom(sdk.ErrForbidden, "Write permission on project required to import generated workflow.")
			}
		}

		// load project with key
		p, err := project.Load(api.mustDB(), api.Cache, req.ProjectKey, u,
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithApplications,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithPlatforms)
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
			wti, err = workflowtemplate.GetInstanceByWorkflowIDAndTemplateID(tx, wf.ID, wt.ID)
			if err != nil {
				return err
			}
		}
		if wti == nil {
			// try to get a instance not assign to a workflow but with the same slug
			wtis, err := workflowtemplate.GetInstancesByTemplateIDAndProjectIDAndWorkflowIDNull(tx, wt.ID, p.ID)
			if err != nil {
				return err
			}

			for _, res := range wtis {
				if res.Request.WorkflowName == req.WorkflowName {
					if wti == nil {
						wti = &res
					} else {
						// if there are more than one instance found, delete others
						if err := workflowtemplate.DeleteInstance(tx, &res); err != nil {
							return err
						}
					}
				}
			}
		}

		// if a previous instance exist for the same workflow update it, else create a new one
		var old *sdk.WorkflowTemplateInstance
		if wti != nil {
			clone := sdk.WorkflowTemplateInstance(*wti)
			old = &clone
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

		// parse the generated workflow to find its name
		var wor exportentities.Workflow
		if err := yaml.Unmarshal([]byte(res.Workflow), &wor); err != nil {
			return sdk.NewError(sdk.Error{
				ID:      sdk.ErrWrongRequest.ID,
				Message: "Cannot parse generated workflow",
			}, err)
		}
		wti.WorkflowName = wor.Name
		if err := workflowtemplate.UpdateInstance(tx, wti); err != nil {
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

		if withImport {
			tr := tar.NewReader(buf)

			msgs, wkf, err := workflow.Push(ctx, api.mustDB(), api.Cache, p, tr, nil, u, project.DecryptWithBuiltinKey)
			if err != nil {
				return sdk.WrapError(err, "Cannot push generated workflow")
			}
			msgStrings := translate(r, msgs)

			if w != nil {
				w.Header().Add(sdk.ResponseWorkflowIDHeader, fmt.Sprintf("%d", wkf.ID))
				w.Header().Add(sdk.ResponseWorkflowNameHeader, wkf.Name)
			}

			return service.WriteJSON(w, msgStrings, http.StatusOK)
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

		is, err := workflowtemplate.GetInstancesByTemplateIDAndProjectIDs(api.mustDB(), t.ID, sdk.ProjectsToIDs(ps))
		if err != nil {
			return err
		}

		mProjects := make(map[int64]sdk.Project, len(ps))
		for i := range ps {
			mProjects[ps[i].ID] = ps[i]
		}
		for i := range is {
			p := mProjects[is[i].ProjectID]
			is[i].Project = &p
		}

		isPointers := make([]*sdk.WorkflowTemplateInstance, len(is))
		for i := range is {
			isPointers[i] = &is[i]
		}

		if err := workflowtemplate.AggregateAuditsOnWorkflowTemplateInstance(api.mustDB(), isPointers...); err != nil {
			return err
		}
		if err := workflow.AggregateOnWorkflowTemplateInstance(api.mustDB(), isPointers...); err != nil {
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
		wti, err := workflowtemplate.GetInstanceByWorkflowID(api.mustDB(), wf.ID)
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

func (api *API) getTemplateAuditsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		ctx, err := api.middlewareTemplate(false)(ctx, w, r)
		if err != nil {
			return err
		}
		t := getWorkflowTemplate(ctx)

		since := r.FormValue("sinceVersion")
		var version int64
		if since != "" {
			version, err = strconv.ParseInt(since, 10, 64)
			if err != nil || version < 0 {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
		}

		as, err := workflowtemplate.GetAuditsByTemplateIDsAndEventTypesAndVersionGTE(api.mustDB(),
			[]int64{t.ID}, []string{"WorkflowTemplateAdd", "WorkflowTemplateUpdate"}, version)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, as, http.StatusOK)
	}
}

func (api *API) getTemplateUsageHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		ctx, err := api.middlewareTemplate(false)(ctx, w, r)
		if err != nil {
			return err
		}
		wfTmpl := getWorkflowTemplate(ctx)

		wfs, err := workflow.LoadByWorkflowTemplateID(ctx, api.mustDB(), wfTmpl.ID, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "Cannot load templates")
		}

		return service.WriteJSON(w, wfs, http.StatusOK)
	}
}
