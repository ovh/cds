package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func (api *API) deleteGroupFromProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		groupName := vars["group"]

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "deleteGroupFromProjectHandler: Cannot start transaction")
		}
		defer tx.Rollback()

		p, err := project.Load(tx, api.Cache, key, deprecatedGetUser(ctx), project.LoadOptions.WithGroups)
		if err != nil {
			return sdk.WrapError(err, "deleteGroupFromProjectHandler: Cannot load %s", key)
		}

		g, err := group.LoadGroup(tx, groupName)
		if err != nil {
			return sdk.WrapError(err, "deleteGroupFromProjectHandler: Cannot find %s", groupName)
		}

		var gp sdk.GroupPermission
		for _, gpp := range p.ProjectGroups {
			if gpp.Group.ID == g.ID {
				gp = gpp
				break
			}
		}
		if gp.Permission == 0 {
			return sdk.WrapError(sdk.ErrGroupNotFound, "deleteGroupFromProjectHandler: Group %s doesn't exist on project %s", groupName, p.Key)
		}

		if err := group.DeleteGroupFromProject(tx, p.ID, g.ID); err != nil {
			return sdk.WrapError(err, "deleteGroupFromProjectHandler: Cannot delete group %s from project %s", g.Name, p.Name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteGroupFromProjectHandler: Cannot commit transaction")
		}

		event.PublishDeleteProjectPermission(p, gp, deprecatedGetUser(ctx))

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) updateGroupRoleOnProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		groupName := vars["group"]
		onlyProject := FormBool(r, "onlyProject")

		var groupProject sdk.GroupPermission
		if err := service.UnmarshalBody(r, &groupProject); err != nil {
			return sdk.WrapError(err, "Unable to unmarshal")
		}

		if groupName != groupProject.Group.Name {
			return sdk.ErrGroupNotFound
		}

		tx, errb := api.mustDB().Begin()
		if errb != nil {
			return sdk.WrapError(errb, "updateGroupRoleHandler: Cannot start transaction")
		}
		defer tx.Rollback()

		p, errl := project.Load(tx, api.Cache, key, deprecatedGetUser(ctx), project.LoadOptions.WithGroups)
		if errl != nil {
			return sdk.WrapError(errl, "updateGroupRoleHandler: Cannot load project %s", key)
		}

		g, errlg := group.LoadGroup(tx, groupProject.Group.Name)
		if errlg != nil {
			return sdk.WrapError(errlg, "updateGroupRoleHandler: Cannot find %s", groupProject.Group.Name)
		}
		groupProject.Group.ID = g.ID

		var gpInProject sdk.GroupPermission
		nbGrpWrite := 0
		for _, gp := range p.ProjectGroups {
			if gp.Group.ID == g.ID {
				gpInProject = gp
			}
			if gp.Permission == permission.PermissionReadWriteExecute {
				nbGrpWrite++
			}
		}

		if gpInProject.Permission == 0 {
			return sdk.WrapError(sdk.ErrGroupNotFound, "updateGroupRoleHandler: Group is not attached to this project: %s", key)
		}

		if group.IsDefaultGroupID(g.ID) && groupProject.Permission > permission.PermissionRead {
			return sdk.WrapError(sdk.ErrDefaultGroupPermission, "updateGroupRoleHandler: only read permission is allowed to default group")
		}

		if groupProject.Permission != permission.PermissionReadWriteExecute {
			// If the updated group is the only one in write mode, return error
			if nbGrpWrite == 1 && gpInProject.Permission == permission.PermissionReadWriteExecute {
				return sdk.WrapError(sdk.ErrGroupNeedWrite, "updateGroupRoleHandler: Cannot remove write permission for this group %s on this project %s", g.Name, p.Name)
			}
		}

		if err := group.UpdateGroupRoleInProject(tx, p.ID, g.ID, groupProject.Permission); err != nil {
			return sdk.WrapError(err, "updateGroupRoleHandler: Cannot add group %s in project %s", g.Name, p.Name)
		}

		if !onlyProject {
			wfList, err := workflow.LoadAllNames(tx, p.ID, deprecatedGetUser(ctx))
			if err != nil {
				return sdk.WrapError(err, "cannot load all workflow names for project id %d key %s", p.ID, p.Key)
			}
			for _, wf := range wfList {
				role, errLoad := group.LoadRoleGroupInWorkflow(tx, wf.ID, groupProject.Group.ID)
				if errLoad != nil {
					if errLoad == sql.ErrNoRows {
						continue
					}
					return sdk.WrapError(errLoad, "cannot load role for workflow %s with id %d and group id %d", wf.Name, wf.ID, groupProject.Group.ID)
				}

				if gpInProject.Permission != role { // If project role and workflow role aren't sync do not update
					continue
				}

				if err := group.UpdateWorkflowGroup(tx, &sdk.Workflow{ID: wf.ID, ProjectID: p.ID}, groupProject); err != nil {
					return sdk.WrapError(err, "cannot update group %d in workflow %s with id %d", groupProject.Group.ID, wf.Name, wf.ID)
				}
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateGroupRoleHandler: Cannot start transaction")
		}

		newGP := sdk.GroupPermission{
			Permission: groupProject.Permission,
			Group:      gpInProject.Group,
		}
		event.PublishUpdateProjectPermission(p, newGP, gpInProject, deprecatedGetUser(ctx))

		return service.WriteJSON(w, groupProject, http.StatusOK)
	}
}

