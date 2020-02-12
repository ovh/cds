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
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func (api *API) deleteGroupFromProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		groupName := vars["groupName"]

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		proj, err := project.Load(tx, api.Cache, key)
		if err != nil {
			return sdk.WrapError(err, "cannot load %s", key)
		}

		grp, err := group.LoadByName(ctx, tx, groupName)
		if err != nil {
			return sdk.WrapError(err, "cannot find %s", groupName)
		}

		link, err := group.LoadLinkGroupProjectForGroupIDAndProjectID(ctx, tx, grp.ID, proj.ID)
		if err != nil {
			return err
		}

		if err := group.DeleteLinkGroupProject(tx, link); err != nil {
			return sdk.WrapError(err, "cannot delete group %s from project %s", grp.Name, proj.Name)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "cannot commit transaction")
		}

		event.PublishDeleteProjectPermission(ctx, proj, sdk.GroupPermission{
			Group:      *grp,
			Permission: link.Role,
		})

		return service.WriteJSON(w, nil, http.StatusOK)
	}
}

func (api *API) putGroupRoleOnProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		groupName := vars["groupName"]
		onlyProject := FormBool(r, "onlyProject")

		var data sdk.GroupPermission
		if err := service.UnmarshalBody(r, &data); err != nil {
			return err
		}
		if err := data.IsValid(); err != nil {
			return err
		}

		tx, errb := api.mustDB().Begin()
		if errb != nil {
			return sdk.WrapError(errb, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		proj, err := project.Load(tx, api.Cache, key)
		if err != nil {
			return sdk.WrapError(err, "cannot load project %s", key)
		}

		grp, err := group.LoadByName(ctx, tx, groupName)
		if err != nil {
			return sdk.WrapError(err, "cannot find %s", groupName)
		}

		if group.IsDefaultGroupID(grp.ID) && data.Permission > sdk.PermissionRead {
			return sdk.NewErrorFrom(sdk.ErrDefaultGroupPermission, "only read permission is allowed to default group")
		}

		oldLink, err := group.LoadLinkGroupProjectForGroupIDAndProjectID(ctx, tx, grp.ID, proj.ID)
		if err != nil {
			return err
		}

		newLink := *oldLink
		newLink.Role = data.Permission

		if err := group.UpdateLinkGroupProject(tx, &newLink); err != nil {
			return err
		}

		if !onlyProject {
			wfList, err := workflow.LoadAllNames(tx, proj.ID)
			if err != nil {
				return sdk.WrapError(err, "cannot load all workflow names for project id %d key %s", proj.ID, proj.Key)
			}
			for _, wf := range wfList {
				role, err := group.LoadRoleGroupInWorkflow(tx, wf.ID, grp.ID)
				if err != nil {
					if err == sdk.Cause(sql.ErrNoRows) {
						continue
					}
					return sdk.WrapError(err, "cannot load role for workflow %s with id %d and group id %d", wf.Name, wf.ID, grp.ID)
				}

				if oldLink.Role != role { // If project role and workflow role aren't sync do not update
					continue
				}

				if err := group.UpdateWorkflowGroup(ctx, tx,
					&sdk.Workflow{ID: wf.ID, ProjectID: proj.ID},
					sdk.GroupPermission{Group: *grp, Permission: newLink.Role},
				); err != nil {
					return sdk.WrapError(err, "cannot update group %d in workflow %s with id %d", grp.ID, wf.Name, wf.ID)
				}
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "updateGroupRoleHandler: Cannot start transaction")
		}

		newGroupPermission := sdk.GroupPermission{Permission: newLink.Role, Group: *grp}
		event.PublishUpdateProjectPermission(ctx, proj, newGroupPermission,
			sdk.GroupPermission{Permission: oldLink.Role, Group: *grp},
			getAPIConsumer(ctx))

		return service.WriteJSON(w, newGroupPermission, http.StatusOK)
	}
}

