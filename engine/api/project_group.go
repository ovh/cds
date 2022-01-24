package api

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
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

		proj, err := project.Load(ctx, tx, key)
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
			return sdk.WithStack(err)
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
		onlyProject := service.FormBool(r, "onlyProject")

		var data sdk.GroupPermission
		if err := service.UnmarshalBody(r, &data); err != nil {
			return err
		}
		if err := data.IsValid(); err != nil {
			return err
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "cannot start transaction")
		}
		defer tx.Rollback() // nolint

		proj, err := project.Load(ctx, tx, key)
		if err != nil {
			return sdk.WrapError(err, "cannot load project %s", key)
		}

		grp, err := group.LoadByName(ctx, tx, groupName, group.LoadOptions.WithOrganization, group.LoadOptions.WithMembers)
		if err != nil {
			return sdk.WrapError(err, "cannot find %s", groupName)
		}

		if group.IsDefaultGroupID(grp.ID) && data.Permission > sdk.PermissionRead {
			return sdk.NewErrorFrom(sdk.ErrDefaultGroupPermission, "only read permission is allowed to default group")
		}

		if err := projectPermissionCheckOrganizationMatch(ctx, tx, proj, grp, data.Permission); err != nil {
			return err
		}

		oldLink, err := group.LoadLinkGroupProjectForGroupIDAndProjectID(ctx, tx, grp.ID, proj.ID)
		if err != nil {
			return err
		}
		if !isGroupAdmin(ctx, grp) && data.Permission > oldLink.Role {
			if isAdmin(ctx) {
				trackSudo(ctx, w)
			} else {
				return sdk.WithStack(sdk.ErrInvalidGroupAdmin)
			}
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
			return sdk.WrapError(err, "cannot start transaction")
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
		onlyProject := service.FormBool(r, "onlyProject")

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

		proj, err := project.Load(ctx, tx, key)
		if err != nil {
			return sdk.WrapError(err, "cannot load %s", key)
		}

		grp, err := group.LoadByName(ctx, tx, data.Group.Name, group.LoadOptions.WithOrganization, group.LoadOptions.WithMembers)
		if err != nil {
			return sdk.WrapError(err, "cannot find %s", data.Group.Name)
		}

		if !isGroupAdmin(ctx, grp) {
			if isAdmin(ctx) {
				trackSudo(ctx, w)
			} else {
				return sdk.WithStack(sdk.ErrInvalidGroupAdmin)
			}
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

		if err := projectPermissionCheckOrganizationMatch(ctx, tx, proj, grp, data.Permission); err != nil {
			return err
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
			return sdk.WithStack(err)
		}

		newGroupPermission := sdk.GroupPermission{Permission: newLink.Role, Group: *grp}
		event.PublishAddProjectPermission(ctx, proj, newGroupPermission, getAPIConsumer(ctx))

		return service.WriteJSON(w, newGroupPermission, http.StatusOK)
	}
}

func projectPermissionCheckOrganizationMatch(ctx context.Context, db gorp.SqlExecutor, proj *sdk.Project, grp *sdk.Group, role int) error {
	if role > sdk.PermissionRead {
		if len(proj.ProjectGroups) == 0 {
			if err := group.LoadGroupsIntoProject(ctx, db, proj); err != nil {
				return err
			}
		}
		projectOrganization, err := proj.ProjectGroups.ComputeOrganization()
		if err != nil {
			return sdk.NewError(sdk.ErrForbidden, err)
		}
		if projectOrganization == "" {
			return nil
		}
		if grp.Organization != projectOrganization {
			if grp.Organization == "" {
				return sdk.NewErrorFrom(sdk.ErrForbidden, "given group without organization don't match project organization %q", projectOrganization)
			}
			return sdk.NewErrorFrom(sdk.ErrForbidden, "given group with organization %q don't match project organization %q", grp.Organization, projectOrganization)
		}
	}
	return nil
}
