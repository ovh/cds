package api

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/api/workflowtemplate"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getTemplatesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var ts []sdk.WorkflowTemplate
		var err error
		if isMaintainer(ctx) {
			ts, err = workflowtemplate.LoadAll(ctx, api.mustDB(),
				workflowtemplate.LoadOptions.Default,
				workflowtemplate.LoadOptions.WithAudits,
			)
		} else {
			ts, err = workflowtemplate.LoadAllByGroupIDs(ctx, api.mustDB(),
				append(getAPIConsumer(ctx).GetGroupIDs(), group.SharedInfraGroup.ID),
				workflowtemplate.LoadOptions.Default,
				workflowtemplate.LoadOptions.WithAudits,
			)
		}
		if err != nil {
			return err
		}

		return service.WriteJSON(w, ts, http.StatusOK)
	}
}

func (api *API) postTemplateHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var data sdk.WorkflowTemplate
		if err := service.UnmarshalBody(r, &data); err != nil {
			return err
		}
		if err := data.IsValid(); err != nil {
			return err
		}

		var grp *sdk.Group
		var err error
		// if imported from url try to download files then overrides request
		if data.ImportURL != "" {
			t := new(bytes.Buffer)
			if err := exportentities.DownloadTemplate(data.ImportURL, t); err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			wt, err := ReadFromTar(tar.NewReader(t))
			if err != nil {
				return err
			}
			wt.ImportURL = data.ImportURL
			data = wt

			// group name should be set
			if data.Group == nil {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing group name")
			}

			grp, err = group.LoadByName(ctx, api.mustDB(), data.Group.Name, group.LoadOptions.WithMembers)
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			data.GroupID = grp.ID

			// check the workflow template extracted
			if err := data.IsValid(); err != nil {
				return err
			}
		} else {
			grp, err = group.LoadByID(ctx, api.mustDB(), data.GroupID, group.LoadOptions.WithMembers)
			if err != nil {
				return err
			}
		}

		data.Version = 0

		if !isGroupAdmin(ctx, grp) && !isAdmin(ctx) {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		// execute template with no instance only to check if parsing is ok
		if _, err := workflowtemplate.Execute(&data, nil); err != nil {
			return err
		}

		// duplicate couple of group id and slug will failed with sql constraint
		if err := workflowtemplate.Insert(api.mustDB(), &data); err != nil {
			return err
		}

		newTemplate, err := workflowtemplate.LoadByID(ctx, api.mustDB(), data.ID, workflowtemplate.LoadOptions.Default)
		if err != nil {
			return err
		}

		event.PublishWorkflowTemplateAdd(ctx, *newTemplate, getAPIConsumer(ctx))

		if err := workflowtemplate.LoadOptions.WithAudits(ctx, api.mustDB(), newTemplate); err != nil {
			return err
		}

		// aggregate extra data for ui
		newTemplate.Editable = true

		return service.WriteJSON(w, newTemplate, http.StatusOK)
	}
}

func (api *API) getTemplateHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		groupName := vars["permGroupName"]
		templateSlug := vars["permTemplateSlug"]

		g, err := group.LoadByName(ctx, api.mustDB(), groupName, group.LoadOptions.WithMembers)
		if err != nil {
			return err
		}

		wt, err := workflowtemplate.LoadBySlugAndGroupID(ctx, api.mustDB(), templateSlug, g.ID,
			workflowtemplate.LoadOptions.Default,
			workflowtemplate.LoadOptions.WithAudits,
		)
		if err != nil {
			return err
		}
		wt.Editable = isGroupAdmin(ctx, g) || isAdmin(ctx)

		return service.WriteJSON(w, wt, http.StatusOK)
	}
}

