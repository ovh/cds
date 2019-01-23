package api

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) updateGroupRoleOnEnvironmentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		envName := vars["permEnvironmentName"]
		groupName := vars["group"]

		var groupEnvironment sdk.GroupPermission
		if err := service.UnmarshalBody(r, &groupEnvironment); err != nil {
			return sdk.WrapError(err, "Cannot read body")
		}

		g, errG := group.LoadGroup(api.mustDB(), groupName)
		if errG != nil {
			return sdk.WrapError(errG, "Cannot load group %s", groupName)
		}

		if group.IsDefaultGroupID(g.ID) && groupEnvironment.Permission > permission.PermissionRead {
			return sdk.WrapError(sdk.ErrDefaultGroupPermission, "Only read permission is allowed to default group")
		}

		env, errE := environment.LoadEnvironmentByName(api.mustDB(), key, envName)
		if errE != nil {
			return sdk.WrapError(errE, "Cannot load environment %s", envName)
		}

		var writeGroupID int64
		var gpOld sdk.GroupPermission
		for _, gp := range env.EnvironmentGroups {
			if gp.Permission == permission.PermissionReadWriteExecute && gp.Group.ID != g.ID {
				writeGroupID = gp.Group.ID
			}
			if gp.Group.ID == g.ID {
				gpOld = gp
			}
		}

		if groupEnvironment.Permission != permission.PermissionReadWriteExecute {
			if writeGroupID == 0 {
				log.Warning("updateGroupRoleOnEnvironmentHandler: Cannot remove write permission on group %s for environment %s", groupName, envName)
				return sdk.WrapError(sdk.ErrGroupNeedWrite, "Cannot remove write permission on group %s for environment %s", groupName, envName)
			}
		}

		tx, errB := api.mustDB().Begin()
		if errB != nil {
			return sdk.WrapError(errB, "Cannot start transaction")
		}
		defer tx.Rollback()

		if err := group.UpdateGroupRoleInEnvironment(tx, env.ID, g.ID, groupEnvironment.Permission); err != nil {
			return sdk.WrapError(err, "Cannot update permission for group %s in environment %s", groupName, envName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		envUpdated, errE := environment.LoadEnvironmentByName(api.mustDB(), key, envName)
		if errE != nil {
			return sdk.WrapError(errE, "Cannot load updated environment")
		}
		envUpdated.Permission = permission.EnvironmentPermission(key, envUpdated.Name, deprecatedGetUser(ctx))
		envUpdated.ProjectKey = key

		groupEnvironment.Group = *g
		event.PublishEnvironmentPermissionUpdate(key, *envUpdated, groupEnvironment, gpOld, deprecatedGetUser(ctx))

		return service.WriteJSON(w, envUpdated, http.StatusOK)
	}
}

func (api *API) addGroupsInEnvironmentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		envName := vars["permEnvironmentName"]

		var groupPermission []sdk.GroupPermission
		if err := service.UnmarshalBody(r, &groupPermission); err != nil {
			return sdk.WrapError(err, "Cannot read body")
		}

		env, err := environment.LoadEnvironmentByName(api.mustDB(), key, envName)
		if err != nil {
			return sdk.WrapError(err, "Cannot load environment %s", envName)
		}

		tx, errB := api.mustDB().Begin()
		if errB != nil {
			return sdk.WrapError(errB, "Cannot start transaction")
		}
		defer tx.Rollback()

		for _, gp := range groupPermission {
			g, errL := group.LoadGroup(tx, gp.Group.Name)
			if errL != nil {
				return sdk.WrapError(errL, "Cannot find group %s", gp.Group.Name)
			}

			if group.IsDefaultGroupID(g.ID) && gp.Permission > permission.PermissionRead {
				return sdk.WrapError(sdk.ErrDefaultGroupPermission, "Only read permission is allowed to default group")
			}

			alreadyAdded, errA := group.IsInEnvironment(tx, env.ID, g.ID)
			if errA != nil {
				return sdk.WrapError(errA, "Cannot check if group is in env")
			}

			if alreadyAdded {
				return sdk.WrapError(sdk.ErrGroupPresent, "Group %s already in environment %s", g.Name, env.Name)
			}

			if err := group.InsertGroupInEnvironment(tx, env.ID, g.ID, gp.Permission); err != nil {
				return sdk.WrapError(err, "Cannot add group %s in environment %s", g.Name, env.Name)
			}
		}

		// Update last modified on environment
		if err := environment.UpdateEnvironment(tx, env); err != nil {
			return sdk.WrapError(err, "Cannot update environment %s", env.Name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		envUpdated, errL := environment.LoadEnvironmentByName(api.mustDB(), key, envName)
		if errL != nil {
			return sdk.WrapError(errL, "Cannot load updated environment")
		}
		envUpdated.Permission = permission.EnvironmentPermission(key, envUpdated.Name, deprecatedGetUser(ctx))
		envUpdated.ProjectKey = key

		for _, gp := range groupPermission {
			event.PublishEnvironmentPermissionAdd(key, *envUpdated, gp, deprecatedGetUser(ctx))
		}

		return service.WriteJSON(w, envUpdated, http.StatusOK)
	}
}

