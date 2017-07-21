package main

import (
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
)

func updateGroupRoleOnApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	groupName := vars["group"]

	var groupApplication sdk.GroupPermission
	if err := UnmarshalBody(r, &groupApplication); err != nil {
		return err
	}

	g, errLoadGroup := group.LoadGroup(db, groupName)
	if errLoadGroup != nil {
		return sdk.WrapError(errLoadGroup, "updateGroupRoleOnApplicationHandler: Cannot load group %s", groupName)
	}

	if groupApplication.Permission != permission.PermissionReadWriteExecute {
		permissions, err := group.LoadAllApplicationGroupByRole(db, c.Application.ID, permission.PermissionReadWriteExecute)
		if err != nil {
			return sdk.WrapError(err, "updateGroupRoleOnApplicationHandler: Cannot load group for application %s", c.Application.Name)
		}

		if len(permissions) == 1 && permissions[0].Group.ID == g.ID {
			return sdk.WrapError(sdk.ErrGroupNeedWrite, "updateGroupRoleOnApplicationHandler: Cannot remove write permission for group %s in application %s", groupName, c.Application.Name)
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "updateGroupRoleOnApplicationHandler: Cannot start transaction")
	}
	defer tx.Rollback()

	if err := group.UpdateGroupRoleInApplication(tx, c.Project.Key, c.Application.Name, groupName, groupApplication.Permission); err != nil {
		return sdk.WrapError(err, "updateGroupRoleOnApplicationHandler: Cannot update permission for group %s in application %s", groupName, c.Application.Name)
	}

	if err := application.UpdateLastModified(tx, c.Application, c.User); err != nil {
		return sdk.WrapError(err, "updateGroupsInApplicationHandler: Cannot update last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "updateGroupRoleOnApplicationHandler: Cannot commit transaction")
	}

	if err := application.LoadGroupByApplication(db, c.Application); err != nil {
		return sdk.WrapError(err, "updateGroupRoleOnApplicationHandler: Cannot load application groups")
	}

	return WriteJSON(w, r, c.Application.Name, http.StatusOK)
}

// Deprecated
func updateGroupsInApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	var groupsPermission []sdk.GroupPermission
	if err := UnmarshalBody(r, &groupsPermission); err != nil {
		return err
	}

	if len(groupsPermission) == 0 {
		return sdk.WrapError(sdk.ErrGroupNeedWrite, "updateGroupsInApplicationHandler: Cannot remove all groups for application")
	}

	found := false
	for _, gp := range groupsPermission {
		if gp.Permission == permission.PermissionReadWriteExecute {
			found = true
			break
		}
	}
	if !found {
		return sdk.WrapError(sdk.ErrGroupNeedWrite, "updateGroupsInApplicationHandler: Need one group with write permission.")
	}

	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "updateGroupsInApplicationHandler: Cannot start transaction")
	}
	defer tx.Rollback()

	if err := group.DeleteAllGroupFromApplication(tx, c.Application.ID); err != nil {
		return sdk.WrapError(err, "updateGroupsInApplicationHandler: Cannot delete groups from application")
	}

	if err := application.AddGroup(tx, c.Project, c.Application, c.User, groupsPermission...); err != nil {
		return sdk.WrapError(err, "updateGroupsInApplicationHandler: Cannot add groups in application")
	}

	if err := application.UpdateLastModified(tx, c.Application, c.User); err != nil {
		return sdk.WrapError(err, "updateGroupsInApplicationHandler: Cannot update last modified date")
	}

	if err := project.UpdateLastModified(tx, c.User, c.Project); err != nil {
		return sdk.WrapError(err, "updateGroupsInApplicationHandler: Cannot update last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "updateGroupsInApplicationHandler: Cannot commit transaction")
	}

	return WriteJSON(w, r, c.Application, http.StatusOK)
}

func addGroupInApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	var groupPermission sdk.GroupPermission
	if err := UnmarshalBody(r, &groupPermission); err != nil {
		return sdk.WrapError(err, "addGroupInApplicationHandler> Cannot unmarshal request")
	}

	g, err := group.LoadGroup(db, groupPermission.Group.Name)
	if err != nil {
		return sdk.WrapError(err, "addGroupInApplicationHandler> Cannot find %s", groupPermission.Group.Name)
	}

	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "addGroupInApplicationHandler> Cannot start transaction")
	}
	defer tx.Rollback()

	if err := application.AddGroup(tx, c.Project, c.Application, c.User, groupPermission); err != nil {
		return sdk.WrapError(err, "addGroupInApplicationHandler> Cannot add group %s in application %s", g.Name, c.Application.Name)
	}

	if err := application.UpdateLastModified(tx, c.Application, c.User); err != nil {
		return sdk.WrapError(err, "addGroupInApplicationHandler> Cannot update application last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "addGroupInApplicationHandler> Cannot commit transaction")
	}

	if err := application.LoadGroupByApplication(db, c.Application); err != nil {
		return sdk.WrapError(err, "addGroupInApplicationHandler> Cannot load application groups")
	}

	return WriteJSON(w, r, c.Application, http.StatusOK)
}

func deleteGroupFromApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	groupName := vars["group"]

	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "deleteGroupFromApplicationHandler: Cannot start transaction")
	}
	defer tx.Rollback()

	if err := group.DeleteGroupFromApplication(tx, c.Project.Key, c.Application.Name, groupName); err != nil {
		return sdk.WrapError(err, "deleteGroupFromApplicationHandler: Cannot delete group %s from pipeline %s", groupName, c.Application.Name)
	}

	if err := application.UpdateLastModified(tx, c.Application, c.User); err != nil {
		return sdk.WrapError(err, "deleteGroupFromApplicationHandler: Cannot update application last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "deleteGroupFromApplicationHandler: Cannot commit transaction")
	}

	if err := application.LoadGroupByApplication(db, c.Application); err != nil {
		return sdk.WrapError(err, "deleteGroupFromApplicationHandler: Cannot load application groups")
	}

	return WriteJSON(w, r, c.Application.Name, http.StatusOK)
}