func (api *API) putTemplateHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		groupName := vars["permGroupName"]
		templateSlug := vars["permTemplateSlug"]

		g, err := group.LoadByName(ctx, api.mustDB(), groupName, group.LoadOptions.WithMembers)
		if err != nil {
			return err
		}

		old, err := workflowtemplate.LoadBySlugAndGroupID(ctx, api.mustDB(), templateSlug, g.ID, workflowtemplate.LoadOptions.Default)
		if err != nil {
			return err
		}

		data := sdk.WorkflowTemplate{}
		if err := service.UnmarshalBody(r, &data); err != nil {
			return err
		}
		if err := data.IsValid(); err != nil {
			return err
		}

		var grp *sdk.Group
		// if imported from url try to download files then overrides request
		if data.ImportURL != "" {
			t := new(bytes.Buffer)
			if err := exportentities.DownloadTemplate(data.ImportURL, t); err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			wt, err := ReadFromTar(tar.NewReader(t))
			if err != nil {
				return err
			}
			wt.ImportURL = data.ImportURL
			data = wt

			// group name should be set
			if data.Group == nil {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing group name")
			}

			// check that the user is admin on the given template's group
			grp, err = group.LoadByName(ctx, api.mustDB(), data.Group.Name, group.LoadOptions.WithMembers)
			if err != nil {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
			data.GroupID = grp.ID

			// check the workflow template extracted
			if err := data.IsValid(); err != nil {
				return err
			}
		} else {
			// check that the group exists and user is admin for group id
			grp, err = group.LoadByID(ctx, api.mustDB(), data.GroupID, group.LoadOptions.WithMembers)
			if err != nil {
				return err
			}
		}

		if !isGroupAdmin(ctx, grp) && !isAdmin(ctx) {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		// update fields from request data
		clone := sdk.WorkflowTemplate(*old)
		clone.Update(data)

		// execute template with no instance only to check if parsing is ok
		if _, err := workflowtemplate.Execute(&clone, nil); err != nil {
			return err
		}

		if err := workflowtemplate.Update(api.mustDB(), &clone); err != nil {
			return err
		}

		newTemplate, err := workflowtemplate.LoadByID(ctx, api.mustDB(), clone.ID, workflowtemplate.LoadOptions.Default)
		if err != nil {
			return err
		}

		event.PublishWorkflowTemplateUpdate(ctx, *old, *newTemplate, data.ChangeMessage, getAPIConsumer(ctx))

		if err := workflowtemplate.LoadOptions.WithAudits(ctx, api.mustDB(), newTemplate); err != nil {
			return err
		}

		// aggregate extra data for ui
		newTemplate.Editable = true

		return service.WriteJSON(w, newTemplate, http.StatusOK)
	}
}

