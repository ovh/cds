package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func deleteGroupFromProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {

	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	groupName := vars["group"]

	p, err := project.Load(db, key, c.User)
	if err != nil {
		log.Warning("deleteGroupFromProjectHandler: Cannot load %s: %s\n", key, err)
		return err

	}

	g, err := group.LoadGroup(db, groupName)
	if err != nil {
		log.Warning("deleteGroupFromProjectHandler: Cannot find %s: %s\n", groupName, err)
		return err

	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteGroupFromProjectHandler: Cannot start transaction:  %s\n", err)
		return err

	}
	defer tx.Rollback()
	if err := group.DeleteGroupFromProject(db, p.ID, g.ID); err != nil {
		log.Warning("deleteGroupFromProjectHandler: Cannot delete group %s from project %s:  %s\n", g.Name, p.Name, err)
		return err

	}
	lastModified, err := project.UpdateProjectDB(db, p.Key, p.Name)
	if err != nil {
		log.Warning("deleteGroupFromProjectHandler: Cannot update project last modified date: %s\n", err)
		return err

	}
	p.LastModified = lastModified

	if err := tx.Commit(); err != nil {
		log.Warning("deleteGroupFromProjectHandler: Cannot commit transaction:  %s\n", err)
		return err

	}

	if err := group.LoadGroupByProject(db, p); err != nil {
		log.Warning("deleteGroupFromProjectHandler: Cannot load groups for project %s:  %s\n", p.Key, err)
		return err

	}

	return WriteJSON(w, r, p, http.StatusOK)
}

func updateGroupRoleOnProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	groupName := vars["group"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return sdk.ErrWrongRequest

	}

	var groupProject sdk.GroupPermission
	if err := json.Unmarshal(data, &groupProject); err != nil {
		return sdk.ErrWrongRequest

	}

	if groupName != groupProject.Group.Name {
		return sdk.ErrGroupNotFound

	}

	p, err := project.Load(db, key, c.User)
	if err != nil {
		log.Warning("updateGroupRoleHandler: Cannot load %s: %s\n", key, err)
		return sdk.ErrNoProject

	}

	g, err := group.LoadGroup(db, groupProject.Group.Name)
	if err != nil {
		log.Warning("updateGroupRoleHandler: Cannot find %s: %s\n", groupProject.Group.Name, err)
		return sdk.ErrGroupNotFound

	}

	groupInProject, err := group.CheckGroupInProject(db, p.ID, g.ID)
	if err != nil {
		log.Warning("updateGroupRoleHandler: Cannot check if group %s is already in the project %s: %s\n", g.Name, p.Name, err)
		return err

	}
	if !groupInProject {
		log.Warning("updateGroupRoleHandler: Group is not attached to this project: %s\n", err)
		return sdk.ErrGroupNotFound

	}

	if groupProject.Permission != permission.PermissionReadWriteExecute {
		permissions, err := group.LoadAllProjectGroupByRole(db, p.ID, permission.PermissionReadWriteExecute)
		if err != nil {
			log.Warning("updateGroupRoleHandler: Cannot load group for the given project %s:  %s\n", p.Name, err)
			return err

		}
		// If the updated group is the only one in write mode, return error
		if len(permissions) == 1 && permissions[0].Group.ID == g.ID {
			log.Warning("updateGroupRoleHandler: Cannot remove write permission for this group %s on this project %s\n", g.Name, p.Name)
			return sdk.ErrGroupNeedWrite

		}
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateGroupRoleHandler: Cannot start transaction: %s\n", err)
		return err

	}
	defer tx.Rollback()

	if err := group.UpdateGroupRoleInProject(db, p.ID, g.ID, groupProject.Permission); err != nil {
		log.Warning("updateGroupRoleHandler: Cannot add group %s in project %s:  %s\n", g.Name, p.Name, err)
		return err

	}

	lastModified, err := project.UpdateProjectDB(db, p.Key, p.Name)
	if err != nil {
		log.Warning("updateGroupRoleHandler: Cannot update project last modified date: %s\n", err)
		return err

	}
	p.LastModified = lastModified

	if err := tx.Commit(); err != nil {
		log.Warning("updateGroupRoleHandler: Cannot start transaction: %s\n", err)
		return err

	}

	if err := group.LoadGroupByProject(db, p); err != nil {
		log.Warning("updateGroupRoleHandler: Cannot load group for project %s: %s\n", p.Key, err)
		return err

	}
	return WriteJSON(w, r, p, http.StatusOK)
}

