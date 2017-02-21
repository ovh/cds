package main

import (
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/log"
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
		return err
	}

	if groupName != groupPipeline.Group.Name {
		return sdk.ErrGroupNotFound
	}

	p, err := pipeline.LoadPipeline(db, key, pipelineName, false)
	if err != nil {
		log.Warning("updateGroupRoleOnPipelineHandler: Cannot load %s: %s\n", key, err)
		return sdk.ErrPipelineNotFound
	}

	g, err := group.LoadGroup(db, groupPipeline.Group.Name)
	if err != nil {
		log.Warning("updateGroupRoleOnPipelineHandler: Cannot find %s: %s\n", groupPipeline.Group.Name, err)
		return err
	}

	groupInPipeline, err := group.CheckGroupInPipeline(db, p.ID, g.ID)
	if err != nil {
		log.Warning("updateGroupRoleOnPipelineHandler: Cannot check if group %s is already in the pipeline %s: %s\n", g.Name, p.Name, err)
		return err
	}
	if !groupInPipeline {
		log.Warning("updateGroupRoleOnPipelineHandler: Cannot find group %s in pipeline %s: %s\n", g.Name, p.Name, err)
		return sdk.ErrGroupNotFound
	}

	if groupPipeline.Permission != permission.PermissionReadWriteExecute {
		permissions, err := group.LoadAllPipelineGroupByRole(db, p.ID, permission.PermissionReadWriteExecute)
		if err != nil {
			log.Warning("updateGroupRoleOnPipelineHandler: Cannot load groups for pipeline %s: %s\n", p.Name, err)
			return err
		}
		if len(permissions) == 1 && permissions[0].Group.ID == g.ID {
			log.Warning("updateGroupRoleOnPipelineHandler: Cannot remove write permission for group %s in pipeline %s\n", g.Name, p.Name)
			return sdk.ErrGroupNeedWrite
		}
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateGroupRoleOnPipelineHandler: Cannot start transaction: %s\n", err)
		return err
	}
	defer tx.Rollback()

	if err := group.UpdateGroupRoleInPipeline(tx, p.ID, g.ID, groupPipeline.Permission); err != nil {
		log.Warning("updateGroupRoleOnPipelineHandler: Cannot add group %s in pipeline %s:  %s\n", g.Name, p.Name, err)
		return err
	}

	if err := pipeline.UpdatePipelineLastModified(tx, p); err != nil {
		log.Warning("updateGroupRoleOnPipelineHandler: Cannot update pipeline last_modified date: %s\n", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("updateGroupRoleOnPipelineHandler: Cannot start transaction: %s\n", err)
		return err
	}

	if err := pipeline.LoadGroupByPipeline(db, p); err != nil {
		log.Warning("updateGroupRoleOnPipelineHandler: Cannot load groups for pipeline %s: %s\n", p.Name, err)
		return err
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
		log.Warning("updateGroupsOnPipelineHandler: Cannot remove all groups for pipeline %s", pipelineName)
		return sdk.ErrGroupNeedWrite
	}

	found := false
	for _, gp := range groupsPermission {
		if gp.Permission == permission.PermissionReadWriteExecute {
			found = true
			break
		}
	}
	if !found {
		log.Warning("updateGroupsOnPipelineHandler: Need one group with write permission.")
		return sdk.ErrGroupNeedWrite
	}

	p, err := pipeline.LoadPipeline(db, key, pipelineName, false)
	if err != nil {
		log.Warning("updateGroupsOnPipelineHandler: Cannot load %s: %s\n", key, err)
		return sdk.ErrUnknownError
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateGroupsOnPipelineHandler: Cannot start transaction: %s\n", err)
		return sdk.ErrUnknownError
	}
	defer tx.Rollback()

	err = group.DeleteAllGroupFromPipeline(tx, p.ID)
	if err != nil {
		log.Warning("updateGroupsOnPipelineHandler: Cannot delete groups from pipeline %s: %s\n", p.Name, err)
		return sdk.ErrUnknownError
	}

	for _, g := range groupsPermission {
		groupData, err := group.LoadGroup(tx, g.Group.Name)
		if err != nil {
			log.Warning("updateGroupsOnPipelineHandler: Cannot load group %s: %s\n", g.Group.Name, err)
			return sdk.ErrUnknownError
		}
		err = group.InsertGroupInPipeline(tx, p.ID, groupData.ID, g.Permission)
		if err != nil {
			log.Warning("updateGroupsOnPipelineHandler: Cannot insert group %s in pipeline %s: %s\n", g.Group.Name, p.Name, err)
			return sdk.ErrUnknownError
		}
	}

	err = pipeline.UpdatePipelineLastModified(tx, p)
	if err != nil {
		log.Warning("updateGroupsOnPipelineHandler: Cannot update pipeline last_modified date: %s\n", err)
		return sdk.ErrUnknownError
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("updateGroupsOnPipelineHandler: Cannot commit transaction: %s\n", err)
		return sdk.ErrUnknownError
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
		log.Warning("addGroupInPipeline: Cannot load %s: %s\n", key, err)
		return err
	}

	g, err := group.LoadGroup(db, groupPermission.Group.Name)
	if err != nil {
		log.Warning("addGroupInPipeline: Cannot find %s: %s\n", groupPermission.Group.Name, err)
		return err
	}

	groupInPipeline, err := group.CheckGroupInPipeline(db, p.ID, g.ID)
	if err != nil {
		log.Warning("addGroupInPipeline: Cannot check if group %s is already in the pipeline %s: %s\n", g.Name, p.Name, err)
		return err

	}
	if groupInPipeline {
		log.Warning("addGroupInPipeline: The group is already attached to the pipeline %s: %s\n", g.Name, p.Name, err)
		return sdk.ErrGroupExists

	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("addGroupInPipeline: Cannot start transaction: %s\n", err)
		return err

	}
	defer tx.Rollback()

	if err := group.InsertGroupInPipeline(tx, p.ID, g.ID, groupPermission.Permission); err != nil {
		log.Warning("addGroupInPipeline: Cannot add group %s in pipeline %s:  %s\n", g.Name, p.Name, err)
		return err

	}

	if err := pipeline.UpdatePipelineLastModified(tx, p); err != nil {
		log.Warning("addGroupInPipeline: Cannot update pipeline last_modified date: %s\n", err)
		return err

	}

	if err := tx.Commit(); err != nil {
		log.Warning("addGroupInPipeline: Cannot commit transaction: %s\n", err)
		return err

	}

	if err := pipeline.LoadGroupByPipeline(db, p); err != nil {
		log.Warning("addGroupInPipeline: Cannot load group: %s\n", err)
		return err

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
		log.Warning("deleteGroupFromPipelineHandler: Cannot load %s: %s\n", key, err)
		return err

	}

	g, err := group.LoadGroup(db, groupName)
	if err != nil {
		log.Warning("deleteGroupFromPipelineHandler: Cannot find %s: %s\n", groupName, err)
		return err

	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteGroupFromPipelineHandler: Cannot start transaction: %s\n", err)
		return err

	}
	defer tx.Rollback()

	if err := group.DeleteGroupFromPipeline(tx, p.ID, g.ID); err != nil {
		log.Warning("deleteGroupFromPipelineHandler: Cannot delete group %s from project %s:  %s\n", g.Name, p.Name, err)
		return err

	}

	if err := pipeline.UpdatePipelineLastModified(tx, p); err != nil {
		log.Warning("deleteGroupFromPipelineHandler: Cannot update pipeline last_modified date: %s\n", err)
		return err

	}

	if err := tx.Commit(); err != nil {
		log.Warning("deleteGroupFromPipelineHandler: Cannot commit transaction: %s\n", err)
		return err

	}

	if err := pipeline.LoadGroupByPipeline(db, p); err != nil {
		log.Warning("deleteGroupFromPipelineHandler: Cannot load groups: %s\n", err)
		return err

	}

	return WriteJSON(w, r, p, http.StatusOK)
}
