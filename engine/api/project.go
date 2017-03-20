package main

import (
	"database/sql"
	"net/http"
	"regexp"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func getProjectsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	withApplication := FormBool(r, "application")

	var projects []sdk.Project
	var err error

	if withApplication {
		projects, err = project.LoadAll(db, c.User, project.LoadOptions.WithApplications)
	} else {
		projects, err = project.LoadAll(db, c.User)
	}
	if err != nil {
		return sdk.WrapError(err, "getProjectsHandler")
	}
	return WriteJSON(w, r, projects, http.StatusOK)
}

func updateProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	proj := &sdk.Project{}
	if err := UnmarshalBody(r, proj); err != nil {
		return err
	}

	if proj.Name == "" {
		log.Warning("updateProject: Project name must no be empty")
		return sdk.ErrInvalidProjectName
	}

	// Check Request
	if key != proj.Key {
		log.Warning("updateProject: bad Project key %s/%s \n", key, proj.Key)
		return sdk.ErrWrongRequest
	}

	// Check is project exist
	p, errProj := project.Load(db, key, c.User)
	if errProj != nil {
		log.Warning("updateProject: Cannot load project from db: %s\n", errProj)
		return errProj
	}
	// Update in DB is made given the primary key
	proj.ID = p.ID
	if errUp := project.Update(db, proj); errUp != nil {
		log.Warning("updateProject: Cannot update project %s : %s\n", key, errUp)
		return errUp
	}

	return WriteJSON(w, r, p, http.StatusOK)
}

func getProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	p, errProj := project.Load(db, key, c.User,
		project.LoadOptions.WithVariables,
		project.LoadOptions.WithApplications,
		project.LoadOptions.WithApplicationPipelines,
		project.LoadOptions.WithEnvironments,
		project.LoadOptions.WithGroups,
		project.LoadOptions.WithPermission,
		project.LoadOptions.WithPipelines,
		project.LoadOptions.WithRepositoriesManagers,
	)
	if errProj != nil {
		return sdk.WrapError(errProj, "getProjectHandler (%s)", key)
	}

	return WriteJSON(w, r, p, http.StatusOK)
}

func addProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	//Unmarshal data
	p := &sdk.Project{}
	if err := UnmarshalBody(r, p); err != nil {
		return err
	}

	// check projectKey pattern
	if rgxp := regexp.MustCompile(sdk.ProjectKeyPattern); !rgxp.MatchString(p.Key) {
		log.Warning("AddProject: Project key %s do not respect pattern %s", p.Key, sdk.ProjectKeyPattern)
		return sdk.ErrInvalidProjectKey
	}

	//check project Name
	if p.Name == "" {
		log.Warning("AddProject: Project name must no be empty")
		return sdk.ErrInvalidProjectName

	}

	// Check that project does not already exists
	exist, errExist := project.Exist(db, p.Key)
	if errExist != nil {
		log.Warning("AddProject: Cannot check if project %s exist: %s\n", p.Key, errExist)
		return errExist
	}

	if exist {
		log.Warning("AddProject: Project %s already exists\n", p.Key)
		// Write nice error message here
		return sdk.ErrConflict

	}

	//Create a project within a transaction
	tx, errBegin := db.Begin()
	defer tx.Rollback()
	if errBegin != nil {
		log.Warning("AddProject: Cannot start transaction: %s\n", errBegin)
		return errBegin

	}

	if err := project.Insert(tx, p); err != nil {
		log.Warning("AddProject: Cannot insert project: %s\n", err)
		return err

	}

	// Add group
	for i := range p.ProjectGroups {
		groupPermission := &p.ProjectGroups[i]

		// Insert group
		groupID, new, errGroup := group.AddGroup(tx, &groupPermission.Group)
		if groupID == 0 {
			return errGroup
		}
		groupPermission.Group.ID = groupID

		// Add group on project
		if err := group.InsertGroupInProject(tx, p.ID, groupPermission.Group.ID, groupPermission.Permission); err != nil {
			log.Warning("addProject: Cannot add group %s in project %s:  %s\n", groupPermission.Group.Name, p.Name, err)
			return err

		}

		// Add user in group
		if new {
			if err := group.InsertUserInGroup(tx, groupPermission.Group.ID, c.User.ID, true); err != nil {
				log.Warning("addProject: Cannot add user %s in group %s:  %s\n", c.User.Username, groupPermission.Group.Name, err)
				return err

			}
		}
	}

	for _, v := range p.Variable {
		var errVar error
		switch v.Type {
		case sdk.KeyVariable:
			errVar = project.AddKeyPair(tx, p, v.Name, c.User)
		default:
			errVar = project.InsertVariable(tx, p, &v, c.User)
		}
		if errVar != nil {
			log.Warning("addProject: Cannot add variable %s in project %s:  %s\n", v.Name, p.Name, errVar)
			return errVar
		}
	}

	if err := project.UpdateLastModified(tx, c.User, p); err != nil {
		log.Warning("addProject: Cannot update last modified:  %s\n", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("addProject: Cannot commit transaction:  %s\n", err)
		return err
	}

	return WriteJSON(w, r, p, http.StatusCreated)
}

func deleteProjectHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	p, errProj := project.Load(db, key, c.User, project.LoadOptions.WithPipelines)
	if errProj != nil {
		if errProj != sdk.ErrNoProject {
			log.Warning("deleteProject: load project '%s' from db: %s\n", key, errProj)
		}
		return errProj
	}

	if len(p.Pipelines) > 0 {
		log.Warning("deleteProject> Project '%s' still used by %d pipelines\n", key, len(p.Pipelines))
		return sdk.ErrProjectHasPipeline
	}

	if len(p.Applications) > 0 {
		log.Warning("deleteProject> Project '%s' still used by %d applications\n", key, len(p.Applications))
		return sdk.ErrProjectHasApplication
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		log.Warning("deleteProject: Cannot start transaction: %s\n", errBegin)
		return errBegin
	}
	defer tx.Rollback()

	if err := project.Delete(tx, p.Key); err != nil {
		log.Warning("deleteProject: cannot delete project %s: %s\n", err)
		return err

	}
	if err := tx.Commit(); err != nil {
		log.Warning("deleteProject: Cannot commit transaction: %s\n", err)
		return err
	}
	log.Notice("Project %s deleted.\n", p.Name)

	return nil

}

func getUserLastUpdates(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	sinceHeader := r.Header.Get("If-Modified-Since")
	since := time.Unix(0, 0)
	if sinceHeader != "" {
		since, _ = time.Parse(time.RFC1123, sinceHeader)
	}

	lastUpdates, errUp := project.LastUpdates(db, c.User, since)
	if errUp != nil {
		if errUp == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotModified)
			return nil
		}
		return errUp
	}
	if len(lastUpdates) == 0 {
		w.WriteHeader(http.StatusNotModified)
		return nil
	}

	return WriteJSON(w, r, lastUpdates, http.StatusOK)
}