func (api *API) deleteTemplateHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		groupName := vars["permGroupName"]
		templateSlug := vars["permTemplateSlug"]

		g, err := group.LoadByName(ctx, api.mustDB(), groupName)
		if err != nil {
			return err
		}

		wt, err := workflowtemplate.LoadBySlugAndGroupID(ctx, api.mustDB(), templateSlug, g.ID)
		if err != nil {
			return err
		}

		if err := workflowtemplate.Delete(api.mustDB(), wt); err != nil {
			return err
		}

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) applyTemplate(ctx context.Context, u sdk.Identifiable, p *sdk.Project, wt *sdk.WorkflowTemplate, req sdk.WorkflowTemplateRequest) (sdk.WorkflowTemplateResult, error) {
	var result sdk.WorkflowTemplateResult

	tx, err := api.mustDB().Begin()
	if err != nil {
		return result, sdk.WrapError(err, "cannot start transaction")
	}
	defer func() { _ = tx.Rollback() }()

	var wti *sdk.WorkflowTemplateInstance
	// try to get a instance not assign to a workflow but with the same slug
	wtis, err := workflowtemplate.GetInstancesByTemplateIDAndProjectIDAndRequestWorkflowName(tx, wt.ID, p.ID, req.WorkflowName)
	if err != nil {
		return result, err
	}

	for _, res := range wtis {
		if wti == nil {
			wti = &res
		} else {
			// if there are more than one instance found, delete others
			if err := workflowtemplate.DeleteInstance(tx, &res); err != nil {
				return result, err
			}
		}
	}

	// if the request is for a detached workflow and there is an existing instance, remove it
	if wti != nil && req.Detached {
		if err := workflowtemplate.DeleteInstance(tx, wti); err != nil {
			return result, err
		}
		wti = nil
	}

	// if a previous instance exist for the same workflow update it, else create a new one
	var old *sdk.WorkflowTemplateInstance
	if wti != nil {
		clone := sdk.WorkflowTemplateInstance(*wti)
		old = &clone
		wti.WorkflowTemplateVersion = wt.Version
		wti.Request = req
		if err := workflowtemplate.UpdateInstance(tx, wti); err != nil {
			return result, err
		}
	} else {
		wti = &sdk.WorkflowTemplateInstance{
			ProjectID:               p.ID,
			WorkflowTemplateID:      wt.ID,
			WorkflowTemplateVersion: wt.Version,
			Request:                 req,
		}

		// only store the new instance if request is not for a detached workflow
		if !req.Detached {
			if err := workflowtemplate.InsertInstance(tx, wti); err != nil {
				return result, err
			}
		} else {
			// if is a detached apply set an id based on time
			wti.ID = time.Now().Unix()
		}
	}

	// execute template with request
	result, err = workflowtemplate.Execute(wt, wti)
	if err != nil {
		return result, err
	}

	// parse the generated workflow to find its name an update it in instance if not detached
	// also set the template path in generated workflow if not detached
	if !req.Detached {
		var wor exportentities.Workflow
		if err := yaml.Unmarshal([]byte(result.Workflow), &wor); err != nil {
			return result, sdk.NewError(sdk.Error{
				ID:      sdk.ErrWrongRequest.ID,
				Message: "Cannot parse generated workflow",
			}, err)
		}

		wti.WorkflowName = wor.Name
		if err := workflowtemplate.UpdateInstance(tx, wti); err != nil {
			return result, err
		}

		templatePath := fmt.Sprintf("%s/%s", wt.Group.Name, wt.Slug)
		wor.Template = &templatePath
		b, err := yaml.Marshal(wor)
		if err != nil {
			return result, sdk.NewError(sdk.Error{
				ID:      sdk.ErrWrongRequest.ID,
				Message: "Cannot add template info to generated workflow",
			}, err)
		}
		result.Workflow = string(b)
	}

	if err := tx.Commit(); err != nil {
		return result, sdk.WrapError(err, "cannot commit transaction")
	}

	if old != nil {
		event.PublishWorkflowTemplateInstanceUpdate(ctx, *old, *wti, u)
	} else if !req.Detached {
		event.PublishWorkflowTemplateInstanceAdd(ctx, *wti, u)
	}

	return result, nil
}

