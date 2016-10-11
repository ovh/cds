package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"

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

func deleteGroupFromProjectHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	groupName := vars["group"]

	p, err := project.LoadProject(db, key, c.User)
	if err != nil {
		log.Warning("deleteGroupFromProjectHandler: Cannot load %s: %s\n", key, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	g, err := group.LoadGroup(db, groupName)
	if err != nil {
		log.Warning("deleteGroupFromProjectHandler: Cannot find %s: %s\n", groupName, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	err = group.DeleteGroupFromProject(db, p.ID, g.ID)
	if err != nil {
		log.Warning("deleteGroupFromProjectHandler: Cannot delete group %s from project %s:  %s\n", g.Name, p.Name, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func updateGroupRoleOnProjectHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]
	groupName := vars["group"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

	var groupProject sdk.GroupPermission
	err = json.Unmarshal(data, &groupProject)
	if err != nil {
		WriteError(w, r, err)
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
	if groupInProject {

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

		err = group.UpdateGroupRoleInProject(db, p.ID, g.ID, groupProject.Permission)
		if err != nil {
			log.Warning("updateGroupRoleHandler: Cannot add group %s in project %s:  %s\n", g.Name, p.Name, err)
			WriteError(w, r, err)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

func updateGroupsInProject(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

	var groupProject []sdk.GroupPermission
	err = json.Unmarshal(data, &groupProject)
	if err != nil {
		WriteError(w, r, err)
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

func addGroupInProject(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var groupProject sdk.GroupPermission
	err = json.Unmarshal(data, &groupProject)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	p, err := project.LoadProject(db, key, c.User)
	if err != nil {
		log.Warning("AddGroupInProject: Cannot load %s: %s\n", key, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	g, err := group.LoadGroup(db, groupProject.Group.Name)
	if err != nil {
		log.Warning("AddGroupInProject: Cannot find %s: %s\n", groupProject.Group.Name, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	groupInProject, err := group.CheckGroupInProject(db, p.ID, g.ID)
	if err != nil {
		log.Warning("AddGroupInProject: Cannot check if group %s is already in the project %s: %s\n", g.Name, p.Name, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !groupInProject {

		tx, err := db.Begin()
		if err != nil {
			log.Warning("AddGroupInProject: Cannot open transaction:  %s\n", err)
			WriteError(w, r, err)
			return
		}
		defer tx.Rollback()

		err = group.InsertGroupInProject(tx, p.ID, g.ID, groupProject.Permission)
		if err != nil {
			log.Warning("AddGroupInProject: Cannot add group %s in project %s:  %s\n", g.Name, p.Name, err)
			w.WriteHeader(http.StatusInternalServerError)
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
						err = group.UpdateGroupRoleInApplication(tx, p.Key, app.Name, g.Name, groupProject.Permission)
						if err != nil {
							log.Warning("AddGroupInProject: Cannot update group %s on application %s: %s\n", g.Name, app.Name, err)
							WriteError(w, r, err)
							return
						}
					} else {
						err = group.InsertGroupInApplication(tx, app.ID, g.ID, groupProject.Permission)
						if err != nil {
							log.Warning("AddGroupInProject: Cannot insert group %s on application %s: %s\n", g.Name, app.Name, err)
							WriteError(w, r, err)
							return
						}
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
						err = group.UpdateGroupRoleInPipeline(tx, pip.ID, g.ID, groupProject.Permission)
						if err != nil {
							log.Warning("AddGroupInProject: Cannot update group %s on pipeline %s: %s\n", g.Name, pip.Name, err)
							WriteError(w, r, err)
							return
						}
					} else {
						err = group.InsertGroupInPipeline(tx, pip.ID, g.ID, groupProject.Permission)
						if err != nil {
							log.Warning("AddGroupInProject: Cannot insert group %s on pipeline %s: %s\n", g.Name, pip.Name, err)
							WriteError(w, r, err)
							return
						}
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
						err = group.UpdateGroupRoleInEnvironment(tx, p.Key, env.Name, g.Name, groupProject.Permission)
						if err != nil {
							log.Warning("AddGroupInProject: Cannot update group %s on environment %s: %s\n", g.Name, env.Name, err)
							WriteError(w, r, err)
							return
						}
					} else {
						err = group.InsertGroupInEnvironment(tx, env.ID, g.ID, groupProject.Permission)
						if err != nil {
							log.Warning("AddGroupInProject: Cannot insert group %s on environment %s: %s\n", g.Name, env.Name, err)
							WriteError(w, r, err)
							return
						}
					}
				}
			}
		}

		err = tx.Commit()
		if err != nil {
			log.Warning("AddGroupInProject: Cannot commit transaction:  %s\n", err)
			WriteError(w, r, err)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}
