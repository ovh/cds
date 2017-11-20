package api

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func (api *API) updateGroupRoleOnApplicationHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]
		groupName := vars["group"]

		var groupApplication sdk.GroupPermission
		if err := UnmarshalBody(r, &groupApplication); err != nil {
			return err
		}

		app, errload := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx))
		if errload != nil {
			return sdk.WrapError(errload, "updateGroupRoleOnApplicationHandler: Cannot load application %s", appName)
		}

		g, errLoadGroup := group.LoadGroup(api.mustDB(), groupName)
		if errLoadGroup != nil {
			return sdk.WrapError(errLoadGroup, "updateGroupRoleOnApplicationHandler: Cannot load group %s", groupName)
		}

		if group.IsDefaultGroupID(g.ID) && groupApplication.Permission > permission.PermissionRead {
			return sdk.WrapError(sdk.ErrDefaultGroupPermission, "updateGroupRoleOnApplicationHandler: only read permission is allowed to default group")
		}

		if groupApplication.Permission != permission.PermissionReadWriteExecute {
			permissions, err := group.LoadAllApplicationGroupByRole(api.mustDB(), app.ID, permission.PermissionReadWriteExecute)
			if err != nil {
				return sdk.WrapError(err, "updateGroupRoleOnApplicationHandler: Cannot load group for application %s", appName)
			}

			if len(permissions) == 1 && permissions[0].Group.ID == g.ID {
				return sdk.WrapError(sdk.ErrGroupNeedWrite, "updateGroupRoleOnApplicationHandler: Cannot remove write permission for group %s in application %s", groupName, appName)
			}
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "updateGroupRoleOnApplicationHandler: Cannot start transaction")
		}
		defer tx.Rollback()

		if err := group.UpdateGroupRoleInApplication(tx, app.ID, g.ID, groupApplication.Permission); err != nil {
			return sdk.WrapError(err, "updateGroupRoleOnApplicationHandler: Cannot update permission for group %s in application %s", groupName, appName)
		}

		if err := application.UpdateLastModified(tx, api.Cache, app, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "updateGroupsInApplicationHandler: Cannot update last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateGroupRoleOnApplicationHandler: Cannot commit transaction")
		}

		if err := application.LoadGroupByApplication(api.mustDB(), app); err != nil {
			return sdk.WrapError(err, "updateGroupRoleOnApplicationHandler: Cannot load application groups")
		}

		return WriteJSON(w, r, app, http.StatusOK)
	}
}

// Deprecated
func (api *API) updateGroupsInApplicationHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]

		proj, errload := project.Load(api.mustDB(), api.Cache, key, getUser(ctx))
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

		app, errLoadName := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx))
		if errLoadName != nil {
			return sdk.WrapError(errLoadName, "updateGroupsInApplicationHandler: Cannot load application %s: %s", appName)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "updateGroupsInApplicationHandler: Cannot start transaction")
		}
		defer tx.Rollback()

		if err := group.DeleteAllGroupFromApplication(tx, app.ID); err != nil {
			return sdk.WrapError(err, "updateGroupsInApplicationHandler: Cannot delete groups from application %s", appName)
		}

		if err := application.AddGroup(tx, api.Cache, proj, app, getUser(ctx), groupsPermission...); err != nil {
			return sdk.WrapError(err, "updateGroupsInApplicationHandler: Cannot add groups in application %s", app.Name)
		}

		if err := application.UpdateLastModified(tx, api.Cache, app, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "updateGroupsInApplicationHandler: Cannot update last modified date")
		}

		if err := project.UpdateLastModified(tx, api.Cache, getUser(ctx), proj); err != nil {
			return sdk.WrapError(err, "updateGroupsInApplicationHandler: Cannot update last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateGroupsInApplicationHandler: Cannot commit transaction")
		}

		return WriteJSON(w, r, app, http.StatusOK)
	}
}

func (api *API) addGroupInApplicationHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]

		var groupPermission sdk.GroupPermission
		if err := UnmarshalBody(r, &groupPermission); err != nil {
			return sdk.WrapError(err, "addGroupInApplicationHandler> Cannot unmarshal request")
		}

		proj, err := project.Load(api.mustDB(), api.Cache, key, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "addGroupInApplicationHandler> Cannot load %s", key)
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "addGroupInApplicationHandler> Cannot load %s", appName)
		}

		g, err := group.LoadGroup(api.mustDB(), groupPermission.Group.Name)
		if err != nil {
			return sdk.WrapError(err, "addGroupInApplicationHandler> Cannot find %s", groupPermission.Group.Name)
		}

		if group.IsDefaultGroupID(g.ID) && groupPermission.Permission > permission.PermissionRead {
			return sdk.WrapError(sdk.ErrDefaultGroupPermission, "addGroupInApplicationHandler: only read permission is allowed to default group")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "addGroupInApplicationHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if err := application.AddGroup(tx, api.Cache, proj, app, getUser(ctx), groupPermission); err != nil {
			return sdk.WrapError(err, "addGroupInApplicationHandler> Cannot add group %s in application %s", g.Name, app.Name)
		}

		if err := application.UpdateLastModified(tx, api.Cache, app, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "addGroupInApplicationHandler> Cannot update application last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addGroupInApplicationHandler> Cannot commit transaction")
		}

		if err := application.LoadGroupByApplication(api.mustDB(), app); err != nil {
			return sdk.WrapError(err, "addGroupInApplicationHandler> Cannot load application groups")
		}

		return WriteJSON(w, r, app, http.StatusOK)
	}
}

func (api *API) deleteGroupFromApplicationHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]
		groupName := vars["group"]

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "deleteGroupFromApplicationHandler: Cannot load application %s", appName)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "deleteGroupFromApplicationHandler: Cannot start transaction")
		}
		defer tx.Rollback()

		if err := group.DeleteGroupFromApplication(tx, key, appName, groupName); err != nil {
			return sdk.WrapError(err, "deleteGroupFromApplicationHandler: Cannot delete group %s from pipeline %s", groupName, appName)
		}

		if err := application.UpdateLastModified(tx, api.Cache, app, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "deleteGroupFromApplicationHandler: Cannot update application last modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteGroupFromApplicationHandler: Cannot commit transaction")
		}

		if err := application.LoadGroupByApplication(api.mustDB(), app); err != nil {
			return sdk.WrapError(err, "deleteGroupFromApplicationHandler: Cannot load application groups")
		}

		return WriteJSON(w, r, app, http.StatusOK)
	}
}

func (api *API) importGroupsInApplicationHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		appName := vars["permApplicationName"]
		format := r.FormValue("format")
		forceUpdate := FormBool(r, "forceUpdate")

		proj, errProj := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.WithGroups)
		if errProj != nil {
			return sdk.WrapError(errProj, "importGroupsInApplicationHandler> Cannot load %s", key)
		}

		app, err := application.LoadByName(api.mustDB(), api.Cache, key, appName, getUser(ctx), application.LoadOptions.WithGroups)
		if err != nil {
			return sdk.WrapError(err, "importGroupsInApplicationHandler> Cannot load %s", key)
		}

		groupsToAdd := []sdk.GroupPermission{}
		// Get body
		data, errRead := ioutil.ReadAll(r.Body)
		if errRead != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "importGroupsInApplicationHandler> Unable to read body")
		}

		f, errF := exportentities.GetFormat(format)
		if errF != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "importGroupsInApplicationHandler> Unable to get format")
		}

		var errorParse error
		switch f {
		case exportentities.FormatJSON:
			errorParse = json.Unmarshal(data, &groupsToAdd)
		case exportentities.FormatYAML:
			errorParse = yaml.Unmarshal(data, &groupsToAdd)
		}

		if errorParse != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "importGroupsInApplicationHandler> Cannot parsing")
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
				return sdk.WrapError(sdk.ErrGroupNotFound, "importGroupsInApplicationHandler> Group %v doesn't exist in this project", gr.Group.Name)
			} else if !exist && forceUpdate {
				groupsToAddInProj = append(groupsToAddInProj, sdk.GroupPermission{
					Group:      gr.Group,
					Permission: permission.PermissionRead,
				})
			}
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "importGroupsInApplicationHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if forceUpdate { // clean and update
			for _, gr := range groupsToAddInProj {
				gro, errG := group.LoadGroup(tx, gr.Group.Name)
				if errG != nil {
					return sdk.WrapError(sdk.ErrGroupNotFound, "importGroupsInApplicationHandler> Group %v doesn't exist", gr.Group.Name)
				}
				if err := group.InsertGroupInProject(tx, proj.ID, gro.ID, gr.Permission); err != nil {
					return sdk.WrapError(err, "importGroupsInApplicationHandler> Cannot add group %v in project %s", gr.Group.Name, proj.Name)
				}
				gr.Group = *gro
				proj.ProjectGroups = append(proj.ProjectGroups, gr)
			}

			if err := group.DeleteAllGroupFromApplication(tx, app.ID); err != nil {
				return sdk.WrapError(err, "importGroupsInApplicationHandler> Cannot delete all groups for this application %s", app.Name)
			}

			app.ApplicationGroups = []sdk.GroupPermission{}
			for _, gr := range groupsToAdd {
				gro, errG := group.LoadGroup(tx, gr.Group.Name)
				if errG != nil {
					return sdk.WrapError(sdk.ErrGroupNotFound, "importGroupsInApplicationHandler> Cannot load group %s : %s", gr.Group.Name, errG)
				}
				if err := group.InsertGroupInApplication(tx, app.ID, gro.ID, gr.Permission); err != nil {
					return sdk.WrapError(err, "importGroupsInApplicationHandler> Cannot insert group %s in this application %s", gr.Group.Name, app.Name)
				}
				app.ApplicationGroups = append(app.ApplicationGroups, sdk.GroupPermission{Group: sdk.Group{Name: gr.Group.Name}, Permission: gr.Permission})
			}
		} else { // add new group
			for _, gr := range groupsToAdd {
				if _, errGr := group.GetIDByNameInList(app.ApplicationGroups, gr.Group.Name); errGr == nil {
					return sdk.WrapError(sdk.ErrGroupExists, "importGroupsInApplicationHandler> Group %s in application %s", gr.Group.Name, app.Name)
				}

				grID, errG := group.GetIDByNameInList(proj.ProjectGroups, gr.Group.Name)
				if errG != nil {
					return sdk.WrapError(sdk.ErrGroupNotFound, "importGroupsInApplicationHandler> Cannot find group %s in this project %s : %s", gr.Group.Name, proj.Name, errG)
				}
				if errA := group.InsertGroupInApplication(tx, app.ID, grID, gr.Permission); errA != nil {
					return sdk.WrapError(errA, "importGroupsInApplicationHandler> Cannot insert group %s in this application %s", gr.Group.Name, app.Name)
				}
				app.ApplicationGroups = append(app.ApplicationGroups, sdk.GroupPermission{Group: sdk.Group{Name: gr.Group.Name}, Permission: gr.Permission})
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "importGroupsInApplicationHandler> Cannot commit transaction")
		}

		return WriteJSON(w, r, app, http.StatusOK)
	}
}
