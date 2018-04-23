package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

// deleteWorkflowGroupHandler delete permission for a group on the workflow
func (api *API) deleteWorkflowGroupHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		groupName := vars["groupName"]

		options := workflow.LoadOptions{
			WithoutNode: true,
		}
		wf, err := workflow.Load(api.mustDB(ctx), api.Cache, key, name, getUser(ctx), options)
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
			return sdk.WrapError(sdk.ErrGroupNotFound, "deleteWorkflowGroupHandler")
		}

		tx, errT := api.mustDB(ctx).Begin()
		if errT != nil {
			return sdk.WrapError(errT, "deleteWorkflowGroupHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := workflow.DeleteGroup(tx, wf, oldGp.Group.ID, groupIndex); err != nil {
			return sdk.WrapError(err, "deleteWorkflowGroupHandler> Cannot add group")
		}

		if err := workflow.UpdateLastModifiedDate(tx, api.Cache, getUser(ctx), key, wf); err != nil {
			return sdk.WrapError(err, "deleteWorkflowGroupHandler> Cannot update workflow last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteWorkflowGroupHandler> Cannot commit transaction")
		}

		event.PublishWorkflowPermissionDelete(key, *wf, oldGp, getUser(ctx))

		return WriteJSON(w, wf, http.StatusOK)
	}
}

// putWorkflowGroupHandler update permission for a group on the workflow
func (api *API) putWorkflowGroupHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]
		groupName := vars["groupName"]

		var gp sdk.GroupPermission
		if err := UnmarshalBody(r, &gp); err != nil {
			return sdk.WrapError(err, "putWorkflowGroupHandler")
		}

		if gp.Group.Name != groupName {
			return sdk.WrapError(sdk.ErrInvalidName, "putWorkflowGroupHandler")
		}

		options := workflow.LoadOptions{
			WithoutNode: true,
		}
		wf, err := workflow.Load(api.mustDB(ctx), api.Cache, key, name, getUser(ctx), options)
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
			return sdk.WrapError(sdk.ErrGroupNotFound, "putWorkflowGroupHandler")
		}

		tx, errT := api.mustDB(ctx).Begin()
		if errT != nil {
			return sdk.WrapError(errT, "putWorkflowGroupHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := workflow.UpdateGroup(tx, wf, gp); err != nil {
			return sdk.WrapError(err, "putWorkflowGroupHandler> Cannot add group")
		}

		if err := workflow.UpdateLastModifiedDate(tx, api.Cache, getUser(ctx), key, wf); err != nil {
			return sdk.WrapError(err, "putWorkflowGroupHandler> Cannot update workflow last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "putWorkflowGroupHandler> Cannot commit transaction")
		}

		event.PublishWorkflowPermissionUpdate(key, *wf, gp, oldGp, getUser(ctx))

		return WriteJSON(w, wf, http.StatusOK)
	}
}

// postWorkflowGroupHandler add permission for a group on the workflow
func (api *API) postWorkflowGroupHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		name := vars["permWorkflowName"]

		var gp sdk.GroupPermission
		if err := UnmarshalBody(r, &gp); err != nil {
			return sdk.WrapError(err, "postWorkflowGroupHandler")
		}

		options := workflow.LoadOptions{
			WithoutNode: true,
		}
		wf, err := workflow.Load(api.mustDB(ctx), api.Cache, key, name, getUser(ctx), options)
		if err != nil {
			return sdk.WrapError(err, "postWorkflowGroupHandler")
		}

		for _, gpr := range wf.Groups {
			if gpr.Group.Name == gp.Group.Name {
				return sdk.WrapError(sdk.ErrGroupPresent, "postWorkflowGroupHandler")
			}
		}

		if gp.Group.ID == 0 {
			g, errG := group.LoadGroup(api.mustDB(ctx), gp.Group.Name)
			if errG != nil {
				return sdk.WrapError(errG, "postWorkflowGroupHandler")
			}
			gp.Group = *g
		}

		tx, errT := api.mustDB(ctx).Begin()
		if errT != nil {
			return sdk.WrapError(errT, "postWorkflowGroupHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := workflow.AddGroup(tx, wf, gp); err != nil {
			return sdk.WrapError(err, "postWorkflowGroupHandler> Cannot add group")
		}

		if err := workflow.UpdateLastModifiedDate(tx, api.Cache, getUser(ctx), key, wf); err != nil {
			return sdk.WrapError(err, "postWorkflowGroupHandler> Cannot update workflow last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "postWorkflowGroupHandler> Cannot commit transaction")
		}

		event.PublishWorkflowPermissionAdd(key, *wf, gp, getUser(ctx))

		return WriteJSON(w, wf, http.StatusOK)
	}
}
