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
	"github.com/ovh/cds/engine/log"
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

	app, err := application.LoadByName(db, key, appName, c.User)
	if err != nil {
		log.Warning("updateGroupRoleOnApplicationHandler: Cannot load application %s :%s", appName, err)
		return sdk.ErrApplicationNotFound
	}

	g, err := group.LoadGroup(db, groupName)
	if err != nil {
		log.Warning("updateGroupRoleOnApplicationHandler: Cannot load group %s :%s", groupName, err)
		return sdk.ErrGroupNotFound
	}

	if groupApplication.Permission != permission.PermissionReadWriteExecute {
		permissions, err := group.LoadAllApplicationGroupByRole(db, app.ID, permission.PermissionReadWriteExecute)
		if err != nil {
			log.Warning("updateGroupRoleOnApplicationHandler: Cannot load group for application %s:  %s\n", appName, err)
			return err
		}

		if len(permissions) == 1 && permissions[0].Group.ID == g.ID {
			log.Warning("updateGroupRoleOnApplicationHandler: Cannot remove write permission for group %s in application %s\n", groupName, appName)
			return sdk.ErrGroupNeedWrite
		}
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateGroupRoleOnApplicationHandler: Cannot start transaction: %s\n", err)
		return sdk.ErrUnknownError
	}
	defer tx.Rollback()

	if err := group.UpdateGroupRoleInApplication(tx, key, appName, groupName, groupApplication.Permission); err != nil {
		log.Warning("updateGroupRoleOnApplicationHandler: Cannot update permission for group %s in application %s:  %s\n", groupName, appName, err)
		return err
	}

	if err := application.UpdateLastModified(tx, app, c.User); err != nil {
		log.Warning("updateGroupsInApplicationHandler: Cannot update last modified date: %s\n", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("updateGroupRoleOnApplicationHandler: Cannot commit transaction: %s\n", err)
		return err
	}

	cache.DeleteAll(cache.Key("application", key, "*"+appName+"*"))

	if err := application.LoadGroupByApplication(db, app); err != nil {
		log.Warning("updateGroupRoleOnApplicationHandler: Cannot load application groups: %s\n", err)
		return err
	}

	return WriteJSON(w, r, app, http.StatusOK)
}

// Deprecated
func updateGroupsInApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]

	proj, err := project.Load(db, key, c.User)
	if err != nil {
		log.Warning("addGroupInApplicationHandler> Cannot load %s: %s\n", key, err)
		return err
	}

	var groupsPermission []sdk.GroupPermission
	if err := UnmarshalBody(r, &groupsPermission); err != nil {
		return err
	}

	if len(groupsPermission) == 0 {
		log.Warning("updateGroupsInApplicationHandler: Cannot remove all groups for application %s", appName)
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
		log.Warning("updateGroupsInApplicationHandler: Need one group with write permission.")
		return sdk.ErrGroupNeedWrite
	}

	app, err := application.LoadByName(db, key, appName, c.User)
	if err != nil {
		log.Warning("updateGroupsInApplicationHandler: Cannot load application %s: %s\n", appName, err)
		return sdk.ErrApplicationNotFound
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateGroupsInApplicationHandler: Cannot start transaction: %s\n", err)
		return sdk.ErrUnknownError
	}
	defer tx.Rollback()

	if err := group.DeleteAllGroupFromApplication(tx, app.ID); err != nil {
		log.Warning("updateGroupsInApplicationHandler: Cannot delete groups from application %s: %s\n", appName, err)
		return err
	}

	if err := application.AddGroup(tx, proj, app, groupsPermission...); err != nil {
		log.Warning("updateGroupsInApplicationHandler: Cannot add groups in application %s:  %s\n", app.Name, err)
		return err
	}

	if err := application.UpdateLastModified(tx, app, c.User); err != nil {
		log.Warning("updateGroupsInApplicationHandler: Cannot update last modified date: %s\n", err)
		return err
	}

	if err := project.UpdateLastModified(tx, c.User, proj); err != nil {
		log.Warning("updateGroupsInApplicationHandler: Cannot update last modified date: %s\n", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("updateGroupsInApplicationHandler: Cannot commit transaction: %s\n", err)
		return sdk.ErrUnknownError
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
		return err
	}

	proj, err := project.Load(db, key, c.User)
	if err != nil {
		log.Warning("addGroupInApplicationHandler> Cannot load %s: %s\n", key, err)
		return err
	}

	app, err := application.LoadByName(db, key, appName, c.User)
	if err != nil {
		log.Warning("addGroupInApplicationHandler> Cannot load %s: %s\n", appName, err)
		return err
	}

	g, err := group.LoadGroup(db, groupPermission.Group.Name)
	if err != nil {
		log.Warning("addGroupInApplicationHandler> Cannot find %s: %s\n", groupPermission.Group.Name, err)
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("addGroupInApplicationHandler> Cannot start transaction: %s\n", err)
		return err
	}
	defer tx.Rollback()

	if err := application.AddGroup(tx, proj, app, groupPermission); err != nil {
		log.Warning("addGroupInApplicationHandler> Cannot add group %s in application %s:  %s\n", g.Name, app.Name, err)
		return err
	}

	if err := application.UpdateLastModified(tx, app, c.User); err != nil {
		log.Warning("addGroupInApplicationHandler> Cannot update application last modified date:  %s\n", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("addGroupInApplicationHandler> Cannot commit transaction: %s\n", err)
		return err
	}

	cache.DeleteAll(cache.Key("application", key, "*"+appName+"*"))

	if err := application.LoadGroupByApplication(db, app); err != nil {
		log.Warning("addGroupInApplicationHandler> Cannot load application groups: %s\n", err)
		return err
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
		log.Warning("deleteGroupFromApplicationHandler: Cannot load application %s :  %s\n", appName, err)
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteGroupFromApplicationHandler: Cannot start transaction: %s\n", err)
		return err
	}
	defer tx.Rollback()

	if err := group.DeleteGroupFromApplication(tx, key, appName, groupName); err != nil {
		log.Warning("deleteGroupFromApplicationHandler: Cannot delete group %s from pipeline %s:  %s\n", groupName, appName, err)
		return err
	}

	if err := application.UpdateLastModified(tx, app, c.User); err != nil {
		log.Warning("deleteGroupFromApplicationHandler: Cannot update application last modified date:  %s\n", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("deleteGroupFromApplicationHandler: Cannot commit transaction: %s\n", err)
		return err
	}

	cache.DeleteAll(cache.Key("application", key, "*"+appName+"*"))

	if err := application.LoadGroupByApplication(db, app); err != nil {
		log.Warning("deleteGroupFromApplicationHandler: Cannot load application groups: %s\n", err)
		return err
	}

	return WriteJSON(w, r, app, http.StatusOK)
}
