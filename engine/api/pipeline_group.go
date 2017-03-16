package main

import (
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/sdk"
)

func updateGroupRoleOnPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	pipelineName := vars["permPipelineKey"]
	groupName := vars["group"]

	var groupPipeline sdk.GroupPermission
	if err := UnmarshalBody(r, &groupPipeline); err != nil {
		return sdk.WrapError(err, "updateGroupRoleOnPipelineHandler> cannot unmarshal request")
	}

	if groupName != groupPipeline.Group.Name {
		return sdk.ErrGroupNotFound
	}

	p, errLoadP := pipeline.LoadPipeline(db, key, pipelineName, false)
	if errLoadP != nil {
		return sdk.WrapError(sdk.ErrPipelineNotFound, "updateGroupRoleOnPipelineHandler: Cannot load %s: %s", key, errLoadP)
	}

	g, errLoadG := group.LoadGroup(db, groupPipeline.Group.Name)
	if errLoadG != nil {
		return sdk.WrapError(sdk.ErrGroupNotFound, "updateGroupRoleOnPipelineHandler: Cannot find %s: %s", groupPipeline.Group.Name, errLoadG)
	}

	groupInPipeline, errCheck := group.CheckGroupInPipeline(db, p.ID, g.ID)
	if errCheck != nil {
		return sdk.WrapError(sdk.ErrGroupNotFound, "updateGroupRoleOnPipelineHandler: Cannot check if group %s is already in the pipeline %s: %s", g.Name, p.Name, errCheck)
	}
	if !groupInPipeline {
		return sdk.WrapError(sdk.ErrGroupNotFound, "updateGroupRoleOnPipelineHandler: Cannot find group %s in pipeline %s", g.Name, p.Name)
	}

	if groupPipeline.Permission != permission.PermissionReadWriteExecute {
		permissions, err := group.LoadAllPipelineGroupByRole(db, p.ID, permission.PermissionReadWriteExecute)
		if err != nil {
			return sdk.WrapError(sdk.ErrGroupNeedWrite, "updateGroupRoleOnPipelineHandler: Cannot load groups for pipeline %s: %s", p.Name, err)
		}
		if len(permissions) == 1 && permissions[0].Group.ID == g.ID {
			return sdk.WrapError(sdk.ErrGroupNeedWrite, "updateGroupRoleOnPipelineHandler: Cannot remove write permission for group %s in pipeline %s", g.Name, p.Name)
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(sdk.ErrUnknownError, "updateGroupRoleOnPipelineHandler: Cannot start transaction: %s", err)
	}
	defer tx.Rollback()

	if err := group.UpdateGroupRoleInPipeline(tx, p.ID, g.ID, groupPipeline.Permission); err != nil {
		return sdk.WrapError(sdk.ErrUnknownError, "updateGroupRoleOnPipelineHandler: Cannot add group %s in pipeline %s:  %s", g.Name, p.Name, err)
	}

	if err := pipeline.UpdatePipelineLastModified(tx, p); err != nil {
		return sdk.WrapError(sdk.ErrUnknownError, "updateGroupRoleOnPipelineHandler: Cannot update pipeline last_modified date: %s", err)
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(sdk.ErrUnknownError, "updateGroupRoleOnPipelineHandler: Cannot start transaction: %s", err)
	}

	if err := pipeline.LoadGroupByPipeline(db, p); err != nil {
		return sdk.WrapError(sdk.ErrUnknownError, "updateGroupRoleOnPipelineHandler: Cannot load groups for pipeline %s: %s", p.Name, err)
	}
	return WriteJSON(w, r, p, http.StatusOK)
}

// DEPRECATED
func updateGroupsOnPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	pipelineName := vars["permPipelineKey"]

	var groupsPermission []sdk.GroupPermission
	if err := UnmarshalBody(r, &groupsPermission); err != nil {
		return err
	}

	if len(groupsPermission) == 0 {
		return sdk.WrapError(sdk.ErrGroupNeedWrite, "updateGroupsOnPipelineHandler: Cannot remove all groups for pipeline %s", pipelineName)
	}

	found := false
	for _, gp := range groupsPermission {
		if gp.Permission == permission.PermissionReadWriteExecute {
			found = true
			break
		}
	}
	if !found {
		return sdk.WrapError(sdk.ErrGroupNeedWrite, "updateGroupsOnPipelineHandler: Need one group with write permission.")
	}

	p, errLoad := pipeline.LoadPipeline(db, key, pipelineName, false)
	if errLoad != nil {
		return sdk.WrapError(sdk.ErrUnknownError, "updateGroupsOnPipelineHandler: Cannot load %s: %s", key, errLoad)
	}

	tx, errb := db.Begin()
	if errb != nil {
		return sdk.WrapError(sdk.ErrUnknownError, "updateGroupsOnPipelineHandler: Cannot start transaction: %s", errb)
	}
	defer tx.Rollback()

	if err := group.DeleteAllGroupFromPipeline(tx, p.ID); err != nil {
		return sdk.WrapError(sdk.ErrUnknownError, "updateGroupsOnPipelineHandler: Cannot delete groups from pipeline %s: %s", p.Name, err)
	}

	for _, g := range groupsPermission {
		groupData, err := group.LoadGroup(tx, g.Group.Name)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnknownError, "updateGroupsOnPipelineHandler: Cannot load group %s: %s", g.Group.Name, err)
		}

		if err := group.InsertGroupInPipeline(tx, p.ID, groupData.ID, g.Permission); err != nil {
			return sdk.WrapError(sdk.ErrUnknownError, "updateGroupsOnPipelineHandler: Cannot insert group %s in pipeline %s: %s", g.Group.Name, p.Name, err)
		}
	}

	if err := pipeline.UpdatePipelineLastModified(tx, p); err != nil {
		return sdk.WrapError(sdk.ErrUnknownError, "updateGroupsOnPipelineHandler: Cannot update pipeline last_modified date: %s", err)
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(sdk.ErrUnknownError, "updateGroupsOnPipelineHandler: Cannot commit transaction: %s", err)
	}

	return nil
}

func addGroupInPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	pipelineName := vars["permPipelineKey"]

	var groupPermission sdk.GroupPermission
	if err := UnmarshalBody(r, &groupPermission); err != nil {
		return err
	}

	p, err := pipeline.LoadPipeline(db, key, pipelineName, false)
	if err != nil {
		return sdk.WrapError(sdk.ErrPipelineNotFound, "addGroupInPipeline: Cannot load %s: %s", key, err)
	}

	g, err := group.LoadGroup(db, groupPermission.Group.Name)
	if err != nil {
		return sdk.WrapError(sdk.ErrGroupNotFound, "addGroupInPipeline: Cannot find %s: %s", groupPermission.Group.Name, err)
	}

	groupInPipeline, err := group.CheckGroupInPipeline(db, p.ID, g.ID)
	if err != nil {
		return sdk.WrapError(sdk.ErrGroupNotFound, "addGroupInPipeline: Cannot check if group %s is already in the pipeline %s: %s", g.Name, p.Name, err)

	}
	if groupInPipeline {
		return sdk.WrapError(sdk.ErrGroupExists, "addGroupInPipeline: The group is already attached to the pipeline %s: %s", g.Name, p.Name, err)
	}

	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(sdk.ErrUnknownError, "addGroupInPipeline: Cannot start transaction: %s", err)
	}
	defer tx.Rollback()

	if err := group.InsertGroupInPipeline(tx, p.ID, g.ID, groupPermission.Permission); err != nil {
		return sdk.WrapError(sdk.ErrUnknownError, "addGroupInPipeline: Cannot add group %s in pipeline %s:  %s", g.Name, p.Name, err)
	}

	if err := pipeline.UpdatePipelineLastModified(tx, p); err != nil {
		return sdk.WrapError(sdk.ErrUnknownError, "addGroupInPipeline: Cannot update pipeline last_modified date: %s", err)
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(sdk.ErrUnknownError, "addGroupInPipeline: Cannot commit transaction: %s", err)
	}

	if err := pipeline.LoadGroupByPipeline(db, p); err != nil {
		return sdk.WrapError(sdk.ErrUnknownError, "addGroupInPipeline: Cannot load group: %s", err)
	}
	return WriteJSON(w, r, p, http.StatusOK)
}

func deleteGroupFromPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {

	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	pipelineName := vars["permPipelineKey"]
	groupName := vars["group"]

	p, err := pipeline.LoadPipeline(db, key, pipelineName, false)
	if err != nil {
		return sdk.WrapError(sdk.ErrPipelineNotFound, "deleteGroupFromPipelineHandler: Cannot load %s: %s", key, err)
	}

	g, err := group.LoadGroup(db, groupName)
	if err != nil {
		return sdk.WrapError(sdk.ErrGroupNotFound, "deleteGroupFromPipelineHandler: Cannot find %s: %s", groupName, err)
	}

	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(sdk.ErrUnknownError, "deleteGroupFromPipelineHandler: Cannot start transaction: %s", err)
	}
	defer tx.Rollback()

	if err := group.DeleteGroupFromPipeline(tx, p.ID, g.ID); err != nil {
		return sdk.WrapError(sdk.ErrUnknownError, "deleteGroupFromPipelineHandler: Cannot delete group %s from project %s:  %s", g.Name, p.Name, err)
	}

	if err := pipeline.UpdatePipelineLastModified(tx, p); err != nil {
		return sdk.WrapError(sdk.ErrUnknownError, "deleteGroupFromPipelineHandler: Cannot update pipeline last_modified date: %s", err)
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(sdk.ErrUnknownError, "deleteGroupFromPipelineHandler: Cannot commit transaction: %s", err)
	}

	if err := pipeline.LoadGroupByPipeline(db, p); err != nil {
		return sdk.WrapError(sdk.ErrUnknownError, "deleteGroupFromPipelineHandler: Cannot load groups: %s", err)
	}

	return WriteJSON(w, r, p, http.StatusOK)
}
