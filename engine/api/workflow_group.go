package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// deleteWorkflowGroupHandler delete permission for a group on the workflow
func (api *API) deleteWorkflowGroupHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		groupName := vars["groupName"]
		u := getAPIConsumer(ctx)

		proj, err := project.Load(ctx, api.mustDB(), key, project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WrapError(err, "unable to load projet")
		}

		options := workflow.LoadOptions{}
		wf, err := workflow.Load(ctx, api.mustDB(), api.Cache, *proj, name, options)
		if err != nil {
			return sdk.WithStack(err)
		}

		var groupIndex int
		var oldGp sdk.GroupPermission
		for i := range wf.Groups {
			if wf.Groups[i].Group.Name == groupName {
				oldGp = wf.Groups[i]
				groupIndex = i
				break
			}
		}
		if oldGp.Permission == 0 {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err := group.DeleteWorkflowGroup(tx, wf, oldGp.Group.ID, groupIndex); err != nil {
			return sdk.WrapError(err, "cannot delete group")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		event.PublishWorkflowPermissionDelete(ctx, key, *wf, oldGp, u)

		log.Warn(ctx, "workflow %+v\n", wf)

		return service.WriteJSON(w, wf, http.StatusOK)
	}
}

// putWorkflowGroupHandler update permission for a group on the workflow
func (api *API) putWorkflowGroupHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		groupName := vars["groupName"]

		var gp sdk.GroupPermission
		if err := service.UnmarshalBody(r, &gp); err != nil {
			return sdk.WrapError(err, "putWorkflowGroupHandler")
		}
		if gp.Group.Name != groupName {
			return sdk.WrapError(sdk.ErrInvalidName, "putWorkflowGroupHandler")
		}

		proj, err := project.Load(ctx, api.mustDB(), key, project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WrapError(err, "unable to load projet")
		}

		wf, err := workflow.Load(ctx, api.mustDB(), api.Cache, *proj, name, workflow.LoadOptions{})
		if err != nil {
			return sdk.WithStack(err)
		}
		var oldGp sdk.GroupPermission
		for _, gpr := range wf.Groups {
			if gpr.Group.Name == gp.Group.Name {
				oldGp = gpr
				break
			}
		}
		if oldGp.Permission == 0 {
			return sdk.WrapError(sdk.ErrNotFound, "no permission found for group %q on workflow", gp.Group.Name)
		}

		g, err := group.LoadByName(ctx, api.mustDB(), gp.Group.Name, group.LoadOptions.WithOrganization)
		if err != nil {
			return sdk.WrapError(err, "cannot load group with name %q", gp.Group.Name)
		}
		gp.Group = *g

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err := projectPermissionCheckOrganizationMatch(ctx, tx, proj, &gp.Group, gp.Permission); err != nil {
			return err
		}

		if err := group.UpdateWorkflowGroup(ctx, tx, wf, gp); err != nil {
			return sdk.WrapError(err, "cannot add group")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		event.PublishWorkflowPermissionUpdate(ctx, key, *wf, gp, oldGp, getAPIConsumer(ctx))

		return service.WriteJSON(w, wf, http.StatusOK)
	}
}

// postWorkflowGroupHandler add permission for a group on the workflow
func (api *API) postWorkflowGroupHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		var gp sdk.GroupPermission
		if err := service.UnmarshalBody(r, &gp); err != nil {
			return sdk.WrapError(err, "cannot unmarshal body")
		}

		proj, err := project.Load(ctx, api.mustDB(), key, project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WrapError(err, "unable to load projet")
		}

		wf, err := workflow.Load(ctx, api.mustDB(), api.Cache, *proj, name, workflow.LoadOptions{})
		if err != nil {
			return sdk.WrapError(err, "cannot load workflow")
		}

		for _, gpr := range wf.Groups {
			if gpr.Group.Name == gp.Group.Name {
				return sdk.WrapError(sdk.ErrGroupPresent, "group is already present")
			}
		}

		g, err := group.LoadByName(ctx, api.mustDB(), gp.Group.Name, group.LoadOptions.WithOrganization)
		if err != nil {
			return sdk.WrapError(err, "cannot load group with name %q", gp.Group.Name)
		}
		gp.Group = *g

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		if err := projectPermissionCheckOrganizationMatch(ctx, tx, proj, &gp.Group, gp.Permission); err != nil {
			return err
		}

		if err := group.AddWorkflowGroup(ctx, tx, wf, gp); err != nil {
			return sdk.WrapError(err, "cannot add group")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WithStack(err)
		}

		event.PublishWorkflowPermissionAdd(ctx, key, *wf, gp, getAPIConsumer(ctx))

		return service.WriteJSON(w, wf, http.StatusOK)
	}
}