func (api *API) addGroupInEnvironmentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		envName := vars["permEnvironmentName"]

		var groupPermission sdk.GroupPermission
		if err := service.UnmarshalBody(r, &groupPermission); err != nil {
			return err
		}

		env, err := environment.LoadEnvironmentByName(api.mustDB(), key, envName)
		if err != nil {
			return sdk.WrapError(err, "Cannot load %s", envName)
		}

		g, err := group.LoadGroup(api.mustDB(), groupPermission.Group.Name)
		if err != nil {
			return sdk.WrapError(err, "Cannot find %s", groupPermission.Group.Name)
		}

		if group.IsDefaultGroupID(g.ID) && groupPermission.Permission > permission.PermissionRead {
			return sdk.WrapError(sdk.ErrDefaultGroupPermission, "Only read permission is allowed to default group")
		}

		alreadyAdded, err := group.IsInEnvironment(api.mustDB(), env.ID, g.ID)
		if err != nil {
			return sdk.WrapError(err, "Cannot check if group is in env")
		}

		if alreadyAdded {
			return sdk.WrapError(sdk.ErrGroupPresent, "Group %s is already present in env %s", g.Name, env.Name)
		}

		if err := group.InsertGroupInEnvironment(api.mustDB(), env.ID, g.ID, groupPermission.Permission); err != nil {
			return sdk.WrapError(err, "Cannot add group %s in environment %s", g.Name, env.Name)
		}

		groupPermission.Group = *g
		event.PublishEnvironmentPermissionAdd(key, *env, groupPermission, deprecatedGetUser(ctx))

		return service.WriteJSON(w, nil, http.StatusCreated)
	}
}

func (api *API) deleteGroupFromEnvironmentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		envName := vars["permEnvironmentName"]
		groupName := vars["group"]

		proj, errP := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx))
		if errP != nil {
			return sdk.WrapError(errP, "Cannot load project")
		}

		env, errE := environment.LoadEnvironmentByName(api.mustDB(), proj.Key, envName)
		if errE != nil {
			return sdk.WrapError(errE, "Cannot load environment")
		}

		g, errG := group.LoadGroup(api.mustDB(), groupName)
		if errG != nil {
			return sdk.WrapError(errG, "Cannot load group")
		}

		var gp sdk.GroupPermission
		for _, groupPerm := range env.EnvironmentGroups {
			if groupPerm.Group.ID == g.ID {
				gp = groupPerm
				break
			}
		}

		tx, errT := api.mustDB().Begin()
		if errT != nil {
			return sdk.WrapError(errT, "Cannot start transaction")
		}
		defer tx.Rollback()

		if err := group.DeleteGroupFromEnvironment(tx, env.ID, g.ID); err != nil {
			return sdk.WrapError(err, "Cannot delete group %s from pipeline %s", groupName, envName)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(errT, "Cannot commit transaction")
		}

		event.PublishEnvironmentPermissionDelete(key, *env, gp, deprecatedGetUser(ctx))

		return nil
	}
}

func (api *API) importGroupsInEnvironmentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		envName := vars["permEnvironmentName"]
		format := r.FormValue("format")
		forceUpdate := FormBool(r, "forceUpdate")

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "Cannot start transaction")
		}
		defer tx.Rollback()

		proj, errProj := project.Load(tx, api.Cache, key, deprecatedGetUser(ctx), project.LoadOptions.WithGroups)
		if errProj != nil {
			return sdk.WrapError(errProj, "Cannot load project %s", key)
		}

		env, errE := environment.LoadEnvironmentByName(tx, key, envName)
		if errE != nil {
			return sdk.WrapError(errE, "Cannot load environment %s", envName)
		}

		groupsToAdd := []sdk.GroupPermission{}
		// Get body
		data, errRead := ioutil.ReadAll(r.Body)
		if errRead != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "Unable to read body")
		}

		f, errF := exportentities.GetFormat(format)
		if errF != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "Unable to get format")
		}

		var errorParse error
		switch f {
		case exportentities.FormatJSON:
			errorParse = json.Unmarshal(data, &groupsToAdd)
		case exportentities.FormatYAML:
			errorParse = yaml.Unmarshal(data, &groupsToAdd)
		}

		if errorParse != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "Cannot parsing")
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
				return sdk.WrapError(sdk.ErrGroupNotFound, "Group %v doesn't exist in this project", gr.Group.Name)
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
					return sdk.WrapError(sdk.ErrGroupNotFound, "Group %v doesn't exist", gr.Group.Name)
				}
				if err := group.InsertGroupInProject(tx, proj.ID, gro.ID, gr.Permission); err != nil {
					return sdk.WrapError(err, "Cannot add group %v in project %s", gr.Group.Name, proj.Name)
				}
				gr.Group = *gro
				proj.ProjectGroups = append(proj.ProjectGroups, gr)
			}

			if err := group.DeleteAllGroupFromEnvironment(tx, env.ID); err != nil {
				return sdk.WrapError(err, "Cannot delete all groups for this environment %s", env.Name)
			}

			env.EnvironmentGroups = []sdk.GroupPermission{}
			for _, gr := range groupsToAdd {
				gro, errG := group.LoadGroup(tx, gr.Group.Name)
				if errG != nil {
					return sdk.WrapError(sdk.ErrGroupNotFound, "Cannot load group %s : %s", gr.Group.Name, errG)
				}
				if err := group.InsertGroupInEnvironment(tx, env.ID, gro.ID, gr.Permission); err != nil {
					return sdk.WrapError(err, "Cannot insert group %s in this environment %s", gr.Group.Name, env.Name)
				}
				env.EnvironmentGroups = append(env.EnvironmentGroups, sdk.GroupPermission{Group: sdk.Group{Name: gr.Group.Name}, Permission: gr.Permission})
			}
		} else { // add new group
			for _, gr := range groupsToAdd {
				if _, errGr := group.GetIDByNameInList(env.EnvironmentGroups, gr.Group.Name); errGr == nil {
					return sdk.WrapError(sdk.ErrGroupExists, "Group %s in environment %s", gr.Group.Name, env.Name)
				}

				grID, errG := group.GetIDByNameInList(proj.ProjectGroups, gr.Group.Name)
				if errG != nil {
					return sdk.WrapError(sdk.ErrGroupNotFound, "Cannot find group %s in this project %s : %s", gr.Group.Name, proj.Name, errG)
				}
				if errA := group.InsertGroupInEnvironment(tx, env.ID, grID, gr.Permission); errA != nil {
					return sdk.WrapError(errA, "Cannot insert group %s in this environment %s", gr.Group.Name, env.Name)
				}
				env.EnvironmentGroups = append(env.EnvironmentGroups, sdk.GroupPermission{Group: sdk.Group{Name: gr.Group.Name}, Permission: gr.Permission})
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		return service.WriteJSON(w, env, http.StatusOK)
	}
}
