package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// deleteWorkflowGroupHandler delete permission for a group on the workflow
func (api *API) deleteWorkflowGroupHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		groupName := vars["groupName"]
		u := getAPIConsumer(ctx)

		proj, err := project.Load(api.mustDB(), api.Cache, key, project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WrapError(err, "unable to load projet")
		}

		options := workflow.LoadOptions{}
		wf, err := workflow.Load(ctx, api.mustDB(), api.Cache, proj, name, options)
		if err != nil {
			return sdk.WrapError(err, "deleteWorkflowGroupHandler")
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
			return sdk.ErrNotFound
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "cannot start transaction")
		}
		defer tx.Rollback()

		if err := group.DeleteWorkflowGroup(tx, wf, oldGp.Group.ID, groupIndex); err != nil {
			return sdk.WrapError(err, "cannot delete group")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "cannot commit transaction")
		}

		event.PublishWorkflowPermissionDelete(key, *wf, oldGp, u)

		log.Warning("workflow %+v\n", wf)

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

		proj, err := project.Load(api.mustDB(), api.Cache, key, project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WrapError(err, "unable to load projet")
		}

		options := workflow.LoadOptions{}
		wf, err := workflow.Load(ctx, api.mustDB(), api.Cache, proj, name, options)
		if err != nil {
			return sdk.WrapError(err, "putWorkflowGroupHandler")
		}

		var oldGp sdk.GroupPermission
		for _, gpr := range wf.Groups {
			if gpr.Group.Name == gp.Group.Name {
				oldGp = gpr
				break
			}
		}

		if oldGp.Permission == 0 {
			return sdk.WrapError(sdk.ErrNotFound, "putWorkflowGroupHandler")
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "putWorkflowGroupHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := group.UpdateWorkflowGroup(ctx, tx, wf, gp); err != nil {
			return sdk.WrapError(err, "Cannot add group")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		event.PublishWorkflowPermissionUpdate(key, *wf, gp, oldGp, getAPIConsumer(ctx))

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
			return sdk.WrapError(err, "postWorkflowGroupHandler")
		}

		proj, err := project.Load(api.mustDB(), api.Cache, key, project.LoadOptions.WithIntegrations)
		if err != nil {
			return sdk.WrapError(err, "unable to load projet")
		}

		options := workflow.LoadOptions{}
		wf, err := workflow.Load(ctx, api.mustDB(), api.Cache, proj, name, options)
		if err != nil {
			return sdk.WrapError(err, "postWorkflowGroupHandler")
		}

		for _, gpr := range wf.Groups {
			if gpr.Group.Name == gp.Group.Name {
				return sdk.WrapError(sdk.ErrGroupPresent, "postWorkflowGroupHandler")
			}
		}

		if gp.Group.ID == 0 {
			g, errG := group.LoadByName(ctx, api.mustDB(), gp.Group.Name)
			if errG != nil {
				return sdk.WrapError(errG, "postWorkflowGroupHandler")
			}
			gp.Group = *g
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback()

		if err := group.AddWorkflowGroup(ctx, tx, wf, gp); err != nil {
			return sdk.WrapError(err, "cannot add group")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "cannot commit transaction")
		}

		event.PublishWorkflowPermissionAdd(key, *wf, gp, getAPIConsumer(ctx))

		return service.WriteJSON(w, wf, http.StatusOK)
	}
}
