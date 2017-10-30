package api

import (
	"context"
	"io/ioutil"
	"net/http"

	yaml "gopkg.in/yaml.v2"

	"github.com/gorilla/mux"
	"github.com/hashicorp/hcl"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func (api *API) updateGroupRoleOnPipelineHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		pipelineName := vars["permPipelineKey"]
		groupName := vars["group"]

		var groupPipeline sdk.GroupPermission
		if err := UnmarshalBody(r, &groupPipeline); err != nil {
			return sdk.WrapError(err, "updateGroupRoleOnPipelineHandler> cannot unmarshal request")
		}

		if groupName != groupPipeline.Group.Name {
			return sdk.ErrGroupNotFound
		}

		proj, errproj := project.Load(api.mustDB(), api.Cache, key, getUser(ctx))
		if errproj != nil {
			return sdk.WrapError(errproj, "updateGroupRoleOnPipelineHandler> unable to load project")
		}

		p, errLoadP := pipeline.LoadPipeline(api.mustDB(), key, pipelineName, false)
		if errLoadP != nil {
			return sdk.WrapError(errLoadP, "updateGroupRoleOnPipelineHandler: Cannot load %s", key)
		}

		g, errLoadG := group.LoadGroup(api.mustDB(), groupPipeline.Group.Name)
		if errLoadG != nil {
			return sdk.WrapError(errLoadG, "updateGroupRoleOnPipelineHandler: Cannot find %s", groupPipeline.Group.Name)
		}

		if group.IsDefaultGroupID(g.ID) && groupPipeline.Permission > permission.PermissionRead {
			return sdk.WrapError(sdk.ErrDefaultGroupPermission, "updateGroupRoleOnPipelineHandler: only read permission is allowed to default group")
		}

		groupInPipeline, errCheck := group.CheckGroupInPipeline(api.mustDB(), p.ID, g.ID)
		if errCheck != nil {
			return sdk.WrapError(sdk.ErrGroupNotFound, "updateGroupRoleOnPipelineHandler: Cannot check if group %s is already in the pipeline %s: %s", g.Name, p.Name, errCheck)
		}
		if !groupInPipeline {
			return sdk.WrapError(sdk.ErrGroupNotFound, "updateGroupRoleOnPipelineHandler: Cannot find group %s in pipeline %s", g.Name, p.Name)
		}

		if groupPipeline.Permission != permission.PermissionReadWriteExecute {
			permissions, err := group.LoadAllPipelineGroupByRole(api.mustDB(), p.ID, permission.PermissionReadWriteExecute)
			if err != nil {
				return sdk.WrapError(sdk.ErrGroupNeedWrite, "updateGroupRoleOnPipelineHandler: Cannot load groups for pipeline %s: %s", p.Name, err)
			}
			if len(permissions) == 1 && permissions[0].Group.ID == g.ID {
				return sdk.WrapError(sdk.ErrGroupNeedWrite, "updateGroupRoleOnPipelineHandler: Cannot remove write permission for group %s in pipeline %s", g.Name, p.Name)
			}
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "updateGroupRoleOnPipelineHandler: Cannot start transaction")
		}
		defer tx.Rollback()

		if err := group.UpdateGroupRoleInPipeline(tx, p.ID, g.ID, groupPipeline.Permission); err != nil {
			return sdk.WrapError(err, "updateGroupRoleOnPipelineHandler: Cannot add group %s in pipeline %s", g.Name, p.Name)
		}

		if err := pipeline.UpdatePipelineLastModified(tx, proj, p, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "updateGroupRoleOnPipelineHandler: Cannot update pipeline last_modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateGroupRoleOnPipelineHandler: Cannot start transaction")
		}

		if err := pipeline.LoadGroupByPipeline(api.mustDB(), p); err != nil {
			return sdk.WrapError(err, "updateGroupRoleOnPipelineHandler: Cannot load groups for pipeline %s", p.Name)
		}
		return WriteJSON(w, r, p, http.StatusOK)
	}
}

