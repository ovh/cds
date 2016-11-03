package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func updateGroupRoleOnApplicationHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	groupName := vars["group"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("updateGroupRoleOnApplicationHandler: Cannot read body :%s", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	var groupApplication sdk.GroupPermission
	if err := json.Unmarshal(data, &groupApplication); err != nil {
		log.Warning("updateGroupRoleOnApplicationHandler: Cannot unmarshal body :%s", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	app, err := application.LoadApplicationByName(db, key, appName)
	if err != nil {
		log.Warning("updateGroupRoleOnApplicationHandler: Cannot load application %s :%s", appName, err)
		WriteError(w, r, sdk.ErrApplicationNotFound)
		return
	}

	g, err := group.LoadGroup(db, groupName)
	if err != nil {
		log.Warning("updateGroupRoleOnApplicationHandler: Cannot load group %s :%s", groupName, err)
		WriteError(w, r, sdk.ErrGroupNotFound)
		return
	}

	if groupApplication.Permission != permission.PermissionReadWriteExecute {
		permissions, err := group.LoadAllApplicationGroupByRole(db, app.ID, permission.PermissionReadWriteExecute)
		if err != nil {
			log.Warning("updateGroupRoleOnApplicationHandler: Cannot load group for application %s:  %s\n", appName, err)
			WriteError(w, r, err)
			return
		}

		if len(permissions) == 1 && permissions[0].Group.ID == g.ID {
			log.Warning("updateGroupRoleOnApplicationHandler: Cannot remove write permission for group %s in application %s\n", groupName, appName)
			WriteError(w, r, sdk.ErrGroupNeedWrite)
			return
		}
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateGroupRoleOnApplicationHandler: Cannot start transaction: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}
	defer tx.Rollback()

	if err := group.UpdateGroupRoleInApplication(tx, key, appName, groupName, groupApplication.Permission); err != nil {
		log.Warning("updateGroupRoleOnApplicationHandler: Cannot update permission for group %s in application %s:  %s\n", groupName, appName, err)
		WriteError(w, r, err)
		return
	}

	if err := application.UpdateLastModified(tx, app); err != nil {
		log.Warning("updateGroupRoleOnApplicationHandler: Cannot update last modified date:  %s\n", err)
		WriteError(w, r, err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Warning("updateGroupRoleOnApplicationHandler: Cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	cache.DeleteAll(cache.Key("application", key, "*"+appName+"*"))

	if err := application.LoadGroupByApplication(db, app); err != nil {
		log.Warning("updateGroupRoleOnApplicationHandler: Cannot load application groups: %s\n", err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, app, http.StatusOK)
}

func updateGroupsInApplicationHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("updateGroupsInApplicationHandler: Cannot read body :%s", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	var groupsPermission []sdk.GroupPermission
	if err := json.Unmarshal(data, &groupsPermission); err != nil {
		log.Warning("updateGroupsInApplicationHandler: Cannot unmarshal body :%s", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	if len(groupsPermission) == 0 {
		log.Warning("updateGroupsInApplicationHandler: Cannot remove all groups for application %s", appName)
		WriteError(w, r, sdk.ErrGroupNeedWrite)
		return
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
		WriteError(w, r, sdk.ErrGroupNeedWrite)
		return
	}

	app, err := application.LoadApplicationByName(db, key, appName)
	if err != nil {
		log.Warning("updateGroupsInApplicationHandler: Cannot load application %s: %s\n", appName, err)
		WriteError(w, r, sdk.ErrApplicationNotFound)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateGroupsInApplicationHandler: Cannot start transaction: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}
	defer tx.Rollback()

	if err := group.DeleteAllGroupFromApplication(tx, app.ID); err != nil {
		log.Warning("updateGroupsInApplicationHandler: Cannot delete groups from application %s: %s\n", appName, err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

	for _, gp := range groupsPermission {
		g, err := group.LoadGroup(tx, gp.Group.Name)
		if err != nil {
			log.Warning("updateGroupsInApplicationHandler: Cannot find %s: %s\n", gp.Group.Name, err)
			WriteError(w, r, sdk.ErrGroupNotFound)
			return
		}

		if err := group.InsertGroupInApplication(tx, app.ID, g.ID, gp.Permission); err != nil {
			log.Warning("updateGroupsInApplicationHandler: Cannot add group %s in application %s:  %s\n", g.Name, app.Name, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	if err := application.UpdateLastModified(tx, app); err != nil {
		log.Warning("updateGroupsInApplicationHandler: Cannot update last modified date: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

	if err = tx.Commit(); err != nil {
		log.Warning("updateGroupsInApplicationHandler: Cannot commit transaction: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

	cache.DeleteAll(cache.Key("application", key, "*"+appName+"*"))
	w.WriteHeader(http.StatusOK)
}

func addGroupInApplicationHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("addGroupInApplicationHandler: Cannot read body :%s", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	var groupPermission sdk.GroupPermission
	if err := json.Unmarshal(data, &groupPermission); err != nil {
		log.Warning("addGroupInApplicationHandler: Cannot unmarshal body :%s", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	app, err := application.LoadApplicationByName(db, key, appName)
	if err != nil {
		log.Warning("addGroupInApplicationHandler: Cannot load %s: %s\n", appName, err)
		WriteError(w, r, err)
		return
	}

	g, err := group.LoadGroup(db, groupPermission.Group.Name)
	if err != nil {
		log.Warning("addGroupInApplicationHandler: Cannot find %s: %s\n", groupPermission.Group.Name, err)
		WriteError(w, r, err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("addGroupInApplicationHandler: Cannot start transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	if err := group.InsertGroupInApplication(tx, app.ID, g.ID, groupPermission.Permission); err != nil {
		log.Warning("addGroupInApplicationHandler: Cannot add group %s in application %s:  %s\n", g.Name, app.Name, err)
		WriteError(w, r, err)
		return
	}

	if err := application.UpdateLastModified(tx, app); err != nil {
		log.Warning("addGroupInApplicationHandler: Cannot update application last modified date:  %s\n", err)
		WriteError(w, r, err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Warning("addGroupInApplicationHandler: Cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	cache.DeleteAll(cache.Key("application", key, "*"+appName+"*"))

	if err := application.LoadGroupByApplication(db, app); err != nil {
		log.Warning("addGroupInApplicationHandler: Cannot load application groups: %s\n", err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, app, http.StatusOK)
}

func deleteGroupFromApplicationHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	appName := vars["permApplicationName"]
	groupName := vars["group"]

	app, err := application.LoadApplicationByName(db, key, appName)
	if err != nil {
		log.Warning("deleteGroupFromApplicationHandler: Cannot load application %s :  %s\n", appName, err)
		WriteError(w, r, err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteGroupFromApplicationHandler: Cannot start transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	if err := group.DeleteGroupFromApplication(tx, key, appName, groupName); err != nil {
		log.Warning("deleteGroupFromApplicationHandler: Cannot delete group %s from pipeline %s:  %s\n", groupName, appName, err)
		WriteError(w, r, err)
		return
	}

	if err := application.UpdateLastModified(tx, app); err != nil {
		log.Warning("deleteGroupFromApplicationHandler: Cannot update application last modified date:  %s\n", err)
		WriteError(w, r, err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Warning("deleteGroupFromApplicationHandler: Cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	cache.DeleteAll(cache.Key("application", key, "*"+appName+"*"))

	if err := application.LoadGroupByApplication(db, app); err != nil {
		log.Warning("deleteGroupFromApplicationHandler: Cannot load application groups: %s\n", err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, app, http.StatusOK)
}
