package main

import (
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/sdk"
)

func deleteGroupFromProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	groupName := vars["group"]

	p, err := project.Load(db, key, c.User)
	if err != nil {
		return sdk.WrapError(err, "deleteGroupFromProjectHandler: Cannot load %s", key)
	}

	g, err := group.LoadGroup(db, groupName)
	if err != nil {
		return sdk.WrapError(err, "deleteGroupFromProjectHandler: Cannot find %s", groupName)
	}

	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "deleteGroupFromProjectHandler: Cannot start transaction")
	}

	defer tx.Rollback()
	if err := group.DeleteGroupFromProject(db, p.ID, g.ID); err != nil {
		return sdk.WrapError(err, "deleteGroupFromProjectHandler: Cannot delete group %s from project %s", g.Name, p.Name)
	}

	if err := project.UpdateLastModified(tx, c.User, p); err != nil {
		return sdk.WrapError(err, "deleteGroupFromProjectHandler: Cannot update last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "deleteGroupFromProjectHandler: Cannot commit transaction")
	}

	return WriteJSON(w, r, nil, http.StatusOK)
}

func updateGroupRoleOnProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	groupName := vars["group"]

	var groupProject sdk.GroupPermission
	if err := UnmarshalBody(r, &groupProject); err != nil {
		return sdk.WrapError(err, "updateGroupRoleOnProjectHandler> unable to unmarshal")
	}

	if groupName != groupProject.Group.Name {
		return sdk.ErrGroupNotFound
	}

	p, errl := project.Load(db, key, c.User)
	if errl != nil {
		return sdk.WrapError(errl, "updateGroupRoleHandler: Cannot load %s: %s", key)
	}

	g, errlg := group.LoadGroup(db, groupProject.Group.Name)
	if errlg != nil {
		return sdk.WrapError(errlg, "updateGroupRoleHandler: Cannot find %s", groupProject.Group.Name)
	}

	groupInProject, errcg := group.CheckGroupInProject(db, p.ID, g.ID)
	if errcg != nil {
		return sdk.WrapError(errcg, "updateGroupRoleHandler: Cannot check if group %s is already in the project %s", g.Name, p.Name)
	}

	if !groupInProject {
		return sdk.WrapError(sdk.ErrGroupNotFound, "updateGroupRoleHandler: Group is not attached to this project: %s")
	}

	if groupProject.Permission != permission.PermissionReadWriteExecute {
		permissions, err := group.LoadAllProjectGroupByRole(db, p.ID, permission.PermissionReadWriteExecute)
		if err != nil {
			return sdk.WrapError(err, "updateGroupRoleHandler: Cannot load group for the given project %s", p.Name)
		}
		// If the updated group is the only one in write mode, return error
		if len(permissions) == 1 && permissions[0].Group.ID == g.ID {
			return sdk.WrapError(sdk.ErrGroupNeedWrite, "updateGroupRoleHandler: Cannot remove write permission for this group %s on this project %s", g.Name, p.Name)
		}
	}

	tx, errb := db.Begin()
	if errb != nil {
		return sdk.WrapError(errb, "updateGroupRoleHandler: Cannot start transaction")
	}
	defer tx.Rollback()

	if err := group.UpdateGroupRoleInProject(db, p.ID, g.ID, groupProject.Permission); err != nil {
		return sdk.WrapError(err, "updateGroupRoleHandler: Cannot add group %s in project %s", g.Name, p.Name)
	}

	if err := project.UpdateLastModified(tx, c.User, p); err != nil {
		return sdk.WrapError(err, "updateGroupRoleHandler: Cannot update last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "updateGroupRoleHandler: Cannot start transaction: %s")
	}
	return WriteJSON(w, r, groupProject, http.StatusOK)
}

// Deprecated
func updateGroupsInProject(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	var groupProject []sdk.GroupPermission
	if err := UnmarshalBody(r, &groupProject); err != nil {
		return sdk.WrapError(err, "updateGroupsInProject> unable to unmarshal")
	}

	if len(groupProject) == 0 {
		return sdk.WrapError(sdk.ErrGroupNeedWrite, "updateGroupsInProject: Cannot remove all groups.")
	}

	found := false
	for _, gp := range groupProject {
		if gp.Permission == permission.PermissionReadWriteExecute {
			found = true
			break
		}
	}
	if !found {
		return sdk.WrapError(sdk.ErrGroupNeedWrite, "updateGroupsInProject: Need one group with write permission.")
	}

	p, err := project.Load(db, key, c.User)
	if err != nil {
		return sdk.WrapError(err, "updateGroupsInProject: Cannot load %s")
	}

	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "updateGroupsInProject: Cannot start transaction")
	}
	defer tx.Rollback()

	err = group.DeleteGroupProjectByProject(tx, p.ID)
	if err != nil {
		return sdk.WrapError(err, "updateGroupsInProject: Cannot delete groups from project %s", p.Name)
	}

	for _, g := range groupProject {
		groupData, errl := group.LoadGroup(tx, g.Group.Name)
		if errl != nil {
			return sdk.WrapError(errl, "updateGroupsInProject: Cannot load group %s", g.Group.Name)
		}

		if err := group.InsertGroupInProject(tx, p.ID, groupData.ID, g.Permission); err != nil {
			return sdk.WrapError(err, "updateGroupsInProject: Cannot add group %s in project %s", g.Group.Name, p.Name)
		}
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "updateGroupsInProject: Cannot commit transaction")
	}
	return nil
}

