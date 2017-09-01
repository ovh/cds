package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) updateGroupRoleOnEnvironmentHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		envName := vars["permEnvironmentName"]
		groupName := vars["group"]

		var groupEnvironment sdk.GroupPermission
		if err := UnmarshalBody(r, &groupEnvironment); err != nil {
			return sdk.WrapError(err, "updateGroupRoleOnEnvironmentHandler> Cannot read body")
		}

		g, errG := group.LoadGroup(api.MustDB(), groupName)
		if errG != nil {
			return sdk.WrapError(errG, "updateGroupRoleOnEnvironmentHandler> Cannot load group %s", groupName)
		}

		env, errE := environment.LoadEnvironmentByName(api.MustDB(), key, envName)
		if errE != nil {
			return sdk.WrapError(errE, "updateGroupRoleOnEnvironmentHandler> Cannot load environment %s", envName)
		}

		if groupEnvironment.Permission != permission.PermissionReadWriteExecute {
			permissions, errR := group.LoadAllEnvironmentGroupByRole(api.MustDB(), env.ID, permission.PermissionReadWriteExecute)
			if errR != nil {
				return sdk.WrapError(errR, "updateGroupRoleOnEnvironmentHandler> Cannot load group %s for environment %s", groupName, envName)
			}

			if len(permissions) == 1 && permissions[0].Group.ID == g.ID {
				log.Warning("updateGroupRoleOnEnvironmentHandler: Cannot remove write permission on group %s for environment %s :%s", groupName, envName)
				return sdk.WrapError(sdk.ErrGroupNeedWrite, "updateGroupRoleOnEnvironmentHandler> Cannot remove write permission on group %s for environment %s", groupName, envName)
			}
		}

		p, errP := project.Load(api.MustDB(), key, getUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "updateGroupRoleOnEnvironmentHandler> Cannot load project %s", key)
		}

		tx, errB := api.MustDB().Begin()
		if errB != nil {
			return sdk.WrapError(errB, "updateGroupRoleOnEnvironmentHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := group.UpdateGroupRoleInEnvironment(tx, key, envName, groupName, groupEnvironment.Permission); err != nil {
			return sdk.WrapError(err, "updateGroupRoleOnEnvironmentHandler: Cannot update permission for group %s in environment %s", groupName, envName)
		}

		if err := environment.UpdateLastModified(tx, getUser(ctx), env); err != nil {
			return sdk.WrapError(err, "updateGroupRoleOnEnvironmentHandler: Cannot update environment last modified date")
		}

		if err := project.UpdateLastModified(tx, getUser(ctx), p); err != nil {
			return sdk.WrapError(err, "updateGroupRoleOnEnvironmentHandler: Cannot update project last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateGroupRoleOnEnvironmentHandler> Cannot commit transaction")
		}

		envUpdated, errE := environment.LoadEnvironmentByName(api.MustDB(), key, envName)
		if errE != nil {
			return sdk.WrapError(errE, "updateGroupRoleOnEnvironmentHandler> Cannot load updated environment")
		}

		return WriteJSON(w, r, envUpdated, http.StatusOK)
	}
}

func (api *API) addGroupsInEnvironmentHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		envName := vars["permEnvironmentName"]

		var groupPermission []sdk.GroupPermission
		if err := UnmarshalBody(r, &groupPermission); err != nil {
			return sdk.WrapError(err, "addGroupsInEnvironmentHandler> Cannot read body")
		}

		env, err := environment.LoadEnvironmentByName(api.MustDB(), key, envName)
		if err != nil {
			return sdk.WrapError(err, "addGroupsInEnvironmentHandler> Cannot load environment %s", envName)
		}

		tx, errB := api.MustDB().Begin()
		if errB != nil {
			return sdk.WrapError(errB, "addGroupsInEnvironmentHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		for _, gp := range groupPermission {
			g, errL := group.LoadGroup(tx, gp.Group.Name)
			if errL != nil {
				return sdk.WrapError(errL, "addGroupsInEnvironmentHandler: Cannot find group %s", gp.Group.Name)
			}

			alreadyAdded, errA := group.IsInEnvironment(tx, env.ID, g.ID)
			if errA != nil {
				return sdk.WrapError(errA, "addGroupsInEnvironmentHandler> Cannot check if group is in env")
			}

			if alreadyAdded {
				return sdk.WrapError(sdk.ErrGroupPresent, "addGroupsInEnvironmentHandler> Group %s already in environment %s", g.Name, env.Name)
			}

			if err := group.InsertGroupInEnvironment(tx, env.ID, g.ID, gp.Permission); err != nil {
				return sdk.WrapError(err, "addGroupsInEnvironmentHandler: Cannot add group %s in environment %s", g.Name, env.Name)
			}
		}

		// Update last modified on environment
		if err := environment.UpdateEnvironment(tx, env); err != nil {
			return sdk.WrapError(err, "addGroupsInEnvironmentHandler: Cannot update environment %s", env.Name)
		}

		p, errP := project.Load(tx, key, getUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "addGroupsInEnvironmentHandler: Cannot load project %s", env.Name)
		}

		if err := environment.UpdateLastModified(tx, getUser(ctx), env); err != nil {
			return sdk.WrapError(err, "addGroupsInEnvironmentHandler: Cannot update environment last modified date")
		}

		if err := project.UpdateLastModified(tx, getUser(ctx), p); err != nil {
			return sdk.WrapError(errP, "addGroupsInEnvironmentHandler: Cannot update project %s", p.Key)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addGroupsInEnvironmentHandler: Cannot commit transaction")
		}

		envUpdated, errL := environment.LoadEnvironmentByName(api.MustDB(), key, envName)
		if errL != nil {
			return sdk.WrapError(errL, "addGroupsInEnvironmentHandler: Cannot load updated environment")
		}

		return WriteJSON(w, r, envUpdated, http.StatusOK)
	}
}

func (api *API) addGroupInEnvironmentHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		envName := vars["permEnvironmentName"]

		var groupPermission sdk.GroupPermission
		if err := UnmarshalBody(r, &groupPermission); err != nil {
			return err
		}

		env, err := environment.LoadEnvironmentByName(api.MustDB(), key, envName)
		if err != nil {
			log.Warning("addGroupInEnvironmentHandler: Cannot load %s: %s\n", envName, err)
			return err
		}

		g, err := group.LoadGroup(api.MustDB(), groupPermission.Group.Name)
		if err != nil {
			log.Warning("addGroupInEnvironmentHandler: Cannot find %s: %s\n", groupPermission.Group.Name, err)
			return err
		}

		alreadyAdded, err := group.IsInEnvironment(api.MustDB(), env.ID, g.ID)
		if err != nil {
			log.Warning("addGroupInEnvironmentHandler> Cannot check if group is in env: %s\n", err)
			return err
		}

		if alreadyAdded {
			log.Warning("addGroupInEnvironmentHandler> Group %s is already present in env %s\n", g.Name, env.Name)
			return sdk.ErrGroupPresent
		}

		if err := group.InsertGroupInEnvironment(api.MustDB(), env.ID, g.ID, groupPermission.Permission); err != nil {
			log.Warning("addGroupInEnvironmentHandler: Cannot add group %s in environment %s:  %s\n", g.Name, env.Name, err)
			return err
		}

		return nil
	}
}

func (api *API) deleteGroupFromEnvironmentHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		envName := vars["permEnvironmentName"]
		groupName := vars["group"]

		proj, errP := project.Load(api.MustDB(), key, getUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "deleteGroupFromEnvironmentHandler> Cannot load project")
		}

		env, errE := environment.LoadEnvironmentByName(api.MustDB(), proj.Key, envName)
		if errE != nil {
			return sdk.WrapError(errE, "deleteGroupFromEnvironmentHandler: Cannot load environment")
		}

		tx, errT := api.MustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "deleteGroupFromEnvironmentHandler: Cannot start transaction")
		}
		defer tx.Rollback()

		if err := group.DeleteGroupFromEnvironment(tx, proj.Key, envName, groupName); err != nil {
			return sdk.WrapError(err, "deleteGroupFromEnvironmentHandler: Cannot delete group %s from pipeline %s", groupName, envName)
		}

		if err := project.UpdateLastModified(tx, getUser(ctx), proj); err != nil {
			return sdk.WrapError(err, "deleteGroupFromEnvironmentHandler: Cannot update project last modified date")
		}

		if err := environment.UpdateLastModified(tx, getUser(ctx), env); err != nil {
			return sdk.WrapError(err, "deleteGroupFromEnvironmentHandler: Cannot update environment last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(errT, "deleteGroupFromEnvironmentHandler: Cannot commit transaction")
		}

		return nil
	}
}