func (api *API) postTemplateApplyHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		groupName := vars["groupName"]
		templateSlug := vars["templateSlug"]

		g, err := group.LoadByName(ctx, api.mustDB(), groupName, group.LoadOptions.WithMembers)
		if err != nil {
			return err
		}
		if !(isGroupMember(ctx, g) || isMaintainer(ctx)) {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		wt, err := workflowtemplate.LoadBySlugAndGroupID(ctx, api.mustDB(), templateSlug, g.ID, workflowtemplate.LoadOptions.Default)
		if err != nil {
			return err
		}

		withImport := FormBool(r, "import")

		// parse and check request
		var req sdk.WorkflowTemplateRequest
		if err := service.UnmarshalBody(r, &req); err != nil {
			return err
		}
		if err := wt.CheckParams(req); err != nil {
			return err
		}

		// check permission on project
		if !withImport {
			var hasRPermission = api.checkProjectPermissions(ctx, req.ProjectKey, sdk.PermissionRead, nil) == nil
			if !hasRPermission && !isMaintainer(ctx) && !isAdmin(ctx) {
				return sdk.WithStack(sdk.ErrNoProject)
			}
		} else {
			var hasRWPermission = api.checkProjectPermissions(ctx, req.ProjectKey, sdk.PermissionReadWriteExecute, nil) == nil
			if !hasRWPermission && !isAdmin(ctx) {
				return sdk.NewErrorFrom(sdk.ErrForbidden, "write permission on project required to import generated workflow.")
			}
		}

		// load project with key
		p, err := project.Load(api.mustDB(), api.Cache, req.ProjectKey,
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithApplications,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithIntegrations)
		if err != nil {
			return err
		}

		res, err := api.applyTemplate(ctx, getAPIConsumer(ctx), p, wt, req)
		if err != nil {
			return err
		}

		buf := new(bytes.Buffer)
		if err := workflowtemplate.Tar(ctx, wt, res, buf); err != nil {
			return err
		}

		if withImport {
			tr := tar.NewReader(buf)

			msgs, wkf, err := workflow.Push(ctx, api.mustDB(), api.Cache, p, tr, nil, getAPIConsumer(ctx), project.DecryptWithBuiltinKey)
			if err != nil {
				return sdk.WrapError(err, "cannot push generated workflow")
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

func (api *API) postTemplateBulkHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		groupName := vars["groupName"]
		templateSlug := vars["templateSlug"]

		g, err := group.LoadByName(ctx, api.mustDB(), groupName, group.LoadOptions.WithMembers)
		if err != nil {
			return err
		}
		if !(isGroupMember(ctx, g) || isMaintainer(ctx)) {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		wt, err := workflowtemplate.LoadBySlugAndGroupID(ctx, api.mustDB(), templateSlug, g.ID, workflowtemplate.LoadOptions.Default)
		if err != nil {
			return err
		}

		// check all requests
		var req sdk.WorkflowTemplateBulk
		if err := service.UnmarshalBody(r, &req); err != nil {
			return err
		}
		m := make(map[string]struct{}, len(req.Operations))
		for _, o := range req.Operations {
			// check for duplicated request
			key := fmt.Sprintf("%s-%s", o.Request.ProjectKey, o.Request.WorkflowName)
			if _, ok := m[key]; ok {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "request should be unique for a given project key and workflow name")
			}
			m[key] = struct{}{}

			// check request params
			if err := wt.CheckParams(o.Request); err != nil {
				return err
			}
		}

		consumer := getAPIConsumer(ctx)

		// non admin user should have read/write access to all given project
		if !consumer.Admin() {
			for i := range req.Operations {
				if err := api.checkProjectPermissions(ctx, req.Operations[i].Request.ProjectKey, sdk.PermissionReadWriteExecute, nil); err != nil {
					return sdk.NewErrorFrom(sdk.ErrForbidden, "write permission on project required to import generated workflow.")
				}
			}
		}

		// store the bulk request
		bulk := sdk.WorkflowTemplateBulk{
			UserID:             consumer.AuthentifiedUser.ID,
			WorkflowTemplateID: wt.ID,
			Operations:         make([]sdk.WorkflowTemplateBulkOperation, len(req.Operations)),
		}
		for i := range req.Operations {
			bulk.Operations[i].Status = sdk.OperationStatusPending
			bulk.Operations[i].Request = req.Operations[i].Request
		}
		if err := workflowtemplate.InsertBulk(api.mustDB(), &bulk); err != nil {
			return err
		}

		// start async bulk tasks
		sdk.GoRoutine(context.Background(), "api.templateBulkApply", func(ctx context.Context) {
			for i := range bulk.Operations {
				if bulk.Operations[i].Status == sdk.OperationStatusPending {
					bulk.Operations[i].Status = sdk.OperationStatusProcessing
					if err := workflowtemplate.UpdateBulk(api.mustDB(), &bulk); err != nil {
						log.Error(ctx, "%v", err)
						return
					}

					errorDefer := func(err error) error {
						if err != nil {
							bulk.Operations[i].Status = sdk.OperationStatusError
							bulk.Operations[i].Error = fmt.Sprintf("%s", sdk.Cause(err))
							if err := workflowtemplate.UpdateBulk(api.mustDB(), &bulk); err != nil {
								return err
							}
						}

						return nil
					}

					// load project with key
					p, err := project.Load(api.mustDB(), api.Cache, bulk.Operations[i].Request.ProjectKey,
						project.LoadOptions.WithGroups,
						project.LoadOptions.WithApplications,
						project.LoadOptions.WithEnvironments,
						project.LoadOptions.WithPipelines,
						project.LoadOptions.WithApplicationWithDeploymentStrategies,
						project.LoadOptions.WithIntegrations)
					if err != nil {
						if errD := errorDefer(err); errD != nil {
							log.Error(ctx, "%v", errD)
							return
						}
						continue
					}

					// apply and import workflow
					res, err := api.applyTemplate(ctx, consumer, p, wt, bulk.Operations[i].Request)
					if err != nil {
						if errD := errorDefer(err); errD != nil {
							log.Error(ctx, "%v", errD)
							return
						}
						continue
					}

					buf := new(bytes.Buffer)
					if err := workflowtemplate.Tar(ctx, wt, res, buf); err != nil {
						if errD := errorDefer(err); errD != nil {
							log.Error(ctx, "%v", errD)
							return
						}
						continue
					}

					tr := tar.NewReader(buf)

					_, _, err = workflow.Push(ctx, api.mustDB(), api.Cache, p, tr, nil, consumer, project.DecryptWithBuiltinKey)
					if err != nil {
						if errD := errorDefer(sdk.WrapError(err, "cannot push generated workflow")); errD != nil {
							log.Error(ctx, "%v", errD)
							return
						}
						continue
					}

					bulk.Operations[i].Status = sdk.OperationStatusDone
					if err := workflowtemplate.UpdateBulk(api.mustDB(), &bulk); err != nil {
						log.Error(ctx, "%v", err)
						return
					}
				}
			}
		})

		// returns created bulk
		return service.WriteJSON(w, bulk, http.StatusOK)
	}
}

func (api *API) getTemplateBulkHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		id, _ := requestVarInt(r, "bulkID") // ignore error, will check if not 0
		if id == 0 {
			return sdk.WrapError(sdk.ErrWrongRequest, "invalid given id")
		}

		vars := mux.Vars(r)

		groupName := vars["groupName"]
		templateSlug := vars["templateSlug"]

		g, err := group.LoadByName(ctx, api.mustDB(), groupName, group.LoadOptions.WithMembers)
		if err != nil {
			return err
		}
		if !(isGroupMember(ctx, g) || isMaintainer(ctx)) {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		wt, err := workflowtemplate.LoadBySlugAndGroupID(ctx, api.mustDB(), templateSlug, g.ID)
		if err != nil {
			return err
		}

		b, err := workflowtemplate.GetBulkByIDAndTemplateID(api.mustDB(), id, wt.ID)
		if err != nil {
			return err
		}
		if b == nil || (b.UserID != getAPIConsumer(ctx).AuthentifiedUser.ID && !isMaintainer(ctx) && !isAdmin(ctx)) {
			return sdk.NewErrorFrom(sdk.ErrNotFound, "no workflow template bulk found for id %d", id)
		}
		sort.Slice(b.Operations, func(i, j int) bool {
			return b.Operations[i].Request.WorkflowName < b.Operations[j].Request.WorkflowName
		})

		return service.WriteJSON(w, b, http.StatusOK)
	}
}