// Deprecated
func updateGroupsInProject(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return sdk.ErrWrongRequest

	}

	var groupProject []sdk.GroupPermission
	err = json.Unmarshal(data, &groupProject)
	if err != nil {
		return sdk.ErrWrongRequest

	}

	if len(groupProject) == 0 {
		log.Warning("updateGroupsInProject: Cannot remove all groups.")
		return sdk.ErrGroupNeedWrite

	}

	found := false
	for _, gp := range groupProject {
		if gp.Permission == permission.PermissionReadWriteExecute {
			found = true
			break
		}
	}
	if !found {
		log.Warning("updateGroupsInProject: Need one group with write permission.")
		return sdk.ErrGroupNeedWrite

	}

	p, err := project.Load(db, key, c.User)
	if err != nil {
		log.Warning("updateGroupsInProject: Cannot load %s: %s\n", key, err)
		return sdk.ErrUnknownError

	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateGroupsInProject: Cannot start transaction: %s\n", err)
		return sdk.ErrUnknownError

	}
	defer tx.Rollback()

	err = group.DeleteGroupProjectByProject(tx, p.ID)
	if err != nil {
		log.Warning("updateGroupsInProject: Cannot delete groups from project %s: %s\n", p.Name, err)
		return sdk.ErrUnknownError

	}

	for _, g := range groupProject {
		groupData, err := group.LoadGroup(tx, g.Group.Name)
		if err != nil {
			log.Warning("updateGroupsInProject: Cannot load group %s : %s\n", g.Group.Name, err)
			return sdk.ErrUnknownError

		}

		err = group.InsertGroupInProject(tx, p.ID, groupData.ID, g.Permission)
		if err != nil {
			log.Warning("updateGroupsInProject: Cannot add group %s in project %s: %s\n", g.Group.Name, p.Name, err)
			return sdk.ErrUnknownError

		}
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("updateGroupsInProject: Cannot commit transaction: %s", err)
		return sdk.ErrUnknownError

	}
	return nil

}