// DEPRECATED
func (api *API) updateGroupsOnPipelineHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		pipelineName := vars["permPipelineKey"]

		proj, errproj := project.Load(api.mustDB(), api.Cache, key, getUser(ctx))
		if errproj != nil {
			return sdk.WrapError(errproj, "updateGroupsOnPipelineHandler> unable to load project")
		}

		var groupsPermission []sdk.GroupPermission
		if err := UnmarshalBody(r, &groupsPermission); err != nil {
			return err
		}

		if len(groupsPermission) == 0 {
			return sdk.WrapError(sdk.ErrGroupNeedWrite, "updateGroupsOnPipelineHandler: Cannot remove all groups for pipeline %s", pipelineName)
		}

		found := false
		for _, gp := range groupsPermission {
			if gp.Permission == permission.PermissionReadWriteExecute {
				found = true
				break
			}
		}
		if !found {
			return sdk.WrapError(sdk.ErrGroupNeedWrite, "updateGroupsOnPipelineHandler: Need one group with write permission.")
		}

		p, errLoad := pipeline.LoadPipeline(api.mustDB(), key, pipelineName, false)
		if errLoad != nil {
			return sdk.WrapError(errLoad, "updateGroupsOnPipelineHandler: Cannot load %s", key)
		}

		tx, errb := api.mustDB().Begin()
		if errb != nil {
			return sdk.WrapError(errb, "updateGroupsOnPipelineHandler: Cannot start transaction")
		}
		defer tx.Rollback()

		if err := group.DeleteAllGroupFromPipeline(tx, p.ID); err != nil {
			return sdk.WrapError(err, "updateGroupsOnPipelineHandler: Cannot delete groups from pipeline %s", p.Name)
		}

		for _, g := range groupsPermission {
			groupData, err := group.LoadGroup(tx, g.Group.Name)
			if err != nil {
				return sdk.WrapError(err, "updateGroupsOnPipelineHandler: Cannot load group %s", g.Group.Name)
			}

			if err := group.InsertGroupInPipeline(tx, p.ID, groupData.ID, g.Permission); err != nil {
				return sdk.WrapError(err, "updateGroupsOnPipelineHandler: Cannot insert group %s in pipeline %s", g.Group.Name, p.Name)
			}
		}

		if err := pipeline.UpdatePipelineLastModified(tx, proj, p, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "updateGroupsOnPipelineHandler: Cannot update pipeline last_modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateGroupsOnPipelineHandler: Cannot commit transaction")
		}

		return nil
	}
}

func (api *API) addGroupInPipelineHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		pipelineName := vars["permPipelineKey"]

		proj, errproj := project.Load(api.mustDB(), api.Cache, key, getUser(ctx))
		if errproj != nil {
			return sdk.WrapError(errproj, "addGroupInPipelineHandler> unable to load project")
		}

		var groupPermission sdk.GroupPermission
		if err := UnmarshalBody(r, &groupPermission); err != nil {
			return err
		}

		p, err := pipeline.LoadPipeline(api.mustDB(), key, pipelineName, false)
		if err != nil {
			return sdk.WrapError(err, "addGroupInPipeline: Cannot load %s", key)
		}

		g, err := group.LoadGroup(api.mustDB(), groupPermission.Group.Name)
		if err != nil {
			return sdk.WrapError(err, "addGroupInPipeline: Cannot find %s", groupPermission.Group.Name)
		}

		if group.IsDefaultGroupID(g.ID) && groupPermission.Permission > permission.PermissionRead {
			return sdk.WrapError(sdk.ErrDefaultGroupPermission, "addGroupInPipeline: only read permission is allowed to default group")
		}

		groupInPipeline, err := group.CheckGroupInPipeline(api.mustDB(), p.ID, g.ID)
		if err != nil {
			return sdk.WrapError(err, "addGroupInPipeline: Cannot check if group %s is already in the pipeline %s", g.Name, p.Name)

		}
		if groupInPipeline {
			return sdk.WrapError(sdk.ErrGroupExists, "addGroupInPipeline: The group is already attached to the pipeline %s: %s", g.Name, p.Name, err)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "addGroupInPipeline: Cannot start transaction")
		}
		defer tx.Rollback()

		if err := group.InsertGroupInPipeline(tx, p.ID, g.ID, groupPermission.Permission); err != nil {
			return sdk.WrapError(err, "addGroupInPipeline: Cannot add group %s in pipeline %s", g.Name, p.Name)
		}

		if err := pipeline.UpdatePipelineLastModified(tx, proj, p, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "addGroupInPipeline: Cannot update pipeline last_modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "addGroupInPipeline: Cannot commit transaction")
		}

		if err := pipeline.LoadGroupByPipeline(api.mustDB(), p); err != nil {
			return sdk.WrapError(err, "addGroupInPipeline: Cannot load group")
		}
		return WriteJSON(w, r, p, http.StatusOK)
	}
}

