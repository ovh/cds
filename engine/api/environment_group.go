package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func updateGroupRoleOnEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	envName := vars["permEnvironmentName"]
	groupName := vars["group"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("updateGroupRoleOnEnvironmentHandler: Cannot read body :%s", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

	var groupEnvironment sdk.GroupPermission
	err = json.Unmarshal(data, &groupEnvironment)
	if err != nil {
		log.Warning("updateGroupRoleOnEnvironmentHandler: Cannot unmarshal body :%s", err)
		WriteError(w, r, err)
		return
	}

	g, err := group.LoadGroup(db, groupName)
	if err != nil {
		log.Warning("updateGroupRoleOnEnvironmentHandler: Canont load group %s :%s", groupName, err)
		WriteError(w, r, sdk.ErrGroupNotFound)
		return
	}

	env, err := environment.LoadEnvironmentByName(db, key, envName)
	if err != nil {
		log.Warning("updateGroupRoleOnEnvironmentHandler: Canont load environment %s :%s", envName, err)
		WriteError(w, r, sdk.ErrNoEnvironment)
		return
	}

	if groupEnvironment.Permission != permission.PermissionReadWriteExecute {
		permissions, err := group.LoadAllEnvironmentGroupByRole(db, env.ID, permission.PermissionReadWriteExecute)
		if err != nil {
			log.Warning("updateGroupRoleOnEnvironmentHandler: Cannot load group for environment %s :%s", envName, err)
			WriteError(w, r, err)
			return
		}

		if len(permissions) == 1 && permissions[0].Group.ID == g.ID {
			log.Warning("updateGroupRoleOnEnvironmentHandler: Cannot remove write permission on group %s for environment %s :%s", groupName, envName)
			WriteError(w, r, sdk.ErrGroupNeedWrite)
			return
		}
	}

	err = group.UpdateGroupRoleInEnvironment(db, key, envName, groupName, groupEnvironment.Permission)
	if err != nil {
		log.Warning("updateGroupRoleOnEnvironmentHandler: Cannot update permission for group %s in environment %s:  %s\n", groupName, envName, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func addGroupInEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	envName := vars["permEnvironmentName"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("addGroupInEnvironmentHandler: Cannot read body :%s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var groupPermission sdk.GroupPermission
	err = json.Unmarshal(data, &groupPermission)
	if err != nil {
		log.Warning("addGroupInEnvironmentHandler: Cannot unmarshal body :%s", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	env, err := environment.LoadEnvironmentByName(db, key, envName)
	if err != nil {
		log.Warning("addGroupInEnvironmentHandler: Cannot load %s: %s\n", envName, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	g, err := group.LoadGroup(db, groupPermission.Group.Name)
	if err != nil {
		log.Warning("addGroupInEnvironmentHandler: Cannot find %s: %s\n", groupPermission.Group.Name, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	alreadyAdded, err := group.IsInEnvironment(db, env.ID, g.ID)
	if err != nil {
		log.Warning("addGroupInEnvironmentHandler> Cannot check if group is in env: %s\n", err)
		WriteError(w, r, err)
		return
	}

	if alreadyAdded {
		log.Warning("addGroupInEnvironmentHandler> Group %s is already present in env %s\n", g.Name, env.Name)
		WriteError(w, r, sdk.ErrGroupPresent)
		return
	}

	err = group.InsertGroupInEnvironment(db, env.ID, g.ID, groupPermission.Permission)
	if err != nil {
		log.Warning("addGroupInEnvironmentHandler: Cannot add group %s in environment %s:  %s\n", g.Name, env.Name, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func deleteGroupFromEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	envName := vars["permEnvironmentName"]
	groupName := vars["group"]

	err := group.DeleteGroupFromEnvironment(db, key, envName, groupName)
	if err != nil {
		log.Warning("deleteGroupFromEnvironmentHandler: Cannot delete group %s from pipeline %s:  %s\n", groupName, envName, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
