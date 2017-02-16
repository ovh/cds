package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func getProjects(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	projects, err := project.LoadAll(db, c.User)
	if err != nil {
		log.Warning("GetProjects> Cannot load projects from db: %s\n", err)
		return err
	}
	return WriteJSON(w, r, projects, http.StatusOK)
}

func updateProject(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	// Get body
	data, errRead := ioutil.ReadAll(r.Body)
	if errRead != nil {
		return sdk.ErrWrongRequest
	}

	proj := &sdk.Project{}
	if err := json.Unmarshal(data, proj); err != nil {
		return sdk.ErrWrongRequest
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

	lastModified, errUp := project.UpdateProjectDB(db, key, proj.Name)
	if errUp != nil {
		log.Warning("updateProject: Cannot update project %s : %s\n", key, errUp)
		return errUp
	}

	p.Name = proj.Name
	p.LastModified = lastModified

	return WriteJSON(w, r, p, http.StatusOK)
}

func getProject(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	historyLengthString := r.FormValue("applicationHistory")
	applicationStatus := r.FormValue("applicationStatus")

	historyLength := 0

	if historyLengthString != "" {
		var errAtoi error
		historyLength, errAtoi = strconv.Atoi(historyLengthString)
		if errAtoi != nil {
			log.Warning("getProject: applicationHistory must be an integer: %s\n", errAtoi)
			return errAtoi
		}
	}

	p, errProj := project.Load(db, key, c.User, project.WithVariables(), project.WithApplications(historyLength))
	if errProj != nil {
		log.Warning("getProject: Cannot load project from db: %s\n", errProj)
		return errProj
	}

	pipelines, errPip := pipeline.LoadPipelines(db, p.ID, false, c.User)
	if errPip != nil {
		log.Warning("getProject: Cannot load pipelines from db: %s\n", errPip)
		return errPip
	}
	p.Pipelines = append(p.Pipelines, pipelines...)

	envs, errEnv := environment.LoadEnvironments(db, key, true, c.User)
	if errEnv != nil {
		log.Warning("getProject: Cannot load environments from db: %s\n", errEnv)
		return errEnv

	}
	p.Environments = append(p.Environments, envs...)

	if err := group.LoadGroupByProject(db, p); err != nil {
		log.Warning("getProject: Cannot load groups from db: %s\n", err)
		return err

	}

	p.Permission = permission.ProjectPermission(p.Key, c.User)

	for i := range p.Environments {
		env := &p.Environments[i]
		env.Permission = permission.EnvironmentPermission(env.ID, c.User)
	}

	if applicationStatus == "true" {
		for i := range p.Applications {
			var errBuild error
			p.Applications[i].PipelinesBuild, errBuild = pipeline.GetAllLastBuildByApplication(db, p.Applications[i].ID, "", 0)
			if errBuild != nil {
				log.Warning("GetProject: Cannot load app status: %s\n", errBuild)
				return errBuild
			}
		}
	}

	var errRepos error
	p.ReposManager, errRepos = repositoriesmanager.LoadAllForProject(db, p.Key)
	if errRepos != nil {
		log.Warning("GetProject: Cannot load repos manager for project %s: %s\n", p.Key, errRepos)
		return errRepos
	}

	return WriteJSON(w, r, p, http.StatusOK)
}

func addProject(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get body
	data, errRead := ioutil.ReadAll(r.Body)
	if errRead != nil {
		return sdk.ErrWrongRequest

	}

	//Unmarshal data
	p := &sdk.Project{}
	if err := json.Unmarshal(data, &p); err != nil {
		return sdk.ErrWrongRequest

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

	if err := project.InsertProject(tx, p); err != nil {
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
			errVar = project.AddKeyPairToProject(tx, p, v.Name)
		default:
			errVar = project.InsertVariableInProject(tx, p, v)
		}
		if errVar != nil {
			log.Warning("addProject: Cannot add variable %s in project %s:  %s\n", v.Name, p.Name, errVar)
			return errVar
		}
	}

	if err := tx.Commit(); err != nil {
		log.Warning("addProject: Cannot commit transaction:  %s\n", err)
		return err
	}

	return WriteJSON(w, r, p, http.StatusCreated)
}

func deleteProject(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	p, errProj := project.Load(db, key, c.User)
	if errProj != nil {
		if errProj != sdk.ErrNoProject {
			log.Warning("deleteProject: load project '%s' from db: %s\n", key, errProj)
		}
		return errProj
	}

	countPipeline, errCount := pipeline.CountPipelineByProject(db, p.ID)
	if errCount != nil {
		log.Warning("deleteProject: Cannot count pipeline for project %s: %s\n", p.Name, errCount)
		return errCount
	}
	if countPipeline > 0 {
		log.Warning("deleteProject> Project '%s' still used by %d pipelines\n", key, countPipeline)
		return sdk.ErrProjectHasPipeline
	}

	countApplications, errCountApp := application.CountApplicationByProject(db, p.ID)
	if errCountApp != nil {
		log.Warning("deleteProject: Cannot count application for project %s: %s\n", p.Name, errCountApp)
		return errCountApp
	}
	if countApplications > 0 {
		log.Warning("deleteProject> Project '%s' still used by %d applications\n", key, countApplications)
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