func (api *API) getTemplateInstancesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		groupName := vars["groupName"]
		templateSlug := vars["templateSlug"]

		g, err := group.LoadByName(ctx, api.mustDB(), groupName, group.LoadOptions.WithMembers)
		if err != nil {
			return err
		}
		if !(isGroupMember(ctx, g) || isMaintainer(ctx)) {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		wt, err := workflowtemplate.LoadBySlugAndGroupID(ctx, api.mustDB(), templateSlug, g.ID)
		if err != nil {
			return err
		}

		var ps sdk.Projects
		if isMaintainer(ctx) {
			ps, err = project.LoadAll(ctx, api.mustDB(), api.Cache)
		} else {
			ps, err = project.LoadAllByGroupIDs(ctx, api.mustDB(), api.Cache, getAPIConsumer(ctx).GetGroupIDs())
		}
		if err != nil {
			return err
		}

		is, err := workflowtemplate.GetInstancesByTemplateIDAndProjectIDs(api.mustDB(), wt.ID, sdk.ProjectsToIDs(ps))
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
		proj, err := project.Load(api.mustDB(), api.Cache, key, project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WrapError(err, "unable to load projet")
		}
		wf, err := workflow.Load(ctx, api.mustDB(), api.Cache, proj, workflowName, workflow.LoadOptions{})
		if err != nil {
			if sdk.ErrorIs(err, sdk.ErrWorkflowNotFound) {
				return sdk.NewErrorFrom(sdk.ErrNotFound, "cannot load workflow %s", workflowName)
			}
			return sdk.WithStack(err)
		}

		// return the template instance if workflow is a generated one
		wti, err := workflowtemplate.LoadInstanceByWorkflowID(ctx, api.mustDB(), wf.ID, workflowtemplate.LoadInstanceOptions.WithTemplate)
		if err != nil {
			return err
		}

		wti.Project = proj

		return service.WriteJSON(w, wti, http.StatusOK)
	}
}

func (api *API) deleteTemplateInstanceHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		groupName := vars["groupName"]
		templateSlug := vars["templateSlug"]

		g, err := group.LoadByName(ctx, api.mustDB(), groupName, group.LoadOptions.WithMembers)
		if err != nil {
			return err
		}
		if !(isGroupMember(ctx, g) || isMaintainer(ctx)) {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		wt, err := workflowtemplate.LoadBySlugAndGroupID(ctx, api.mustDB(), templateSlug, g.ID)
		if err != nil {
			return err
		}

		var ps []sdk.Project
		if isAdmin(ctx) {
			ps, err = project.LoadAll(ctx, api.mustDB(), api.Cache)
		} else {
			ps, err = project.LoadAllByGroupIDs(ctx, api.mustDB(), api.Cache, getAPIConsumer(ctx).GetGroupIDs())
		}
		if err != nil {
			return err
		}

		instanceID, err := requestVarInt(r, "instanceID")
		if err != nil {
			return err
		}

		wti, err := workflowtemplate.GetInstanceByIDForTemplateIDAndProjectIDs(api.mustDB(), instanceID, wt.ID, sdk.ProjectsToIDs(ps))
		if err != nil {
			return err
		}
		if wti == nil {
			return sdk.NewErrorFrom(sdk.ErrNotFound, "no workflow template instance found")
		}

		if err := workflowtemplate.DeleteInstance(api.mustDB(), wti); err != nil {
			return err
		}

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) postTemplatePullHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		groupName := vars["permGroupName"]
		templateSlug := vars["permTemplateSlug"]

		g, err := group.LoadByName(ctx, api.mustDB(), groupName, group.LoadOptions.WithMembers)
		if err != nil {
			return err
		}

		wt, err := workflowtemplate.LoadBySlugAndGroupID(ctx, api.mustDB(), templateSlug, g.ID, workflowtemplate.LoadOptions.Default)
		if err != nil {
			return err
		}

		buf := new(bytes.Buffer)
		if err := workflowtemplate.Pull(ctx, wt, exportentities.FormatYAML, buf); err != nil {
			return err
		}

		w.Header().Add("Content-Type", "application/tar")
		w.WriteHeader(http.StatusOK)
		_, err = io.Copy(w, buf)
		return sdk.WrapError(err, "unable to copy content buffer in the response writer")
	}
}

func (api *API) postTemplatePushHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		btes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Error(ctx, "%v", sdk.WrapError(err, "unable to read body"))
			return sdk.ErrWrongRequest
		}
		defer r.Body.Close()

		tr := tar.NewReader(bytes.NewReader(btes))
		wt, err := ReadFromTar(tr)
		if err != nil {
			return err
		}

		// group name should be set
		if wt.Group == nil {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing group name")
		}

		// check that the user is admin on the given template's group
		grp, err := group.LoadByName(ctx, api.mustDB(), wt.Group.Name, group.LoadOptions.WithMembers)
		if err != nil {
			return sdk.NewError(sdk.ErrWrongRequest, err)
		}
		wt.GroupID = grp.ID

		if !isGroupAdmin(ctx, grp) && !isAdmin(ctx) {
			return sdk.WithStack(sdk.ErrInvalidGroupAdmin)
		}

		// check the workflow template extracted
		if err := wt.IsValid(); err != nil {
			return err
		}

		msgs, err := workflowtemplate.Push(ctx, api.mustDB(), &wt, getAPIConsumer(ctx))
		if err != nil {
			return sdk.WrapError(err, "cannot push template")
		}

		w.Header().Add(sdk.ResponseTemplateGroupNameHeader, wt.Group.Name)
		w.Header().Add(sdk.ResponseTemplateSlugHeader, wt.Slug)

		return service.WriteJSON(w, translate(r, msgs), http.StatusOK)
	}
}

