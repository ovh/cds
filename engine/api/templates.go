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

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/ascode"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/operation"
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
			wt, err := exportentities.ReadTemplateFromTar(tar.NewReader(t))
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

		data.Version = 1

		if !isGroupAdmin(ctx, grp) && !isAdmin(ctx) {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		// execute template with no instance only to check if parsing is ok
		if _, err := workflowtemplate.Parse(data); err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		// duplicate couple of group id and slug will failed with sql constraint
		if err := workflowtemplate.Insert(tx, &data); err != nil {
			return err
		}

		newTemplate, err := workflowtemplate.LoadByID(ctx, tx, data.ID, workflowtemplate.LoadOptions.Default)
		if err != nil {
			return err
		}

		if err := workflowtemplate.CreateAuditAdd(tx, *newTemplate, getAPIConsumer(ctx)); err != nil {
			return err
		}

		if err := workflowtemplate.LoadOptions.WithAudits(ctx, tx, newTemplate); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
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
			wt, err := exportentities.ReadTemplateFromTar(tar.NewReader(t))
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

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint

		// update fields from request data
		clone := sdk.WorkflowTemplate(*old)
		clone.Update(data)

		// execute template with no instance only to check if parsing is ok
		if _, err := workflowtemplate.Parse(clone); err != nil {
			return err
		}

		if err := workflowtemplate.Update(tx, &clone); err != nil {
			return err
		}

		newTemplate, err := workflowtemplate.LoadByID(ctx, tx, clone.ID, workflowtemplate.LoadOptions.Default)
		if err != nil {
			return err
		}

		if err := workflowtemplate.CreateAuditUpdate(tx, *old, *newTemplate, data.ChangeMessage, getAPIConsumer(ctx)); err != nil {
			return err
		}

		if err := workflowtemplate.LoadOptions.WithAudits(ctx, tx, newTemplate); err != nil {
			return err
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
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
		branch := FormString(r, "branch")
		message := FormString(r, "message")

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

		// non admin user should have read/write to given project
		consumer := getAPIConsumer(ctx)
		if !consumer.Admin() {
			if err := api.checkProjectPermissions(ctx, req.ProjectKey, sdk.PermissionReadWriteExecute, nil); err != nil {
				return sdk.NewErrorFrom(sdk.ErrForbidden, "write permission on project required to import generated workflow.")
			}
		}

		// load project with key
		p, err := project.Load(api.mustDB(), req.ProjectKey,
			project.LoadOptions.WithGroups,
			project.LoadOptions.WithApplications,
			project.LoadOptions.WithEnvironments,
			project.LoadOptions.WithPipelines,
			project.LoadOptions.WithApplicationWithDeploymentStrategies,
			project.LoadOptions.WithIntegrations,
			project.LoadOptions.WithClearKeys,
		)
		if err != nil {
			return err
		}

		data := exportentities.WorkflowComponents{
			Template: exportentities.TemplateInstance{
				Name:       req.WorkflowName,
				From:       wt.PathWithVersion(),
				Parameters: req.Parameters,
			},
		}

		if !withImport && !req.Detached {
			buf := new(bytes.Buffer)
			if err := exportentities.TarWorkflowComponents(ctx, data, buf); err != nil {
				return err
			}
			return service.Write(w, buf.Bytes(), http.StatusOK, "application/tar")
		}

		// In case we want to generated a workflow not detached from the template, we need to check if the template
		// was not already applied to the same target workflow. If there is already a workflow and it's ascode we will
		// create a PR from the apply request and we will not execute the template.
		if withImport && !req.Detached {
			wti, err := workflowtemplate.LoadInstanceByTemplateIDAndProjectIDAndRequestWorkflowName(ctx, api.mustDB(), wt.ID, p.ID, req.WorkflowName)
			if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
				return err
			}
			if wti != nil && wti.WorkflowID != nil {
				existingWorkflow, err := workflow.LoadByID(ctx, api.mustDB(), api.Cache, *p, *wti.WorkflowID, workflow.LoadOptions{})
				if err != nil {
					return err
				}
				if existingWorkflow.FromRepository != "" {
					var rootApp *sdk.Application
					if existingWorkflow.WorkflowData.Node.Context != nil && existingWorkflow.WorkflowData.Node.Context.ApplicationID != 0 {
						rootApp, err = application.LoadByIDWithClearVCSStrategyPassword(api.mustDB(), existingWorkflow.WorkflowData.Node.Context.ApplicationID)
						if err != nil {
							return err
						}
					}
					if rootApp == nil {
						return sdk.NewErrorFrom(sdk.ErrWrongRequest, "cannot find the root application of the workflow")
					}

					ope, err := operation.PushOperationUpdate(ctx, api.mustDB(), api.Cache, *p, data, rootApp.VCSServer, rootApp.RepositoryFullname, branch, message, rootApp.RepositoryStrategy, consumer)
					if err != nil {
						return err
					}

					sdk.GoRoutine(context.Background(), fmt.Sprintf("UpdateAsCodeResult-%s", ope.UUID), func(ctx context.Context) {
						ed := ascode.EntityData{
							Operation: ope,
							Name:      existingWorkflow.Name,
							ID:        existingWorkflow.ID,
							Type:      ascode.WorkflowEvent,
							FromRepo:  existingWorkflow.FromRepository,
						}
						asCodeEvent := ascode.UpdateAsCodeResult(ctx, api.mustDB(), api.Cache, *p, *rootApp, ed, consumer)
						if asCodeEvent != nil {
							event.PublishAsCodeEvent(ctx, p.Key, *asCodeEvent, consumer)
						}
					}, api.PanicDump())

					return service.WriteJSON(w, ope, http.StatusOK)
				}
			}
		}

		mods := []workflowtemplate.TemplateRequestModifierFunc{
			workflowtemplate.TemplateRequestModifiers.DefaultKeys(*p),
		}
		if req.Detached {
			mods = append(mods, workflowtemplate.TemplateRequestModifiers.Detached)
		}
		_, wti, err := workflowtemplate.CheckAndExecuteTemplate(ctx, api.mustDB(), *consumer, *p, &data, mods...)
		if err != nil {
			return err
		}

		log.Debug("postTemplateApplyHandler> template %s applied (withImport=%v)", wt.Slug, withImport)

		if !withImport {
			buf := new(bytes.Buffer)
			if err := exportentities.TarWorkflowComponents(ctx, data, buf); err != nil {
				return err
			}
			return service.Write(w, buf.Bytes(), http.StatusOK, "application/tar")
		}

		msgs, wkf, oldWkf, err := workflow.Push(ctx, api.mustDB(), api.Cache, p, data, nil, consumer, project.DecryptWithBuiltinKey)
		if err != nil {
			return sdk.WrapError(err, "cannot push generated workflow")
		}
		if err := workflowtemplate.UpdateTemplateInstanceWithWorkflow(ctx, api.mustDB(), *wkf, *consumer, wti); err != nil {
			return err
		}

		msgStrings := translate(r, msgs)

		log.Debug("postTemplateApplyHandler> importing the workflow %s from template %s", wkf.Name, wt.Slug)

		if w != nil {
			w.Header().Add(sdk.ResponseWorkflowIDHeader, fmt.Sprintf("%d", wkf.ID))
			w.Header().Add(sdk.ResponseWorkflowNameHeader, wkf.Name)
		}

		if oldWkf != nil {
			event.PublishWorkflowUpdate(ctx, p.Key, *wkf, *oldWkf, getAPIConsumer(ctx))
		} else {
			event.PublishWorkflowAdd(ctx, p.Key, *wkf, getAPIConsumer(ctx))
		}

		return service.WriteJSON(w, msgStrings, http.StatusOK)
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

		branch := FormString(r, "branch")
		message := FormString(r, "message")

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
							log.Error(ctx, "%+v", err)
							bulk.Operations[i].Status = sdk.OperationStatusError
							bulk.Operations[i].Error = fmt.Sprintf("%s", sdk.Cause(err))
							if err := workflowtemplate.UpdateBulk(api.mustDB(), &bulk); err != nil {
								return err
							}
						}

						return nil
					}

					// load project with key
					p, err := project.Load(api.mustDB(), bulk.Operations[i].Request.ProjectKey,
						project.LoadOptions.WithGroups,
						project.LoadOptions.WithApplications,
						project.LoadOptions.WithEnvironments,
						project.LoadOptions.WithPipelines,
						project.LoadOptions.WithApplicationWithDeploymentStrategies,
						project.LoadOptions.WithIntegrations,
						project.LoadOptions.WithClearKeys,
					)
					if err != nil {
						if errD := errorDefer(err); errD != nil {
							log.Error(ctx, "%v", errD)
							return
						}
						continue
					}

					// apply and import workflow
					data := exportentities.WorkflowComponents{
						Template: exportentities.TemplateInstance{
							Name:       bulk.Operations[i].Request.WorkflowName,
							From:       wt.PathWithVersion(),
							Parameters: bulk.Operations[i].Request.Parameters,
						},
					}

					// In case we want to update a workflow that is ascode, we want to create a PR instead of pushing directly the new workflow.
					wti, err := workflowtemplate.LoadInstanceByTemplateIDAndProjectIDAndRequestWorkflowName(ctx, api.mustDB(), wt.ID, p.ID, data.Template.Name)
					if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
						if errD := errorDefer(err); errD != nil {
							log.Error(ctx, "%v", errD)
							return
						}
						continue
					}
					if wti != nil && wti.WorkflowID != nil {
						existingWorkflow, err := workflow.LoadByID(ctx, api.mustDB(), api.Cache, *p, *wti.WorkflowID, workflow.LoadOptions{})
						if err != nil {
							if errD := errorDefer(err); errD != nil {
								log.Error(ctx, "%v", errD)
								return
							}
							continue
						}
						if existingWorkflow.FromRepository != "" {
							var rootApp *sdk.Application
							if existingWorkflow.WorkflowData.Node.Context != nil && existingWorkflow.WorkflowData.Node.Context.ApplicationID != 0 {
								rootApp, err = application.LoadByIDWithClearVCSStrategyPassword(api.mustDB(), existingWorkflow.WorkflowData.Node.Context.ApplicationID)
								if err != nil {
									if errD := errorDefer(err); errD != nil {
										log.Error(ctx, "%v", errD)
										return
									}
									continue
								}
							}
							if rootApp == nil {
								if errD := errorDefer(sdk.NewErrorFrom(sdk.ErrWrongRequest, "cannot find the root application of the workflow")); errD != nil {
									log.Error(ctx, "%v", errD)
									return
								}
								continue
							}

							ope, err := operation.PushOperationUpdate(ctx, api.mustDB(), api.Cache, *p, data, rootApp.VCSServer, rootApp.RepositoryFullname, branch, message, rootApp.RepositoryStrategy, consumer)
							if err != nil {
								if errD := errorDefer(err); errD != nil {
									log.Error(ctx, "%v", errD)
									return
								}
								continue
							}

							ed := ascode.EntityData{
								Operation: ope,
								Name:      existingWorkflow.Name,
								ID:        existingWorkflow.ID,
								Type:      ascode.WorkflowEvent,
								FromRepo:  existingWorkflow.FromRepository,
							}
							asCodeEvent := ascode.UpdateAsCodeResult(ctx, api.mustDB(), api.Cache, *p, *rootApp, ed, consumer)
							if asCodeEvent != nil {
								event.PublishAsCodeEvent(ctx, p.Key, *asCodeEvent, consumer)
							}

							bulk.Operations[i].Status = sdk.OperationStatusDone
							if err := workflowtemplate.UpdateBulk(api.mustDB(), &bulk); err != nil {
								log.Error(ctx, "%v", err)
								return
							}

							continue
						}
					}

					mods := []workflowtemplate.TemplateRequestModifierFunc{
						workflowtemplate.TemplateRequestModifiers.DefaultKeys(*p),
					}
					_, wti, err = workflowtemplate.CheckAndExecuteTemplate(ctx, api.mustDB(), *consumer, *p, &data, mods...)
					if err != nil {
						if errD := errorDefer(err); errD != nil {
							log.Error(ctx, "%v", errD)
							return
						}
						continue
					}

					_, wkf, _, err := workflow.Push(ctx, api.mustDB(), api.Cache, p, data, nil, consumer, project.DecryptWithBuiltinKey)
					if err != nil {
						if errD := errorDefer(sdk.WrapError(err, "cannot push generated workflow")); errD != nil {
							log.Error(ctx, "%v", errD)
							return
						}
						continue
					}

					if err := workflowtemplate.UpdateTemplateInstanceWithWorkflow(ctx, api.mustDB(), *wkf, *consumer, wti); err != nil {
						if errD := errorDefer(err); errD != nil {
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
			ps, err = project.LoadAll(ctx, api.mustDB(), api.Cache, project.LoadOptions.WithKeys)
		} else {
			ps, err = project.LoadAllByGroupIDs(ctx, api.mustDB(), api.Cache, getAPIConsumer(ctx).GetGroupIDs(), project.LoadOptions.WithKeys)
		}
		if err != nil {
			return err
		}

		is, err := workflowtemplate.LoadInstancesByTemplateIDAndProjectIDs(ctx, api.mustDB(), wt.ID, sdk.ProjectsToIDs(ps),
			workflowtemplate.LoadInstanceOptions.WithAudits)
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

		// Add project and workflow on instances
		isPointers := make([]*sdk.WorkflowTemplateInstance, len(is))
		for i := range is {
			isPointers[i] = &is[i]
		}
		if err := workflow.AggregateOnWorkflowTemplateInstance(ctx, api.mustDB(), isPointers...); err != nil {
			return err
		}

		return service.WriteJSON(w, is, http.StatusOK)
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

		wti, err := workflowtemplate.LoadInstanceByIDForTemplateIDAndProjectIDs(ctx, api.mustDB(), instanceID, wt.ID, sdk.ProjectsToIDs(ps))
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
			return sdk.WithStack(sdk.ErrWrongRequest)
		}
		defer r.Body.Close()

		tr := tar.NewReader(bytes.NewReader(btes))
		wt, err := exportentities.ReadTemplateFromTar(tr)
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

		as, err := workflowtemplate.LoadAuditsByTemplateIDAndVersionGTE(api.mustDB(), wt.ID, version)
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