func (api *API) importGroupsInPipeline() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get project name in URL
		vars := mux.Vars(r)
		key := vars["key"]
		pipelineName := vars["permPipelineKey"]
		format := r.FormValue("format")
		forceUpdate := FormBool(r, "forceUpdate")

		proj, errProj := project.Load(api.mustDB(), api.Cache, key, getUser(ctx), project.LoadOptions.WithGroups)
		if errProj != nil {
			return sdk.WrapError(errProj, "importGroupsInPipeline> Cannot load %s", key)
		}

		pip, err := pipeline.LoadPipeline(api.mustDB(), key, pipelineName, false)
		if err != nil {
			return sdk.WrapError(err, "addGroupInPipeline: Cannot load %s", key)
		}

		var groupsToAdd = []sdk.GroupPermission{}
		// Get body
		data, errRead := ioutil.ReadAll(r.Body)
		if errRead != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "importGroupsInPipeline> Unable to read body")
		}

		f, errF := exportentities.GetFormat(format)
		if errF != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "importGroupsInPipeline> Unable to get format")
		}

		var errorParse error
		switch f {
		case exportentities.FormatJSON, exportentities.FormatHCL:
			errorParse = hcl.Unmarshal(data, &groupsToAdd)
		case exportentities.FormatYAML:
			errorParse = yaml.Unmarshal(data, &groupsToAdd)
		}

		if errorParse != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "importGroupsInPipeline> Cannot parsing")
		}

		for _, gr := range groupsToAdd {
			exist := false
			for _, gro := range proj.ProjectGroups {
				if gr.Group.Name == gro.Group.Name {
					exist = true
				}
			}
			if !exist {
				return sdk.WrapError(sdk.ErrGroupNotFound, "importGroupsInPipeline> Group %v doesn't exist in this project", gr.Group.Name)
			}
		}

		tx, errBegin := api.mustDB().Begin()
		if errBegin != nil {
			return sdk.WrapError(errBegin, "importGroupsInPipeline> Cannot start transaction")
		}
		defer tx.Rollback()

		for _, gr := range groupsToAdd {
			groupID, errCheck := group.GetGroupInPipelineByName(tx, pip.ID, gr.Group.Name)
			if errCheck != nil {
				return sdk.WrapError(sdk.ErrGroupNotFound, "importGroupsInPipeline> Cannot check if group %s is already in the pipeline %s: %s", gr.Group.Name, pip.Name, errCheck)
			}
			if groupID != 0 && !forceUpdate {
				return sdk.WrapError(sdk.ErrGroupExists, "importGroupsInPipeline> Cannot find group %s in pipeline %s", gr.Group.Name, pip.Name)
			}
			if groupID != 0 && forceUpdate { // UPDATE
				if errU := group.UpdateGroupRoleInPipeline(tx, pip.ID, groupID, gr.Permission); errU != nil {
					return sdk.WrapError(errU, "importGroupsInPipeline> Cannot update group %s in pipeline %s", gr.Group.Name, pip.Name)
				}
			} else if groupID == 0 { //ADD
				grID, errG := group.GetIdByNameInList(proj.ProjectGroups, gr.Group.Name)
				if errG != nil {
					return sdk.WrapError(sdk.ErrGroupNotFound, "importGroupsInPipeline> Cannot find group %s in this project %s : %s", gr.Group.Name, proj.Name, errG)
				}
				if errA := group.InsertGroupInPipeline(tx, pip.ID, grID, gr.Permission); errA != nil {
					return sdk.WrapError(errA, "importGroupsInPipeline> Cannot insert group %s in this pipeline %s", gr.Group.Name, pip.Name)
				}
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "importGroupsInPipeline> Cannot commit transaction")
		}

		permissions := []sdk.GroupPermission{}
		for _, gr := range groupsToAdd {
			permissions = append(permissions, sdk.GroupPermission{Group: sdk.Group{Name: gr.Group.Name}, Permission: gr.Permission})
		}
		pip.GroupPermission = permissions

		return WriteJSON(w, r, pip, http.StatusOK)
	}
}

func (api *API) deleteGroupFromPipelineHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		pipelineName := vars["permPipelineKey"]
		groupName := vars["group"]

		p, err := pipeline.LoadPipeline(api.mustDB(), key, pipelineName, false)
		if err != nil {
			return sdk.WrapError(err, "deleteGroupFromPipelineHandler: Cannot load %s", key)
		}

		g, err := group.LoadGroup(api.mustDB(), groupName)
		if err != nil {
			return sdk.WrapError(err, "deleteGroupFromPipelineHandler: Cannot find %s", groupName)
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "deleteGroupFromPipelineHandler: Cannot start transaction")
		}
		defer tx.Rollback()

		if err := group.DeleteGroupFromPipeline(tx, p.ID, g.ID); err != nil {
			return sdk.WrapError(err, "deleteGroupFromPipelineHandler: Cannot delete group %s from project %s", g.Name, p.Name)
		}

		proj, errproj := project.Load(api.mustDB(), api.Cache, key, getUser(ctx))
		if errproj != nil {
			return sdk.WrapError(errproj, "deleteGroupFromPipelineHandler> unable to load project")
		}

		if err := pipeline.UpdatePipelineLastModified(tx, proj, p, getUser(ctx)); err != nil {
			return sdk.WrapError(err, "deleteGroupFromPipelineHandler: Cannot update pipeline last_modified date")
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "deleteGroupFromPipelineHandler: Cannot commit transaction")
		}

		if err := pipeline.LoadGroupByPipeline(api.mustDB(), p); err != nil {
			return sdk.WrapError(err, "deleteGroupFromPipelineHandler: Cannot load groups")
		}

		return WriteJSON(w, r, p, http.StatusOK)
	}
}
