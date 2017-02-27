package main

import (
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func updateGroupRoleOnEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	envName := vars["permEnvironmentName"]
	groupName := vars["group"]

	var groupEnvironment sdk.GroupPermission
	if err := UnmarshalBody(r, &groupEnvironment); err != nil {
		return err
	}

	g, err := group.LoadGroup(db, groupName)
	if err != nil {
		log.Warning("updateGroupRoleOnEnvironmentHandler: Canont load group %s :%s", groupName, err)
		return sdk.ErrGroupNotFound
	}

	env, err := environment.LoadEnvironmentByName(db, key, envName)
	if err != nil {
		log.Warning("updateGroupRoleOnEnvironmentHandler: Canont load environment %s :%s", envName, err)
		return sdk.ErrNoEnvironment
	}

	if groupEnvironment.Permission != permission.PermissionReadWriteExecute {
		permissions, err := group.LoadAllEnvironmentGroupByRole(db, env.ID, permission.PermissionReadWriteExecute)
		if err != nil {
			log.Warning("updateGroupRoleOnEnvironmentHandler: Cannot load group for environment %s :%s", envName, err)
			return err
		}

		if len(permissions) == 1 && permissions[0].Group.ID == g.ID {
			log.Warning("updateGroupRoleOnEnvironmentHandler: Cannot remove write permission on group %s for environment %s :%s", groupName, envName)
			return sdk.ErrGroupNeedWrite
		}
	}

	err = group.UpdateGroupRoleInEnvironment(db, key, envName, groupName, groupEnvironment.Permission)
	if err != nil {
		log.Warning("updateGroupRoleOnEnvironmentHandler: Cannot update permission for group %s in environment %s:  %s\n", groupName, envName, err)
		return err
	}
	return nil
}

func addGroupInEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	envName := vars["permEnvironmentName"]

	var groupPermission sdk.GroupPermission
	if err := UnmarshalBody(r, &groupPermission); err != nil {
		return err
	}

	env, err := environment.LoadEnvironmentByName(db, key, envName)
	if err != nil {
		log.Warning("addGroupInEnvironmentHandler: Cannot load %s: %s\n", envName, err)
		return err
	}

	g, err := group.LoadGroup(db, groupPermission.Group.Name)
	if err != nil {
		log.Warning("addGroupInEnvironmentHandler: Cannot find %s: %s\n", groupPermission.Group.Name, err)
		return err
	}

	alreadyAdded, err := group.IsInEnvironment(db, env.ID, g.ID)
	if err != nil {
		log.Warning("addGroupInEnvironmentHandler> Cannot check if group is in env: %s\n", err)
		return err
	}

	if alreadyAdded {
		log.Warning("addGroupInEnvironmentHandler> Group %s is already present in env %s\n", g.Name, env.Name)
		return sdk.ErrGroupPresent
	}

	if err := group.InsertGroupInEnvironment(db, env.ID, g.ID, groupPermission.Permission); err != nil {
		log.Warning("addGroupInEnvironmentHandler: Cannot add group %s in environment %s:  %s\n", g.Name, env.Name, err)
		return err
	}

	return nil
}

func deleteGroupFromEnvironmentHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	envName := vars["permEnvironmentName"]
	groupName := vars["group"]

	err := group.DeleteGroupFromEnvironment(db, key, envName, groupName)
	if err != nil {
		log.Warning("deleteGroupFromEnvironmentHandler: Cannot delete group %s from pipeline %s:  %s\n", groupName, envName, err)
		return err
	}
	return nil
}