func (api *API) postGroupInProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		onlyProject := FormBool(r, "onlyProject")

		var data sdk.GroupPermission
		if err := service.UnmarshalBody(r, &data); err != nil {
			return err
		}
		if err := data.IsValid(); err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot open transaction")
		}
		defer tx.Rollback() // nolint

		proj, err := project.Load(tx, api.Cache, key)
		if err != nil {
			return sdk.WrapError(err, "cannot load %s", key)
		}

		grp, err := group.LoadByName(ctx, tx, data.Group.Name)
		if err != nil {
			return sdk.WrapError(err, "cannot find %s", data.Group.Name)
		}

		if group.IsDefaultGroupID(grp.ID) && data.Permission > sdk.PermissionRead {
			return sdk.NewErrorFrom(sdk.ErrDefaultGroupPermission, "only read permission is allowed to default group")
		}

		link, err := group.LoadLinkGroupProjectForGroupIDAndProjectID(ctx, tx, grp.ID, proj.ID)
		if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
			return err
		}
		if link != nil {
			return sdk.NewErrorFrom(sdk.ErrGroupExists, "group already in the project %s", proj.Name)
		}

		newLink := group.LinkGroupProject{
			GroupID:   grp.ID,
			ProjectID: proj.ID,
			Role:      data.Permission,
		}
		if err := group.InsertLinkGroupProject(ctx, tx, &newLink); err != nil {
			return err
		}

		if !onlyProject {
			wfList, err := workflow.LoadAllNames(tx, proj.ID)
			if err != nil {
				return sdk.WrapError(err, "cannot load all workflow names for project id %d key %s", proj.ID, proj.Key)
			}
			for _, wf := range wfList {
				if err := group.UpsertWorkflowGroup(tx, proj.ID, wf.ID, sdk.GroupPermission{
					Group:      *grp,
					Permission: data.Permission,
				}); err != nil {
					return sdk.WrapError(err, "cannot upsert group %d in workflow %s with id %d", grp.ID, wf.Name, wf.ID)
				}
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "cannot commit transaction")
		}

		newGroupPermission := sdk.GroupPermission{Permission: newLink.Role, Group: *grp}
		event.PublishAddProjectPermission(ctx, proj, newGroupPermission, getAPIConsumer(ctx))

		return service.WriteJSON(w, newGroupPermission, http.StatusOK)
	}
}

func (api *API) postImportGroupsInProjectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars[permProjectKey]
		format := r.FormValue("format")
		forceUpdate := FormBool(r, "forceUpdate")

		proj, err := project.Load(api.mustDB(), api.Cache, key)
		if err != nil {
			return sdk.WrapError(err, "cannot load project %s", key)
		}

		var data []sdk.GroupPermission
		buf, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to read body"))
		}
		f, err := exportentities.GetFormat(format)
		if err != nil {
			return sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to get format"))
		}

		var errParse error
		switch f {
		case exportentities.FormatJSON:
			errParse = json.Unmarshal(buf, &data)
		case exportentities.FormatYAML:
			errParse = yaml.Unmarshal(buf, &data)
		}
		if errParse != nil {
			return sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrWrongRequest, "cannot parse given data"))
		}
		for i := range data {
			if err := data[i].IsValid(); err != nil {
				return err
			}
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		// Check and set group on all given group permission
		for i := range data {
			grp, err := group.LoadByName(ctx, tx, data[i].Group.Name)
			if err != nil {
				return err
			}
			data[i].Group = *grp
		}

		if forceUpdate {
			if err := group.DeleteLinksGroupProjectForProjectID(tx, proj.ID); err != nil {
				return sdk.WrapError(err, "cannot delete all groups for project %s", proj.Name)
			}
		} else {
			linksForProject, err := group.LoadLinksGroupProjectForProjectIDs(ctx, tx, []int64{proj.ID})
			if err != nil {
				return err
			}
			for i := range data {
				var exist bool
				for j := range linksForProject {
					if linksForProject[j].GroupID == data[i].Group.ID {
						exist = true
						break
					}
				}
				if exist {
					return sdk.WrapError(sdk.ErrGroupExists, "permission already set in project %s for group %s", proj.Name, data[i].Group.Name)
				}
			}
		}

		for i := range data {
			if err := group.InsertLinkGroupProject(ctx, tx, &group.LinkGroupProject{
				GroupID:   data[i].Group.ID,
				ProjectID: proj.ID,
				Role:      data[i].Permission,
			}); err != nil {
				return sdk.WrapError(err, "cannot add group %v in project %s", data[i].Group.Name, proj.Name)
			}
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "cannot commit transaction")
		}

		if err := project.LoadOptions.WithGroups(api.mustDB(), api.Cache, proj); err != nil {
			return err
		}

		return service.WriteJSON(w, proj, http.StatusOK)
	}
}