func (api *API) addGroupInProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		onlyProject := FormBool(r, "onlyProject")

		var groupProject sdk.GroupPermission
		if err := service.UnmarshalBody(r, &groupProject); err != nil {
			return sdk.WrapError(err, "Unable to unmarshal")
		}

		p, errl := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx))
		if errl != nil {
			return sdk.WrapError(errl, "AddGroupInProject: Cannot load %s", key)
		}

		g, errlg := group.LoadGroup(api.mustDB(), groupProject.Group.Name)
		if errlg != nil {
			return sdk.WrapError(errlg, "AddGroupInProject: Cannot find %s", groupProject.Group.Name)
		}
		groupProject.Group.ID = g.ID

		groupInProject, errc := group.CheckGroupInProject(api.mustDB(), p.ID, g.ID)
		if errc != nil {
			return sdk.WrapError(errc, "AddGroupInProject: Cannot check if group %s is already in the project %s", g.Name, p.Name)
		}
		if groupInProject {
			return sdk.WrapError(sdk.ErrGroupExists, "AddGroupInProject: Group already in the project %s", p.Name)
		}

		if group.IsDefaultGroupID(g.ID) && groupProject.Permission > permission.PermissionRead {
			return sdk.WrapError(sdk.ErrDefaultGroupPermission, "AddGroupInProject: only read permission is allowed to default group")
		}

		tx, errb := api.mustDB().Begin()
		if errb != nil {
			return sdk.WrapError(errb, "AddGroupInProject: Cannot open transaction")
		}
		defer tx.Rollback()

		if err := group.InsertGroupInProject(tx, p.ID, g.ID, groupProject.Permission); err != nil {
			return sdk.WrapError(err, "AddGroupInProject: Cannot add group %s in project %s", g.Name, p.Name)
		}

		if !onlyProject {
			wfList, err := workflow.LoadAllNames(tx, p.ID, deprecatedGetUser(ctx))
			if err != nil {
				return sdk.WrapError(err, "cannot load all workflow names for project id %d key %s", p.ID, p.Key)
			}
			for _, wf := range wfList {
				if err := group.UpsertWorkflowGroup(tx, p.ID, wf.ID, groupProject); err != nil {
					return sdk.WrapError(err, "cannot upsert group %d in workflow %s with id %d", groupProject.Group.ID, wf.Name, wf.ID)
				}
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "AddGroupInProject: Cannot commit transaction")
		}

		event.PublishAddProjectPermission(p, groupProject, deprecatedGetUser(ctx))

		if err := group.LoadGroupByProject(api.mustDB(), p); err != nil {
			return sdk.WrapError(err, "AddGroupInProject: Cannot load groups on project %s", p.Key)
		}

		return service.WriteJSON(w, p.ProjectGroups, http.StatusOK)
	}
}

func (api *API) importGroupsInProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		format := r.FormValue("format")
		forceUpdate := FormBool(r, "forceUpdate")

		proj, errProj := project.Load(api.mustDB(), api.Cache, key, deprecatedGetUser(ctx), project.LoadOptions.WithGroups)
		if errProj != nil {
			return sdk.WrapError(errProj, "importGroupsInProjectHandler> Cannot load %s", key)
		}

		groupsToAdd := []sdk.GroupPermission{}
		// Get body
		data, errRead := ioutil.ReadAll(r.Body)
		if errRead != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "importGroupsInProjectHandler> Unable to read body")
		}

		f, errF := exportentities.GetFormat(format)
		if errF != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "importGroupsInProjectHandler> Unable to get format")
		}

		var errorParse error
		switch f {
		case exportentities.FormatJSON:
			errorParse = json.Unmarshal(data, &groupsToAdd)
		case exportentities.FormatYAML:
			errorParse = yaml.Unmarshal(data, &groupsToAdd)
		}

		if errorParse != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "importGroupsInProjectHandler> Cannot parsing")
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "importGroupsInProjectHandler> Cannot start transaction")
		}
		defer tx.Rollback()

		if forceUpdate {
			if err := group.DeleteAllGroupFromProject(tx, proj.ID); err != nil {
				return sdk.WrapError(err, "Cannot delete all groups for this project %s", proj.Name)
			}
			proj.ProjectGroups = []sdk.GroupPermission{}
		} else {
			for _, gr := range groupsToAdd {
				exist := false
				for _, gro := range proj.ProjectGroups {
					if gr.Group.Name == gro.Group.Name {
						exist = true
					}
				}
				if exist {
					return sdk.WrapError(sdk.ErrGroupExists, "importGroupsInProjectHandler> Group %s in project %s", gr.Group.Name, proj.Name)
				}
			}
		}

		for _, gr := range groupsToAdd {
			gro, errG := group.LoadGroup(tx, gr.Group.Name)
			if errG != nil {
				return sdk.WrapError(sdk.ErrGroupNotFound, "importGroupsInProjectHandler> Group %v doesn't exist", gr.Group.Name)
			}
			if err := group.InsertGroupInProject(tx, proj.ID, gro.ID, gr.Permission); err != nil {
				return sdk.WrapError(err, "Cannot add group %v in project %s", gr.Group.Name, proj.Name)
			}
			gr.Group = *gro
			proj.ProjectGroups = append(proj.ProjectGroups, gr)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		return service.WriteJSON(w, proj, http.StatusOK)
	}
}
