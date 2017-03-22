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
		return sdk.WrapError(errLoadP, "updateGroupRoleOnPipelineHandler: Cannot load %s", key)
	}

	g, errLoadG := group.LoadGroup(db, groupPipeline.Group.Name)
	if errLoadG != nil {
		return sdk.WrapError(errLoadG, "updateGroupRoleOnPipelineHandler: Cannot find %s", groupPipeline.Group.Name)
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
		return sdk.WrapError(err, "updateGroupRoleOnPipelineHandler: Cannot start transaction")
	}
	defer tx.Rollback()

	if err := group.UpdateGroupRoleInPipeline(tx, p.ID, g.ID, groupPipeline.Permission); err != nil {
		return sdk.WrapError(err, "updateGroupRoleOnPipelineHandler: Cannot add group %s in pipeline %s", g.Name, p.Name)
	}

	if err := pipeline.UpdatePipelineLastModified(tx, p); err != nil {
		return sdk.WrapError(err, "updateGroupRoleOnPipelineHandler: Cannot update pipeline last_modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "updateGroupRoleOnPipelineHandler: Cannot start transaction")
	}

	if err := pipeline.LoadGroupByPipeline(db, p); err != nil {
		return sdk.WrapError(err, "updateGroupRoleOnPipelineHandler: Cannot load groups for pipeline %s", p.Name)
	}
	return WriteJSON(w, r, p, http.StatusOK)
}

// DEPRECATED
func updateGroupsOnPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
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
		return sdk.WrapError(errLoad, "updateGroupsOnPipelineHandler: Cannot load %s", key)
	}

	tx, errb := db.Begin()
	if errb != nil {
		return sdk.WrapError(errb, "updateGroupsOnPipelineHandler: Cannot start transaction")
	}
	defer tx.Rollback()

	if err := group.DeleteAllGroupFromPipeline(tx, p.ID); err != nil {
		return sdk.WrapError(err, "updateGroupsOnPipelineHandler: Cannot delete groups from pipeline %s", p.Name)
	}

	for _, g := range groupsPermission {
		groupData, err := group.LoadGroup(tx, g.Group.Name)
		if err != nil {
			return sdk.WrapError(err, "updateGroupsOnPipelineHandler: Cannot load group %s", g.Group.Name)
		}

		if err := group.InsertGroupInPipeline(tx, p.ID, groupData.ID, g.Permission); err != nil {
			return sdk.WrapError(err, "updateGroupsOnPipelineHandler: Cannot insert group %s in pipeline %s", g.Group.Name, p.Name)
		}
	}

	if err := pipeline.UpdatePipelineLastModified(tx, p); err != nil {
		return sdk.WrapError(err, "updateGroupsOnPipelineHandler: Cannot update pipeline last_modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "updateGroupsOnPipelineHandler: Cannot commit transaction")
	}

	return nil
}

func addGroupInPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	pipelineName := vars["permPipelineKey"]

	var groupPermission sdk.GroupPermission
	if err := UnmarshalBody(r, &groupPermission); err != nil {
		return err
	}

	p, err := pipeline.LoadPipeline(db, key, pipelineName, false)
	if err != nil {
		return sdk.WrapError(err, "addGroupInPipeline: Cannot load %s", key)
	}

	g, err := group.LoadGroup(db, groupPermission.Group.Name)
	if err != nil {
		return sdk.WrapError(err, "addGroupInPipeline: Cannot find %s", groupPermission.Group.Name)
	}

	groupInPipeline, err := group.CheckGroupInPipeline(db, p.ID, g.ID)
	if err != nil {
		return sdk.WrapError(err, "addGroupInPipeline: Cannot check if group %s is already in the pipeline %s", g.Name, p.Name)

	}
	if groupInPipeline {
		return sdk.WrapError(sdk.ErrGroupExists, "addGroupInPipeline: The group is already attached to the pipeline %s: %s", g.Name, p.Name, err)
	}

	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "addGroupInPipeline: Cannot start transaction")
	}
	defer tx.Rollback()

	if err := group.InsertGroupInPipeline(tx, p.ID, g.ID, groupPermission.Permission); err != nil {
		return sdk.WrapError(err, "addGroupInPipeline: Cannot add group %s in pipeline %s", g.Name, p.Name)
	}

	if err := pipeline.UpdatePipelineLastModified(tx, p); err != nil {
		return sdk.WrapError(err, "addGroupInPipeline: Cannot update pipeline last_modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "addGroupInPipeline: Cannot commit transaction")
	}

	if err := pipeline.LoadGroupByPipeline(db, p); err != nil {
		return sdk.WrapError(err, "addGroupInPipeline: Cannot load group")
	}
	return WriteJSON(w, r, p, http.StatusOK)
}

func deleteGroupFromPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	pipelineName := vars["permPipelineKey"]
	groupName := vars["group"]

	p, err := pipeline.LoadPipeline(db, key, pipelineName, false)
	if err != nil {
		return sdk.WrapError(err, "deleteGroupFromPipelineHandler: Cannot load %s", key)
	}

	g, err := group.LoadGroup(db, groupName)
	if err != nil {
		return sdk.WrapError(err, "deleteGroupFromPipelineHandler: Cannot find %s", groupName)
	}

	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "deleteGroupFromPipelineHandler: Cannot start transaction")
	}
	defer tx.Rollback()

	if err := group.DeleteGroupFromPipeline(tx, p.ID, g.ID); err != nil {
		return sdk.WrapError(err, "deleteGroupFromPipelineHandler: Cannot delete group %s from project %s", g.Name, p.Name)
	}

	if err := pipeline.UpdatePipelineLastModified(tx, p); err != nil {
		return sdk.WrapError(err, "deleteGroupFromPipelineHandler: Cannot update pipeline last_modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "deleteGroupFromPipelineHandler: Cannot commit transaction")
	}

	if err := pipeline.LoadGroupByPipeline(db, p); err != nil {
		return sdk.WrapError(err, "deleteGroupFromPipelineHandler: Cannot load groups")
	}

	return WriteJSON(w, r, p, http.StatusOK)
}
