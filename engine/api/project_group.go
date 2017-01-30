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

func deleteGroupFromProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {

	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	groupName := vars["group"]

	p, err := project.LoadProject(db, key, c.User)
	if err != nil {
		log.Warning("deleteGroupFromProjectHandler: Cannot load %s: %s\n", key, err)
		WriteError(w, r, err)
		return
	}

	g, err := group.LoadGroup(db, groupName)
	if err != nil {
		log.Warning("deleteGroupFromProjectHandler: Cannot find %s: %s\n", groupName, err)
		WriteError(w, r, err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteGroupFromProjectHandler: Cannot start transaction:  %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()
	if err := group.DeleteGroupFromProject(db, p.ID, g.ID); err != nil {
		log.Warning("deleteGroupFromProjectHandler: Cannot delete group %s from project %s:  %s\n", g.Name, p.Name, err)
		WriteError(w, r, err)
		return
	}
	lastModified, err := project.UpdateProjectDB(db, p.Key, p.Name)
	if err != nil {
		log.Warning("deleteGroupFromProjectHandler: Cannot update project last modified date: %s\n", err)
		WriteError(w, r, err)
		return
	}
	p.LastModified = lastModified.Unix()

	if err := tx.Commit(); err != nil {
		log.Warning("deleteGroupFromProjectHandler: Cannot commit transaction:  %s\n", err)
		WriteError(w, r, err)
		return
	}

	if err := group.LoadGroupByProject(db, p); err != nil {
		log.Warning("deleteGroupFromProjectHandler: Cannot load groups for project %s:  %s\n", p.Key, err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, p, http.StatusOK)
}

func updateGroupRoleOnProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	groupName := vars["group"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	var groupProject sdk.GroupPermission
	if err := json.Unmarshal(data, &groupProject); err != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	if groupName != groupProject.Group.Name {
		WriteError(w, r, sdk.ErrGroupNotFound)
		return
	}

	p, err := project.LoadProject(db, key, c.User)
	if err != nil {
		log.Warning("updateGroupRoleHandler: Cannot load %s: %s\n", key, err)
		WriteError(w, r, sdk.ErrNoProject)
		return
	}

	g, err := group.LoadGroup(db, groupProject.Group.Name)
	if err != nil {
		log.Warning("updateGroupRoleHandler: Cannot find %s: %s\n", groupProject.Group.Name, err)
		WriteError(w, r, sdk.ErrGroupNotFound)
		return
	}

	groupInProject, err := group.CheckGroupInProject(db, p.ID, g.ID)
	if err != nil {
		log.Warning("updateGroupRoleHandler: Cannot check if group %s is already in the project %s: %s\n", g.Name, p.Name, err)
		WriteError(w, r, err)
		return
	}
	if !groupInProject {
		log.Warning("updateGroupRoleHandler: Group is not attached to this project: %s\n", err)
		WriteError(w, r, sdk.ErrGroupNotFound)
		return
	}

	if groupProject.Permission != permission.PermissionReadWriteExecute {
		permissions, err := group.LoadAllProjectGroupByRole(db, p.ID, permission.PermissionReadWriteExecute)
		if err != nil {
			log.Warning("updateGroupRoleHandler: Cannot load group for the given project %s:  %s\n", p.Name, err)
			WriteError(w, r, err)
			return
		}
		// If the updated group is the only one in write mode, return error
		if len(permissions) == 1 && permissions[0].Group.ID == g.ID {
			log.Warning("updateGroupRoleHandler: Cannot remove write permission for this group %s on this project %s\n", g.Name, p.Name)
			WriteError(w, r, sdk.ErrGroupNeedWrite)
			return
		}
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateGroupRoleHandler: Cannot start transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	if err := group.UpdateGroupRoleInProject(db, p.ID, g.ID, groupProject.Permission); err != nil {
		log.Warning("updateGroupRoleHandler: Cannot add group %s in project %s:  %s\n", g.Name, p.Name, err)
		WriteError(w, r, err)
		return
	}

	lastModified, err := project.UpdateProjectDB(db, p.Key, p.Name)
	if err != nil {
		log.Warning("updateGroupRoleHandler: Cannot update project last modified date: %s\n", err)
		WriteError(w, r, err)
		return
	}
	p.LastModified = lastModified.Unix()

	if err := tx.Commit(); err != nil {
		log.Warning("updateGroupRoleHandler: Cannot start transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	if err := group.LoadGroupByProject(db, p); err != nil {
		log.Warning("updateGroupRoleHandler: Cannot load group for project %s: %s\n", p.Key, err)
		WriteError(w, r, err)
		return
	}
	WriteJSON(w, r, p, http.StatusOK)
}

// Deprecated
func updateGroupsInProject(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	var groupProject []sdk.GroupPermission
	err = json.Unmarshal(data, &groupProject)
	if err != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	if len(groupProject) == 0 {
		log.Warning("updateGroupsInProject: Cannot remove all groups.")
		WriteError(w, r, sdk.ErrGroupNeedWrite)
		return
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
		WriteError(w, r, sdk.ErrGroupNeedWrite)
		return
	}

	p, err := project.LoadProject(db, key, c.User)
	if err != nil {
		log.Warning("updateGroupsInProject: Cannot load %s: %s\n", key, err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateGroupsInProject: Cannot start transaction: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}
	defer tx.Rollback()

	err = group.DeleteGroupProjectByProject(tx, p.ID)
	if err != nil {
		log.Warning("updateGroupsInProject: Cannot delete groups from project %s: %s\n", p.Name, err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

	for _, g := range groupProject {
		groupData, err := group.LoadGroup(tx, g.Group.Name)
		if err != nil {
			log.Warning("updateGroupsInProject: Cannot load group %s : %s\n", g.Group.Name, err)
			WriteError(w, r, sdk.ErrUnknownError)
			return
		}

		err = group.InsertGroupInProject(tx, p.ID, groupData.ID, g.Permission)
		if err != nil {
			log.Warning("updateGroupsInProject: Cannot add group %s in project %s: %s\n", g.Group.Name, p.Name, err)
			WriteError(w, r, sdk.ErrUnknownError)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("updateGroupsInProject: Cannot commit transaction: %s", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}
	w.WriteHeader(http.StatusOK)

}

func addGroupInProject(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {

	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	var groupProject sdk.GroupPermission
	if err := json.Unmarshal(data, &groupProject); err != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	p, err := project.LoadProject(db, key, c.User)
	if err != nil {
		log.Warning("AddGroupInProject: Cannot load %s: %s\n", key, err)
		WriteError(w, r, err)
		return
	}

	g, err := group.LoadGroup(db, groupProject.Group.Name)
	if err != nil {
		log.Warning("AddGroupInProject: Cannot find %s: %s\n", groupProject.Group.Name, err)
		WriteError(w, r, err)
		return
	}

	groupInProject, err := group.CheckGroupInProject(db, p.ID, g.ID)
	if err != nil {
		log.Warning("AddGroupInProject: Cannot check if group %s is already in the project %s: %s\n", g.Name, p.Name, err)
		WriteError(w, r, err)
		return
	}
	if groupInProject {
		log.Warning("AddGroupInProject: Group already in the project: %s\n", p.Name, err)
		WriteError(w, r, sdk.ErrGroupExists)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("AddGroupInProject: Cannot open transaction:  %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	if err := group.InsertGroupInProject(tx, p.ID, g.ID, groupProject.Permission); err != nil {
		log.Warning("AddGroupInProject: Cannot add group %s in project %s:  %s\n", g.Name, p.Name, err)
		WriteError(w, r, err)
		return
	}

	// recursive or not?
	if groupProject.Recursive {

		// apply on application
		applications, err := application.LoadApplications(tx, p.Key, false, c.User)
		if err != nil {
			log.Warning("AddGroupInProject: Cannot load applications for project %s:  %s\n", p.Name, err)
			WriteError(w, r, err)
			return
		}

		for _, app := range applications {
			if permission.AccessToApplication(app.ID, c.User, permission.PermissionReadWriteExecute) {
				inApp, err := group.CheckGroupInApplication(tx, app.ID, g.ID)
				if err != nil {
					log.Warning("AddGroupInProject: Cannot check if group %s is already in the application %s: %s\n", g.Name, app.Name, err)
					WriteError(w, r, err)
					return
				}
				if inApp {
					if err := group.UpdateGroupRoleInApplication(tx, p.Key, app.Name, g.Name, groupProject.Permission); err != nil {
						log.Warning("AddGroupInProject: Cannot update group %s on application %s: %s\n", g.Name, app.Name, err)
						WriteError(w, r, err)
						return
					}
				} else if err := group.InsertGroupInApplication(tx, app.ID, g.ID, groupProject.Permission); err != nil {
					log.Warning("AddGroupInProject: Cannot insert group %s on application %s: %s\n", g.Name, app.Name, err)
					WriteError(w, r, err)
					return
				}
			}
		}

		// apply on pipeline
		pipelines, err := pipeline.LoadPipelines(tx, p.ID, false, c.User)
		if err != nil {
			log.Warning("AddGroupInProject: Cannot load pipelines for project %s:  %s\n", p.Name, err)
			WriteError(w, r, err)
			return
		}

		for _, pip := range pipelines {
			if permission.AccessToPipeline(sdk.DefaultEnv.ID, pip.ID, c.User, permission.PermissionReadWriteExecute) {
				inPip, err := group.CheckGroupInPipeline(tx, pip.ID, g.ID)
				if err != nil {
					log.Warning("AddGroupInProject: Cannot check if group %s is already in the pipeline %s: %s\n", g.Name, pip.Name, err)
					WriteError(w, r, err)
					return
				}
				if inPip {
					if err := group.UpdateGroupRoleInPipeline(tx, pip.ID, g.ID, groupProject.Permission); err != nil {
						log.Warning("AddGroupInProject: Cannot update group %s on pipeline %s: %s\n", g.Name, pip.Name, err)
						WriteError(w, r, err)
						return
					}
				} else if err := group.InsertGroupInPipeline(tx, pip.ID, g.ID, groupProject.Permission); err != nil {
					log.Warning("AddGroupInProject: Cannot insert group %s on pipeline %s: %s\n", g.Name, pip.Name, err)
					WriteError(w, r, err)
					return
				}
			}
		}

		// apply on environment
		envs, err := environment.LoadEnvironments(tx, p.Key, false, c.User)
		if err != nil {
			log.Warning("AddGroupInProject: Cannot load environments for project %s:  %s\n", p.Name, err)
			WriteError(w, r, err)
			return
		}

		for _, env := range envs {
			if permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadWriteExecute) {
				inEnv, err := group.IsInEnvironment(tx, env.ID, g.ID)
				if err != nil {
					log.Warning("AddGroupInProject: Cannot check if group %s is already in the environment %s: %s\n", g.Name, env.Name, err)
					WriteError(w, r, err)
					return
				}
				if inEnv {
					if err := group.UpdateGroupRoleInEnvironment(tx, p.Key, env.Name, g.Name, groupProject.Permission); err != nil {
						log.Warning("AddGroupInProject: Cannot update group %s on environment %s: %s\n", g.Name, env.Name, err)
						WriteError(w, r, err)
						return
					}
				} else if err := group.InsertGroupInEnvironment(tx, env.ID, g.ID, groupProject.Permission); err != nil {
					log.Warning("AddGroupInProject: Cannot insert group %s on environment %s: %s\n", g.Name, env.Name, err)
					WriteError(w, r, err)
					return
				}
			}
		}
	}

	lastModified, err := project.UpdateProjectDB(db, p.Key, p.Name)
	if err != nil {
		log.Warning("AddGroupInProject: Cannot update project last modified date: %s\n", err)
		WriteError(w, r, err)
		return
	}
	p.LastModified = lastModified.Unix()

	if err := tx.Commit(); err != nil {
		log.Warning("AddGroupInProject: Cannot commit transaction:  %s\n", err)
		WriteError(w, r, err)
		return
	}

	if err := group.LoadGroupByProject(db, p); err != nil {
		log.Warning("AddGroupInProject: Cannot load groups on project %s:  %s\n", p.Key, err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, p, http.StatusOK)
}