func addGroupInProject(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	var groupProject sdk.GroupPermission
	if err := UnmarshalBody(r, &groupProject); err != nil {
		return sdk.WrapError(err, "addGroupInProject> unable to unmarshal")
	}

	p, errl := project.Load(db, key, c.User)
	if errl != nil {
		return sdk.WrapError(errl, "AddGroupInProject: Cannot load %s", key)
	}

	g, errlg := group.LoadGroup(db, groupProject.Group.Name)
	if errlg != nil {
		return sdk.WrapError(errlg, "AddGroupInProject: Cannot find %s", groupProject.Group.Name)
	}

	groupInProject, errc := group.CheckGroupInProject(db, p.ID, g.ID)
	if errc != nil {
		return sdk.WrapError(errc, "AddGroupInProject: Cannot check if group %s is already in the project %s", g.Name, p.Name)
	}
	if groupInProject {
		return sdk.WrapError(sdk.ErrGroupExists, "AddGroupInProject: Group already in the project %s", p.Name)
	}

	tx, errb := db.Begin()
	if errb != nil {
		return sdk.WrapError(errb, "AddGroupInProject: Cannot open transaction")
	}
	defer tx.Rollback()

	if err := group.InsertGroupInProject(tx, p.ID, g.ID, groupProject.Permission); err != nil {
		return sdk.WrapError(err, "AddGroupInProject: Cannot add group %s in project %s", g.Name, p.Name)
	}

	// apply on application
	applications, errla := application.LoadAll(tx, p.Key, c.User)
	if errla != nil {
		return sdk.WrapError(errla, "AddGroupInProject: Cannot load applications for project %s", p.Name)
	}

	for _, app := range applications {
		if permission.AccessToApplication(app.ID, c.User, permission.PermissionReadWriteExecute) {
			inApp, err := group.CheckGroupInApplication(tx, app.ID, g.ID)
			if err != nil {
				return sdk.WrapError(err, "AddGroupInProject: Cannot check if group %s is already in the application %s", g.Name, app.Name)
			}
			if inApp {
				if err := group.UpdateGroupRoleInApplication(tx, p.Key, app.Name, g.Name, groupProject.Permission); err != nil {
					return sdk.WrapError(err, "AddGroupInProject: Cannot update group %s on application %s", g.Name, app.Name)
				}
			} else if err := application.AddGroup(tx, p, &app, c.User, groupProject); err != nil {
				return sdk.WrapError(err, "AddGroupInProject: Cannot insert group %s on application %s", g.Name, app.Name)
			}
		}
	}

	// apply on pipeline
	pipelines, errlp := pipeline.LoadPipelines(tx, p.ID, false, c.User)
	if errlp != nil {
		return sdk.WrapError(errlp, "AddGroupInProject: Cannot load pipelines for project %s", p.Name)
	}

	for _, pip := range pipelines {
		if permission.AccessToPipeline(sdk.DefaultEnv.ID, pip.ID, c.User, permission.PermissionReadWriteExecute) {
			inPip, err := group.CheckGroupInPipeline(tx, pip.ID, g.ID)
			if err != nil {
				return sdk.WrapError(err, "AddGroupInProject: Cannot check if group %s is already in the pipeline %s", g.Name, pip.Name)
			}
			if inPip {
				if err := group.UpdateGroupRoleInPipeline(tx, pip.ID, g.ID, groupProject.Permission); err != nil {
					return sdk.WrapError(err, "AddGroupInProject: Cannot update group %s on pipeline %s", g.Name, pip.Name)
				}
			} else if err := group.InsertGroupInPipeline(tx, pip.ID, g.ID, groupProject.Permission); err != nil {
				return sdk.WrapError(err, "AddGroupInProject: Cannot insert group %s on pipeline %s", g.Name, pip.Name)
			}
		}
	}

	// apply on environment
	envs, errle := environment.LoadEnvironments(tx, p.Key, false, c.User)
	if errle != nil {
		return sdk.WrapError(errle, "AddGroupInProject: Cannot load environments for project %s", p.Name)
	}

	for _, env := range envs {
		if permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadWriteExecute) {
			inEnv, err := group.IsInEnvironment(tx, env.ID, g.ID)
			if err != nil {
				return sdk.WrapError(err, "AddGroupInProject: Cannot check if group %s is already in the environment %s", g.Name, env.Name)
			}
			if inEnv {
				if err := group.UpdateGroupRoleInEnvironment(tx, p.Key, env.Name, g.Name, groupProject.Permission); err != nil {
					return sdk.WrapError(err, "AddGroupInProject: Cannot update group %s on environment %s", g.Name, env.Name)
				}
			} else if err := group.InsertGroupInEnvironment(tx, env.ID, g.ID, groupProject.Permission); err != nil {
				return sdk.WrapError(err, "AddGroupInProject: Cannot insert group %s on environment %s", g.Name, env.Name)
			}
		}
	}

	if err := project.UpdateLastModified(tx, c.User, p); err != nil {
		return sdk.WrapError(err, "AddGroupInProject: Cannot update last modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "AddGroupInProject: Cannot commit transaction")
	}

	if err := group.LoadGroupByProject(db, p); err != nil {
		return sdk.WrapError(err, "AddGroupInProject: Cannot load groups on project %s", p.Key)
	}

	return WriteJSON(w, r, p.ProjectGroups, http.StatusOK)
}
