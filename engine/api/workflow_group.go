package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

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

		wf, err := workflow.Load(api.mustDB(), api.Cache, key, name, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "deleteWorkflowGroupHandler")
		}

		var groupID int64
		var groupIndex int
		for i := range wf.Groups {
			if wf.Groups[i].Group.Name == groupName {
				groupID = wf.Groups[i].Group.ID
				groupIndex = i
			}
		}

		if groupID == 0 {
			return sdk.WrapError(sdk.ErrGroupNotFound, "deleteWorkflowGroupHandler")
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "deleteWorkflowGroupHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := workflow.DeleteGroup(tx, wf, groupID, groupIndex); err != nil {
			return sdk.WrapError(err, "deleteWorkflowGroupHandler> Cannot add group")
		}

		if err := workflow.UpdateLastModifiedDate(tx, api.Cache, getUser(ctx), key, wf); err != nil {
			return sdk.WrapError(err, "deleteWorkflowGroupHandler> Cannot update workflow last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteWorkflowGroupHandler> Cannot commit transaction")
		}

		return WriteJSON(w, r, wf, http.StatusOK)
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

		wf, err := workflow.Load(api.mustDB(), api.Cache, key, name, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "putWorkflowGroupHandler")
		}

		found := false
		for _, gpr := range wf.Groups {
			if gpr.Group.Name == gp.Group.Name {
				found = true
			}
		}

		if !found {
			return sdk.WrapError(sdk.ErrGroupNotFound, "putWorkflowGroupHandler")
		}

		tx, errT := api.mustDB().Begin()
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

		return WriteJSON(w, r, wf, http.StatusOK)
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

		wf, err := workflow.Load(api.mustDB(), api.Cache, key, name, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "postWorkflowGroupHandler")
		}

		for _, gpr := range wf.Groups {
			if gpr.Group.Name == gp.Group.Name {
				return sdk.WrapError(sdk.ErrGroupPresent, "postWorkflowGroupHandler")
			}
		}

		if gp.Group.ID == 0 {
			g, errG := group.LoadGroup(api.mustDB(), gp.Group.Name)
			if errG != nil {
				return sdk.WrapError(errG, "postWorkflowGroupHandler")
			}
			gp.Group = *g
		}

		tx, errT := api.mustDB().Begin()
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

		return WriteJSON(w, r, wf, http.StatusOK)
	}
}