func addGroupInProject(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {

	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return sdk.ErrWrongRequest

	}

	var groupProject sdk.GroupPermission
	if err := json.Unmarshal(data, &groupProject); err != nil {
		return sdk.ErrWrongRequest

	}

	p, err := project.Load(db, key, c.User)
	if err != nil {
		log.Warning("AddGroupInProject: Cannot load %s: %s\n", key, err)
		return err

	}

	g, err := group.LoadGroup(db, groupProject.Group.Name)
	if err != nil {
		log.Warning("AddGroupInProject: Cannot find %s: %s\n", groupProject.Group.Name, err)
		return err

	}

	groupInProject, err := group.CheckGroupInProject(db, p.ID, g.ID)
	if err != nil {
		log.Warning("AddGroupInProject: Cannot check if group %s is already in the project %s: %s\n", g.Name, p.Name, err)
		return err

	}
	if groupInProject {
		log.Warning("AddGroupInProject: Group already in the project: %s\n", p.Name, err)
		return sdk.ErrGroupExists

	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("AddGroupInProject: Cannot open transaction:  %s\n", err)
		return err

	}
	defer tx.Rollback()

	if err := group.InsertGroupInProject(tx, p.ID, g.ID, groupProject.Permission); err != nil {
		log.Warning("AddGroupInProject: Cannot add group %s in project %s:  %s\n", g.Name, p.Name, err)
		return err

	}

	// recursive or not?
	if groupProject.Recursive {

		// apply on application
		applications, err := application.LoadApplications(tx, p.Key, false, false, c.User)
		if err != nil {
			log.Warning("AddGroupInProject: Cannot load applications for project %s:  %s\n", p.Name, err)
			return err

		}

		for _, app := range applications {
			if permission.AccessToApplication(app.ID, c.User, permission.PermissionReadWriteExecute) {
				inApp, err := group.CheckGroupInApplication(tx, app.ID, g.ID)
				if err != nil {
					log.Warning("AddGroupInProject: Cannot check if group %s is already in the application %s: %s\n", g.Name, app.Name, err)
					return err

				}
				if inApp {
					if err := group.UpdateGroupRoleInApplication(tx, p.Key, app.Name, g.Name, groupProject.Permission); err != nil {
						log.Warning("AddGroupInProject: Cannot update group %s on application %s: %s\n", g.Name, app.Name, err)
						return err

					}
				} else if err := group.InsertGroupInApplication(tx, app.ID, g.ID, groupProject.Permission); err != nil {
					log.Warning("AddGroupInProject: Cannot insert group %s on application %s: %s\n", g.Name, app.Name, err)
					return err

				}
			}
		}

		// apply on pipeline
		pipelines, err := pipeline.LoadPipelines(tx, p.ID, false, c.User)
		if err != nil {
			log.Warning("AddGroupInProject: Cannot load pipelines for project %s:  %s\n", p.Name, err)
			return err

		}

		for _, pip := range pipelines {
			if permission.AccessToPipeline(sdk.DefaultEnv.ID, pip.ID, c.User, permission.PermissionReadWriteExecute) {
				inPip, err := group.CheckGroupInPipeline(tx, pip.ID, g.ID)
				if err != nil {
					log.Warning("AddGroupInProject: Cannot check if group %s is already in the pipeline %s: %s\n", g.Name, pip.Name, err)
					return err

				}
				if inPip {
					if err := group.UpdateGroupRoleInPipeline(tx, pip.ID, g.ID, groupProject.Permission); err != nil {
						log.Warning("AddGroupInProject: Cannot update group %s on pipeline %s: %s\n", g.Name, pip.Name, err)
						return err

					}
				} else if err := group.InsertGroupInPipeline(tx, pip.ID, g.ID, groupProject.Permission); err != nil {
					log.Warning("AddGroupInProject: Cannot insert group %s on pipeline %s: %s\n", g.Name, pip.Name, err)
					return err

				}
			}
		}

		// apply on environment
		envs, err := environment.LoadEnvironments(tx, p.Key, false, c.User)
		if err != nil {
			log.Warning("AddGroupInProject: Cannot load environments for project %s:  %s\n", p.Name, err)
			return err

		}

		for _, env := range envs {
			if permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadWriteExecute) {
				inEnv, err := group.IsInEnvironment(tx, env.ID, g.ID)
				if err != nil {
					log.Warning("AddGroupInProject: Cannot check if group %s is already in the environment %s: %s\n", g.Name, env.Name, err)
					return err

				}
				if inEnv {
					if err := group.UpdateGroupRoleInEnvironment(tx, p.Key, env.Name, g.Name, groupProject.Permission); err != nil {
						log.Warning("AddGroupInProject: Cannot update group %s on environment %s: %s\n", g.Name, env.Name, err)
						return err

					}
				} else if err := group.InsertGroupInEnvironment(tx, env.ID, g.ID, groupProject.Permission); err != nil {
					log.Warning("AddGroupInProject: Cannot insert group %s on environment %s: %s\n", g.Name, env.Name, err)
					return err

				}
			}
		}
	}

	lastModified, err := project.UpdateProjectDB(db, p.Key, p.Name)
	if err != nil {
		log.Warning("AddGroupInProject: Cannot update project last modified date: %s\n", err)
		return err

	}
	p.LastModified = lastModified

	if err := tx.Commit(); err != nil {
		log.Warning("AddGroupInProject: Cannot commit transaction:  %s\n", err)
		return err

	}

	if err := group.LoadGroupByProject(db, p); err != nil {
		log.Warning("AddGroupInProject: Cannot load groups on project %s:  %s\n", p.Key, err)
		return err

	}

	return WriteJSON(w, r, p, http.StatusOK)
}