func (api *API) getTemplateAuditsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		groupName := vars["permGroupName"]
		templateSlug := vars["permTemplateSlug"]

		g, err := group.LoadByName(ctx, api.mustDB(), groupName)
		if err != nil {
			return err
		}

		wt, err := workflowtemplate.LoadBySlugAndGroupID(ctx, api.mustDB(), templateSlug, g.ID)
		if err != nil {
			return err
		}

		since := r.FormValue("sinceVersion")
		var version int64
		if since != "" {
			version, err = strconv.ParseInt(since, 10, 64)
			if err != nil || version < 0 {
				return sdk.NewError(sdk.ErrWrongRequest, err)
			}
		}

		as, err := workflowtemplate.GetAuditsByTemplateIDAndVersionGTE(api.mustDB(), wt.ID, version)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, as, http.StatusOK)
	}
}

func (api *API) getTemplateUsageHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)

		groupName := vars["groupName"]
		templateSlug := vars["templateSlug"]

		g, err := group.LoadByName(ctx, api.mustDB(), groupName, group.LoadOptions.WithMembers)
		if err != nil {
			return err
		}
		if !(isGroupMember(ctx, g) || isMaintainer(ctx)) {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		wt, err := workflowtemplate.LoadBySlugAndGroupID(ctx, api.mustDB(), templateSlug, g.ID)
		if err != nil {
			return err
		}

		wfs, err := workflow.LoadByWorkflowTemplateID(ctx, api.mustDB(), wt.ID)
		if err != nil {
			return sdk.WrapError(err, "cannot load templates")
		}

		if !isMaintainer(ctx) {
			consumer := getAPIConsumer(ctx)

			// filter usage in workflow by user's projects
			ps, err := project.LoadAllByGroupIDs(ctx, api.mustDB(), api.Cache, consumer.GetGroupIDs())
			if err != nil {
				return err
			}
			mProjectIDs := make(map[int64]struct{}, len(ps))
			for i := range ps {
				mProjectIDs[ps[i].ID] = struct{}{}
			}

			filteredWorkflow := []sdk.Workflow{}
			for i := range wfs {
				if _, ok := mProjectIDs[wfs[i].ProjectID]; ok {
					filteredWorkflow = append(filteredWorkflow, wfs[i])
				}
			}
			wfs = filteredWorkflow
		}

		return service.WriteJSON(w, wfs, http.StatusOK)
	}
}

// ReadFromTar returns a workflow template from given tar reader.
func ReadFromTar(tr *tar.Reader) (sdk.WorkflowTemplate, error) {
	var wt sdk.WorkflowTemplate

	// extract template data from tar
	var apps, pips, envs [][]byte
	var wkf []byte
	var tmpl exportentities.Template

	mError := new(sdk.MultiError)
	var templateFileName string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return wt, sdk.NewError(sdk.ErrWrongRequest, sdk.WrapError(err, "Unable to read tar file"))
		}

		buff := new(bytes.Buffer)
		if _, err := io.Copy(buff, tr); err != nil {
			return wt, sdk.NewError(sdk.ErrWrongRequest, sdk.WrapError(err, "Unable to read tar file"))
		}

		b := buff.Bytes()
		switch {
		case strings.Contains(hdr.Name, ".application."):
			apps = append(apps, b)
		case strings.Contains(hdr.Name, ".pipeline."):
			pips = append(pips, b)
		case strings.Contains(hdr.Name, ".environment."):
			envs = append(envs, b)
		case hdr.Name == "workflow.yml":
			// if a workflow was already found, it's a mistake
			if len(wkf) != 0 {
				mError.Append(fmt.Errorf("Two workflow files found"))
				break
			}
			wkf = b
		default:
			// if a template was already found, it's a mistake
			if templateFileName != "" {
				mError.Append(fmt.Errorf("Two template files found: %s and %s", templateFileName, hdr.Name))
				break
			}
			if err := yaml.Unmarshal(b, &tmpl); err != nil {
				mError.Append(sdk.WrapError(err, "Unable to unmarshal template %s", hdr.Name))
				continue
			}
			templateFileName = hdr.Name
		}
	}

	if !mError.IsEmpty() {
		return wt, sdk.NewError(sdk.ErrWorkflowInvalid, mError)
	}

	// init workflow template struct from data
	wt = tmpl.GetTemplate(wkf, pips, apps, envs)

	return wt, nil
}
