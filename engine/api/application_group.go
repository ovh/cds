package main

import (
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
)

func updateGroupRoleOnApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	groupName := vars["group"]

	var groupApplication sdk.GroupPermission
	if err := UnmarshalBody(r, &groupApplication); err != nil {
		return err
	}

	app, errload := application.LoadByName(db, key, appName, c.User)
	if errload != nil {
		return sdk.WrapError(errload, "updateGroupRoleOnApplicationHandler: Cannot load application %s", appName)
	}

	g, errLoadGroup := group.LoadGroup(db, groupName)
	if errLoadGroup != nil {
		return sdk.WrapError(errLoadGroup, "updateGroupRoleOnApplicationHandler: Cannot load group %s", groupName)
	}

	if groupApplication.Permission != permission.PermissionReadWriteExecute {
		permissions, err := group.LoadAllApplicationGroupByRole(db, app.ID, permission.PermissionReadWriteExecute)
		if err != nil {
			return sdk.WrapError(err, "updateGroupRoleOnApplicationHandler: Cannot load group for application %s", appName)
		}

		if len(permissions) == 1 && permissions[0].Group.ID == g.ID {
			return sdk.WrapError(sdk.ErrGroupNeedWrite, "updateGroupRoleOnApplicationHandler: Cannot remove write permission for group %s in application %s", groupName, appName)
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "updateGroupRoleOnApplicationHandler: Cannot start transaction")
	}
	defer tx.Rollback()

	if err := group.UpdateGroupRoleInApplication(tx, key, appName, groupName, groupApplication.Permission); err != nil {
		return sdk.WrapError(err, "updateGroupRoleOnApplicationHandler: Cannot update permission for group %s in application %s", groupName, appName)
	}

	if err := application.UpdateLastModified(tx, app, c.User); err != nil {
		return sdk.WrapError(err, "updateGroupsInApplicationHandler: Cannot update last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "updateGroupRoleOnApplicationHandler: Cannot commit transaction")
	}

	cache.DeleteAll(cache.Key("application", key, "*"+appName+"*"))

	if err := application.LoadGroupByApplication(db, app); err != nil {
		return sdk.WrapError(err, "updateGroupRoleOnApplicationHandler: Cannot load application groups")
	}

	return WriteJSON(w, r, app, http.StatusOK)
}

// Deprecated
func updateGroupsInApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]

	proj, errload := project.Load(db, key, c.User)
	if errload != nil {
		return sdk.WrapError(errload, "addGroupInApplicationHandler> Cannot load %s", key)
	}

	var groupsPermission []sdk.GroupPermission
	if err := UnmarshalBody(r, &groupsPermission); err != nil {
		return err
	}

	if len(groupsPermission) == 0 {
		return sdk.WrapError(sdk.ErrGroupNeedWrite, "updateGroupsInApplicationHandler: Cannot remove all groups for application %s", appName)
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

	app, errLoadName := application.LoadByName(db, key, appName, c.User)
	if errLoadName != nil {
		return sdk.WrapError(errLoadName, "updateGroupsInApplicationHandler: Cannot load application %s: %s", appName)
	}

	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "updateGroupsInApplicationHandler: Cannot start transaction")
	}
	defer tx.Rollback()

	if err := group.DeleteAllGroupFromApplication(tx, app.ID); err != nil {
		return sdk.WrapError(err, "updateGroupsInApplicationHandler: Cannot delete groups from application %s", appName)
	}

	if err := application.AddGroup(tx, proj, app, groupsPermission...); err != nil {
		return sdk.WrapError(err, "updateGroupsInApplicationHandler: Cannot add groups in application %s", app.Name)
	}

	if err := application.UpdateLastModified(tx, app, c.User); err != nil {
		return sdk.WrapError(err, "updateGroupsInApplicationHandler: Cannot update last modified date")
	}

	if err := project.UpdateLastModified(tx, c.User, proj); err != nil {
		return sdk.WrapError(err, "updateGroupsInApplicationHandler: Cannot update last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "updateGroupsInApplicationHandler: Cannot commit transaction")
	}

	cache.DeleteAll(cache.Key("application", key, "*"+appName+"*"))
	return WriteJSON(w, r, app, http.StatusOK)
}

func addGroupInApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]

	var groupPermission sdk.GroupPermission
	if err := UnmarshalBody(r, &groupPermission); err != nil {
		return sdk.WrapError(err, "addGroupInApplicationHandler> Cannot unmarshal request")
	}

	proj, err := project.Load(db, key, c.User)
	if err != nil {
		return sdk.WrapError(err, "addGroupInApplicationHandler> Cannot load %s", key)
	}

	app, err := application.LoadByName(db, key, appName, c.User)
	if err != nil {
		return sdk.WrapError(err, "addGroupInApplicationHandler> Cannot load %s", appName)
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

	if err := application.AddGroup(tx, proj, app, groupPermission); err != nil {
		return sdk.WrapError(err, "addGroupInApplicationHandler> Cannot add group %s in application %s", g.Name, app.Name)
	}

	if err := application.UpdateLastModified(tx, app, c.User); err != nil {
		return sdk.WrapError(err, "addGroupInApplicationHandler> Cannot update application last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "addGroupInApplicationHandler> Cannot commit transaction")
	}

	cache.DeleteAll(cache.Key("application", key, "*"+appName+"*"))

	if err := application.LoadGroupByApplication(db, app); err != nil {
		return sdk.WrapError(err, "addGroupInApplicationHandler> Cannot load application groups")
	}

	return WriteJSON(w, r, app, http.StatusOK)
}

func deleteGroupFromApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	groupName := vars["group"]

	app, err := application.LoadByName(db, key, appName, c.User)
	if err != nil {
		return sdk.WrapError(err, "deleteGroupFromApplicationHandler: Cannot load application %s", appName)
	}

	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "deleteGroupFromApplicationHandler: Cannot start transaction")
	}
	defer tx.Rollback()

	if err := group.DeleteGroupFromApplication(tx, key, appName, groupName); err != nil {
		return sdk.WrapError(err, "deleteGroupFromApplicationHandler: Cannot delete group %s from pipeline %s", groupName, appName)
	}

	if err := application.UpdateLastModified(tx, app, c.User); err != nil {
		return sdk.WrapError(err, "deleteGroupFromApplicationHandler: Cannot update application last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "deleteGroupFromApplicationHandler: Cannot commit transaction")
	}

	cache.DeleteAll(cache.Key("application", key, "*"+appName+"*"))

	if err := application.LoadGroupByApplication(db, app); err != nil {
		return sdk.WrapError(err, "deleteGroupFromApplicationHandler: Cannot load application groups")
	}

	return WriteJSON(w, r, app, http.StatusOK)
}
