package main

import (
	"database/sql"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"time"

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

func getProjects(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	includePipeline := r.FormValue("pipeline")
	includeApplication := r.FormValue("application")
	includeEnvironment := r.FormValue("environment")
	applicationStatus := r.FormValue("applicationStatus")

	projects, err := project.LoadProjects(db, c.User)
	if err != nil {
		log.Warning("GetProjects: Cannot load project from db: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if includeApplication == "true" {
		for _, p := range projects {
			applications, err := application.LoadApplications(db, p.Key, includePipeline == "true", c.User)
			if err != nil {
				log.Warning("GetProjects: Cannot load applications for projects %s : %s\n", p.Key, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			p.Applications = append(p.Applications, applications...)
		}
	}

	if includeEnvironment == "true" {
		for _, p := range projects {
			envs, err := environment.LoadEnvironments(db, p.Key, true, c.User)
			if err != nil {
				log.Warning("GetProjects: Cannot load environments for projects %s : %s\n", p.Key, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			p.Environments = append(p.Environments, envs...)
		}
	}

	if applicationStatus == "true" {
		for projectIndex := range projects {
			for appIndex := range projects[projectIndex].Applications {
				projects[projectIndex].Applications[appIndex].PipelinesBuild, err = pipeline.GetAllLastBuildByApplication(db, projects[projectIndex].Applications[appIndex].ID, "")
				if err != nil {
					log.Warning("GetProjects: Cannot load app status: %s\n", err)
					WriteError(w, r, err)
					return
				}
			}
		}
	}

	WriteJSON(w, r, projects, http.StatusOK)
}

func updateProject(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	projectArg, err := sdk.NewProject("").FromJSON(data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if projectArg.Name == "" {
		log.Warning("updateProject: Project name must no be empty")
		WriteError(w, r, sdk.ErrInvalidProjectName)
		return
	}

	// Check Request
	if key != projectArg.Key {
		log.Warning("updateProject: bad Project key %s/%s \n", key, projectArg.Key)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check is project exist
	p, err := project.LoadProject(db, key, c.User)
	if err != nil {
		log.Warning("updateProject: Cannot load project from db: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	lastModified, err := project.UpdateProjectDB(db, key, projectArg.Name)
	if err != nil {
		log.Warning("updateProject: Cannot update project %s : %s\n", key, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	p.Name = projectArg.Name
	p.LastModified = lastModified.Unix()

	WriteJSON(w, r, p, http.StatusOK)
}

func getProject(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	historyLengthString := r.FormValue("applicationHistory")
	applicationStatus := r.FormValue("applicationStatus")

	historyLength := 0
	var err error
	if historyLengthString != "" {
		historyLength, err = strconv.Atoi(historyLengthString)
		if err != nil {
			log.Warning("getProject: applicationHistory must be an integer: %s\n", err)
			WriteError(w, r, err)
			return
		}
	}

	p, err := project.LoadProject(db, key, c.User, project.WithVariables(), project.WithApplications(historyLength))
	if err != nil {
		log.Warning("getProject: Cannot load project from db: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	pipelines, err := pipeline.LoadPipelines(db, p.ID, false, c.User)
	if err != nil {
		log.Warning("getProject: Cannot load pipelines from db: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	p.Pipelines = append(p.Pipelines, pipelines...)

	envs, err := environment.LoadEnvironments(db, key, true, c.User)
	if err != nil {
		log.Warning("getProject: Cannot load environments from db: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	p.Environments = append(p.Environments, envs...)

	err = group.LoadGroupByProject(db, p)
	if err != nil {
		log.Warning("getProject: Cannot load groups from db: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	p.Permission = permission.ProjectPermission(p.Key, c.User)

	for i := range p.Environments {
		env := &p.Environments[i]
		env.Permission = permission.EnvironmentPermission(env.ID, c.User)
	}

	if applicationStatus == "true" {
		for i := range p.Applications {
			p.Applications[i].PipelinesBuild, err = pipeline.GetAllLastBuildByApplication(db, p.Applications[i].ID, "")
			if err != nil {
				log.Warning("GetProject: Cannot load app status: %s\n", err)
				WriteError(w, r, err)
				return
			}
		}
	}

	p.ReposManager, err = repositoriesmanager.LoadAllForProject(db, p.Key)
	if err != nil {
		log.Warning("GetProject: Cannot load repos manager for project %s: %s\n", p.Key, err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, p, http.StatusOK)
}

func addProject(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	p, err := sdk.NewProject("").FromJSON(data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// check projectKey pattern
	rgxp := regexp.MustCompile(sdk.ProjectKeyPattern)
	if !rgxp.MatchString(p.Key) {
		log.Warning("AddProject: Project key %s do not respect pattern %s", p.Key, sdk.ProjectKeyPattern)
		WriteError(w, r, sdk.ErrInvalidProjectKey)
		return
	}

	if p.Name == "" {
		log.Warning("AddProject: Project name must no be empty")
		WriteError(w, r, sdk.ErrInvalidProjectName)
		return
	}

	// Check that project does not already exists
	exist, err := project.Exist(db, p.Key)
	if err != nil {
		log.Warning("AddProject: Cannot check if project %s exist: %s\n", p.Key, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if exist {
		log.Warning("AddProject: Project %s already exists\n", p.Key)
		// Write nice error message here
		w.WriteHeader(http.StatusConflict)
		return
	}

	tx, err := db.Begin()
	defer tx.Rollback()
	if err != nil {
		log.Warning("AddProject: Cannot start transaction: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(p.Applications) == 1 && p.Applications[0].BuildTemplate.ID != 0 {
		err = project.CreateFromWizard(tx, p, c.User)
		if err != nil {
			log.Warning("AddProject: Cannot create project: %s\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else {
		// Project without application
		err = project.InsertProject(tx, p)
		if err != nil {
			log.Warning("AddProject: Cannot insert project: %s\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Add group
		for i := range p.ProjectGroups {
			groupPermission := &p.ProjectGroups[i]

			// Insert group
			groupID, new, err := group.AddGroup(tx, &groupPermission.Group)
			if groupID == 0 {
				WriteError(w, r, err)
				return
			}
			groupPermission.Group.ID = groupID

			// Add group on project
			err = group.InsertGroupInProject(tx, p.ID, groupPermission.Group.ID, groupPermission.Permission)
			if err != nil {
				log.Warning("addProject: Cannot add group %s in project %s:  %s\n", groupPermission.Group.Name, p.Name, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Add user in group
			if new {
				err = group.InsertUserInGroup(tx, groupPermission.Group.ID, c.User.ID, true)
				if err != nil {
					log.Warning("addProject: Cannot add user %s in group %s:  %s\n", c.User.Username, groupPermission.Group.Name, err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
		}

		var newPipeline sdk.Pipeline
		// Add pipeline if at leat 1 application was added
		if len(p.Applications) > 0 {
			newPipeline.Name = "build"
			newPipeline.Type = sdk.BuildPipeline
			newPipeline.ProjectID = p.ID

			err := pipeline.InsertPipeline(tx, &newPipeline)
			if err != nil {
				log.Warning("addProject: Cannot add build pipeline for project %s: %s \n", p.Name, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			err = group.InsertGroupsInPipeline(tx, p.ProjectGroups, newPipeline.ID)
			if err != nil {
				log.Warning("addProject> Cannot add groups on pipeline: %s\n", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// add parameter for repository
			param := sdk.Parameter{
				Name:        "urlRepository",
				Type:        sdk.StringParameter,
				Description: "Url of the source repository",
				Value:       "",
			}
			err = pipeline.InsertParameterInPipeline(tx, newPipeline.ID, &param)
			if err != nil {
				log.Warning("addProject: Cannot add pipeline parameter : %s \n", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		// Add application
		for _, app := range p.Applications {
			//app.Project = p
			err = application.InsertApplication(tx, p, &app)
			if err != nil {
				log.Warning("addProject: Cannot add application %s in project %s: %s \n", app.Name, p.Name, err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Add Groups
			err = group.InsertGroupsInApplication(tx, p.ProjectGroups, app.ID)
			if err != nil {
				log.Warning("addProject> Cannot add groups on application: %s\n", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Attach pipeline to application
			if newPipeline.ID != 0 {
				err = application.AttachPipeline(tx, app.ID, newPipeline.ID)
				if err != nil {
					log.Warning("addProject: Cannot attach pipeline %s to application %s : %s\n", newPipeline.Name, app.Name, err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}

			// Add variable
			for _, v := range app.Variable {
				variable := sdk.Variable{
					Name:  v.Name,
					Type:  v.Type,
					Value: v.Value,
				}
				err = application.InsertVariable(tx, app.ID, variable)
				if err != nil {
					log.Warning("addProject: Cannot add variable  %s in application %s: %s \n", v.Name, app.Name, err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				if newPipeline.ID != 0 && v.Name == "repositoryUrl" {
					var params []sdk.Parameter
					p := sdk.Parameter{
						Name:  "urlRepository",
						Type:  "string",
						Value: "{{.cds.app.repositoryUrl}}",
					}
					params = append(params, p)
					err = application.UpdatePipelineApplication(tx, app.ID, newPipeline.ID, params)
				}
			}

		}
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("addProject: Cannot commit transaction:  %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	WriteJSON(w, r, p, http.StatusCreated)
	log.Notice("addProject> Project %s created\n", p.Name)
}

func deleteProject(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	p, err := project.LoadProject(db, key, c.User)
	if err != nil {
		if err != sdk.ErrNoProject {
			log.Warning("deleteProject: load project '%s' from db: %s\n", key, err)
		}
		WriteError(w, r, err)
		return
	}

	countPipeline, err := pipeline.CountPipelineByProject(db, p.ID)
	if err != nil {
		log.Warning("deleteProject: Cannot count pipeline for project %s: %s\n", p.Name, err)
		WriteError(w, r, err)
		return
	}
	if countPipeline > 0 {
		log.Warning("deleteProject> Project '%s' still used by %d pipelines\n", key, countPipeline)
		WriteError(w, r, sdk.ErrProjectHasPipeline)
		return
	}

	countApplications, err := application.CountApplicationByProject(db, p.ID)
	if err != nil {
		log.Warning("deleteProject: Cannot count application for project %s: %s\n", p.Name, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if countApplications > 0 {
		log.Warning("deleteProject> Project '%s' still used by %d applications\n", key, countApplications)
		WriteError(w, r, sdk.ErrProjectHasApplication)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteProject: Cannot start transaction: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	err = project.DeleteProject(tx, p.Key)
	if err != nil {
		log.Warning("deleteProject: cannot delete project %s: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	err = tx.Commit()
	if err != nil {
		log.Warning("deleteProject: Cannot commit transaction: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Notice("Project %s deleted.\n", p.Name)

	w.WriteHeader(http.StatusOK)
	return
}

func getUserLastUpdates(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	sinceHeader := r.Header.Get("If-Modified-Since")
	since := time.Unix(0, 0)
	if sinceHeader != "" {
		since, _ = time.Parse(time.RFC1123, sinceHeader)
	}

	log.Debug("getUserLastUpdates> search updates since %v", since)
	lastUpdates, err := project.LastUpdates(db, c.User, since)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		WriteError(w, r, err)
		return
	}
	if len(lastUpdates) == 0 {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	WriteJSON(w, r, lastUpdates, http.StatusOK)

}
