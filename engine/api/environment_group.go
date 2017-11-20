package api

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
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

		g, errG := group.LoadGroup(api.mustDB(), groupName)
		if errG != nil {
			return sdk.WrapError(errG, "updateGroupRoleOnEnvironmentHandler> Cannot load group %s", groupName)
		}

		if group.IsDefaultGroupID(g.ID) && groupEnvironment.Permission > permission.PermissionRead {
			return sdk.WrapError(sdk.ErrDefaultGroupPermission, "updateGroupRoleOnEnvironmentHandler: only read permission is allowed to default group")
		}

		env, errE := environment.LoadEnvironmentByName(api.mustDB(), key, envName)
		if errE != nil {
			return sdk.WrapError(errE, "updateGroupRoleOnEnvironmentHandler> Cannot load environment %s", envName)
		}

		if groupEnvironment.Permission != permission.PermissionReadWriteExecute {
			permissions, errR := group.LoadAllEnvironmentGroupByRole(api.mustDB(), env.ID, permission.PermissionReadWriteExecute)
			if errR != nil {
				return sdk.WrapError(errR, "updateGroupRoleOnEnvironmentHandler> Cannot load group %s for environment %s", groupName, envName)
			}

			if len(permissions) == 1 && permissions[0].Group.ID == g.ID {
				log.Warning("updateGroupRoleOnEnvironmentHandler: Cannot remove write permission on group %s for environment %s :%s", groupName, envName)
				return sdk.WrapError(sdk.ErrGroupNeedWrite, "updateGroupRoleOnEnvironmentHandler> Cannot remove write permission on group %s for environment %s", groupName, envName)
			}
		}

		p, errP := project.Load(api.mustDB(), api.Cache, key, getUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "updateGroupRoleOnEnvironmentHandler> Cannot load project %s", key)
		}

		tx, errB := api.mustDB().Begin()
		if errB != nil {
			return sdk.WrapError(errB, "updateGroupRoleOnEnvironmentHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := group.UpdateGroupRoleInEnvironment(tx, env.ID, g.ID, groupEnvironment.Permission); err != nil {
			return sdk.WrapError(err, "updateGroupRoleOnEnvironmentHandler: Cannot update permission for group %s in environment %s", groupName, envName)
		}

		if err := environment.UpdateLastModified(tx, api.Cache, getUser(ctx), env); err != nil {
			return sdk.WrapError(err, "updateGroupRoleOnEnvironmentHandler: Cannot update environment last modified date")
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p); err != nil {
			return sdk.WrapError(err, "updateGroupRoleOnEnvironmentHandler: Cannot update project last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateGroupRoleOnEnvironmentHandler> Cannot commit transaction")
		}

		envUpdated, errE := environment.LoadEnvironmentByName(api.mustDB(), key, envName)
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

		env, err := environment.LoadEnvironmentByName(api.mustDB(), key, envName)
		if err != nil {
			return sdk.WrapError(err, "addGroupsInEnvironmentHandler> Cannot load environment %s", envName)
		}

		tx, errB := api.mustDB().Begin()
		if errB != nil {
			return sdk.WrapError(errB, "addGroupsInEnvironmentHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		for _, gp := range groupPermission {
			g, errL := group.LoadGroup(tx, gp.Group.Name)
			if errL != nil {
				return sdk.WrapError(errL, "addGroupsInEnvironmentHandler: Cannot find group %s", gp.Group.Name)
			}

			if group.IsDefaultGroupID(g.ID) && gp.Permission > permission.PermissionRead {
				return sdk.WrapError(sdk.ErrDefaultGroupPermission, "addGroupsInEnvironmentHandler: only read permission is allowed to default group")
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

		p, errP := project.Load(tx, api.Cache, key, getUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "addGroupsInEnvironmentHandler: Cannot load project %s", env.Name)
		}

		if err := environment.UpdateLastModified(tx, api.Cache, getUser(ctx), env); err != nil {
			return sdk.WrapError(err, "addGroupsInEnvironmentHandler: Cannot update environment last modified date")
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), p); err != nil {
			return sdk.WrapError(errP, "addGroupsInEnvironmentHandler: Cannot update project %s", p.Key)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addGroupsInEnvironmentHandler: Cannot commit transaction")
		}

		envUpdated, errL := environment.LoadEnvironmentByName(api.mustDB(), key, envName)
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

		env, err := environment.LoadEnvironmentByName(api.mustDB(), key, envName)
		if err != nil {
			return sdk.WrapError(err, "addGroupInEnvironmentHandler: Cannot load %s", envName)
		}

		g, err := group.LoadGroup(api.mustDB(), groupPermission.Group.Name)
		if err != nil {
			return sdk.WrapError(err, "addGroupInEnvironmentHandler: Cannot find %s", groupPermission.Group.Name)
		}

		if group.IsDefaultGroupID(g.ID) && groupPermission.Permission > permission.PermissionRead {
			return sdk.WrapError(sdk.ErrDefaultGroupPermission, "addGroupInEnvironmentHandler: only read permission is allowed to default group")
		}

		alreadyAdded, err := group.IsInEnvironment(api.mustDB(), env.ID, g.ID)
		if err != nil {
			return sdk.WrapError(err, "addGroupInEnvironmentHandler> Cannot check if group is in env")
		}

		if alreadyAdded {
			return sdk.WrapError(sdk.ErrGroupPresent, "addGroupInEnvironmentHandler> Group %s is already present in env %s", g.Name, env.Name)
		}

		if err := group.InsertGroupInEnvironment(api.mustDB(), env.ID, g.ID, groupPermission.Permission); err != nil {
			return sdk.WrapError(err, "addGroupInEnvironmentHandler: Cannot add group %s in environment %s", g.Name, env.Name)
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

		proj, errP := project.Load(api.mustDB(), api.Cache, key, getUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "deleteGroupFromEnvironmentHandler> Cannot load project")
		}

		env, errE := environment.LoadEnvironmentByName(api.mustDB(), proj.Key, envName)
		if errE != nil {
			return sdk.WrapError(errE, "deleteGroupFromEnvironmentHandler: Cannot load environment")
		}

		g, errG := group.LoadGroup(api.mustDB(), envName)
		if errG != nil {
			return sdk.WrapError(errG, "deleteGroupFromEnvironmentHandler: Cannot load group")
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "deleteGroupFromEnvironmentHandler: Cannot start transaction")
		}
		defer tx.Rollback()

		if err := group.DeleteGroupFromEnvironment(tx, env.ID, g.ID); err != nil {
			return sdk.WrapError(err, "deleteGroupFromEnvironmentHandler: Cannot delete group %s from pipeline %s", groupName, envName)
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), proj); err != nil {
			return sdk.WrapError(err, "deleteGroupFromEnvironmentHandler: Cannot update project last modified date")
		}

		if err := environment.UpdateLastModified(tx, api.Cache, getUser(ctx), env); err != nil {
			return sdk.WrapError(err, "deleteGroupFromEnvironmentHandler: Cannot update environment last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(errT, "deleteGroupFromEnvironmentHandler: Cannot commit transaction")
		}

		return nil
	}
}

func (api *API) importGroupsInEnvironmentHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		envName := vars["permEnvironmentName"]
		format := r.FormValue("format")
		forceUpdate := FormBool(r, "forceUpdate")

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "importGroupsInEnvironmentHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		proj, errProj := project.Load(tx, api.Cache, key, getUser(ctx), project.LoadOptions.WithGroups)
		if errProj != nil {
			return sdk.WrapError(errProj, "importGroupsInEnvironmentHandler> Cannot load project %s", key)
		}

		env, errE := environment.LoadEnvironmentByName(tx, key, envName)
		if errE != nil {
			return sdk.WrapError(errE, "importGroupsInEnvironmentHandler> Cannot load environment %s", envName)
		}

		groupsToAdd := []sdk.GroupPermission{}
		// Get body
		data, errRead := ioutil.ReadAll(r.Body)
		if errRead != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "importGroupsInEnvironmentHandler> Unable to read body")
		}

		f, errF := exportentities.GetFormat(format)
		if errF != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "importGroupsInEnvironmentHandler> Unable to get format")
		}

		var errorParse error
		switch f {
		case exportentities.FormatJSON:
			errorParse = json.Unmarshal(data, &groupsToAdd)
		case exportentities.FormatYAML:
			errorParse = yaml.Unmarshal(data, &groupsToAdd)
		}

		if errorParse != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "importGroupsInEnvironmentHandler> Cannot parsing")
		}

		groupsToAddInProj := []sdk.GroupPermission{}
		for _, gr := range groupsToAdd {
			exist := false
			for _, gro := range proj.ProjectGroups {
				if gr.Group.Name == gro.Group.Name {
					exist = true
				}
			}
			if !exist && !forceUpdate {
				return sdk.WrapError(sdk.ErrGroupNotFound, "importGroupsInEnvironmentHandler> Group %v doesn't exist in this project", gr.Group.Name)
			} else if !exist && forceUpdate {
				groupsToAddInProj = append(groupsToAddInProj, sdk.GroupPermission{
					Group:      gr.Group,
					Permission: permission.PermissionRead,
				})
			}
		}

		if forceUpdate { // clean and update
			for _, gr := range groupsToAddInProj {
				gro, errG := group.LoadGroup(tx, gr.Group.Name)
				if errG != nil {
					return sdk.WrapError(sdk.ErrGroupNotFound, "importGroupsInEnvironmentHandler> Group %v doesn't exist", gr.Group.Name)
				}
				if err := group.InsertGroupInProject(tx, proj.ID, gro.ID, gr.Permission); err != nil {
					return sdk.WrapError(err, "importGroupsInEnvironmentHandler> Cannot add group %v in project %s", gr.Group.Name, proj.Name)
				}
				gr.Group = *gro
				proj.ProjectGroups = append(proj.ProjectGroups, gr)
			}

			if err := group.DeleteAllGroupFromEnvironment(tx, env.ID); err != nil {
				return sdk.WrapError(err, "importGroupsInEnvironmentHandler> Cannot delete all groups for this environment %s", env.Name)
			}

			env.EnvironmentGroups = []sdk.GroupPermission{}
			for _, gr := range groupsToAdd {
				gro, errG := group.LoadGroup(tx, gr.Group.Name)
				if errG != nil {
					return sdk.WrapError(sdk.ErrGroupNotFound, "importGroupsInEnvironmentHandler> Cannot load group %s : %s", gr.Group.Name, errG)
				}
				if err := group.InsertGroupInEnvironment(tx, env.ID, gro.ID, gr.Permission); err != nil {
					return sdk.WrapError(err, "importGroupsInEnvironmentHandler> Cannot insert group %s in this environment %s", gr.Group.Name, env.Name)
				}
				env.EnvironmentGroups = append(env.EnvironmentGroups, sdk.GroupPermission{Group: sdk.Group{Name: gr.Group.Name}, Permission: gr.Permission})
			}
		} else { // add new group
			for _, gr := range groupsToAdd {
				if _, errGr := group.GetIDByNameInList(env.EnvironmentGroups, gr.Group.Name); errGr == nil {
					return sdk.WrapError(sdk.ErrGroupExists, "importGroupsInEnvironmentHandler> Group %s in environment %s", gr.Group.Name, env.Name)
				}

				grID, errG := group.GetIDByNameInList(proj.ProjectGroups, gr.Group.Name)
				if errG != nil {
					return sdk.WrapError(sdk.ErrGroupNotFound, "importGroupsInEnvironmentHandler> Cannot find group %s in this project %s : %s", gr.Group.Name, proj.Name, errG)
				}
				if errA := group.InsertGroupInEnvironment(tx, env.ID, grID, gr.Permission); errA != nil {
					return sdk.WrapError(errA, "importGroupsInEnvironmentHandler> Cannot insert group %s in this environment %s", gr.Group.Name, env.Name)
				}
				env.EnvironmentGroups = append(env.EnvironmentGroups, sdk.GroupPermission{Group: sdk.Group{Name: gr.Group.Name}, Permission: gr.Permission})
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "importGroupsInEnvironmentHandler> Cannot commit transaction")
		}

		return WriteJSON(w, r, env, http.StatusOK)
	}
}
