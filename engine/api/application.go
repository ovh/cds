package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/hook"
	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/poller"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"

)

func getApplicationsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]

	applications, err := application.LoadApplications(db, projectKey, false, c.User)
	if err != nil {
		log.Warning("getApplicationsHandler: Cannot load applications from db: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	WriteJSON(w, r, applications, http.StatusOK)
}

func getApplicationTreeHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {

	vars := mux.Vars(r)
	projectKey := vars["key"]
	applicationName := vars["permApplicationName"]

	tree, err := application.LoadCDTree(db, projectKey, applicationName, c.User)
	if err != nil {
		log.Warning("getApplicationTreeHandler: Cannot load CD Tree for applications %s: %s\n", applicationName, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	WriteJSON(w, r, tree, http.StatusOK)
}

func getPipelineBuildBranchHistoryHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]

	err := r.ParseForm()
	if err != nil {
		log.Warning("getPipelineBranchHistoryHandler> Cannot parse form: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

	pageString := r.Form.Get("page")
	nbPerPageString := r.Form.Get("perPage")

	var nbPerPage int
	if nbPerPageString != "" {
		nbPerPage, err = strconv.Atoi(nbPerPageString)
		if err != nil {
			WriteError(w, r, err)
			return
		}
	} else {
		nbPerPage = 20
	}

	var page int
	if pageString != "" {
		page, err = strconv.Atoi(pageString)
		if err != nil {
			WriteError(w, r, err)
			return
		}
	} else {
		nbPerPage = 0
	}

	pbs, err := pipeline.GetBranchHistory(db, projectKey, appName, page, nbPerPage)
	if err != nil {
		log.Warning("getPipelineBranchHistoryHandler> Cannot get history by branch: %s", err)
		WriteError(w, r, fmt.Errorf("Cannot load pipeline branch history: %s", err))
		return
	}

	WriteJSON(w, r, pbs, http.StatusOK)

}

func getApplicationDeployHistoryHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]

	pbs, err := pipeline.GetDeploymentHistory(db, projectKey, appName)
	if err != nil {
		log.Warning("getPipelineDeployHistoryHandler> Cannot get history by env: %s", err)
		WriteError(w, r, fmt.Errorf("Cannot load pipeline deployment history: %s", err))
		return
	}

	WriteJSON(w, r, pbs, http.StatusOK)

}

func getApplicationBranchVersionHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	applicationName := vars["permApplicationName"]

	branch := r.FormValue("branch")

	app, err := application.LoadApplicationByName(db, projectKey, applicationName)
	if err != nil {
		log.Warning("getApplicationBranchVersionHandler: Cannot load application %s for project %s from db: %s\n", applicationName, projectKey, err)
		WriteError(w, r, err)
		return
	}

	versions, err := pipeline.GetVersions(db, app, branch)
	if err != nil {
		log.Warning("getApplicationBranchVersionHandler: Cannot load version for application %s on branch %s: %s\n", applicationName, branch, err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, versions, http.StatusOK)

}

func getApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	applicationName := vars["permApplicationName"]

	applicationStatus := r.FormValue("applicationStatus")
	withPollers := r.FormValue("withPollers")
	withHooks := r.FormValue("withHooks")
	withNotifs := r.FormValue("withNotifs")
	withWorkflow := r.FormValue("withWorkflow")
	withTriggers := r.FormValue("withTriggers")
	branchName := r.FormValue("branchName")
	versionString := r.FormValue("version")

	app, errApp := application.LoadApplicationByName(db, projectKey, applicationName)
	if errApp != nil {
		log.Warning("getApplicationHandler: Cannot load application %s for project %s from db: %s\n", applicationName, projectKey, errApp)
		WriteError(w, r, errApp)
		return
	}

	if withPollers == "true" {
		var errPoller error
		app.RepositoryPollers, errPoller = poller.LoadPollersByApplication(db, app.ID)
		if errPoller != nil {
			log.Warning("getApplicationHandler: Cannot load pollers for application %s: %s\n", applicationName, errPoller)
			WriteError(w, r, errPoller)
			return
		}

	}

	if withHooks == "true" {
		var errHook error
		app.Hooks, errHook = hook.LoadApplicationHooks(db, app.ID)
		if errHook != nil {
			log.Warning("getApplicationHandler: Cannot load hooks for application %s: %s\n", applicationName, errHook)
			WriteError(w, r, errHook)
			return
		}
	}

	if withNotifs == "true" {
		var errNotif error
		app.Notifications, errNotif = notification.LoadAllUserNotificationSettings(db, app.ID)
		if errNotif != nil {
			log.Warning("getApplicationHandler: Cannot load user notifications for application %s: %s\n", applicationName, errNotif)
			WriteError(w, r, errNotif)
			return
		}
	}

	if withTriggers == "true" {
		for i := range app.Pipelines {
			appPip := &app.Pipelines[i]
			var errTrig error
			appPip.Triggers, errTrig = trigger.LoadTriggersByAppAndPipeline(db, app.ID, appPip.Pipeline.ID)
			if errTrig != nil {
				log.Warning("getApplicationHandler: Cannot load triggers: %s\n", errTrig)
				WriteError(w, r, errTrig)
				return
			}
		}
	}

	if withWorkflow == "true" {
		var errWorflow error
		app.Workflows, errWorflow = application.LoadCDTree(db, projectKey, applicationName, c.User)
		if errWorflow != nil {
			log.Warning("getApplicationHandler: Cannot load CD Tree for applications %s: %s\n", app.Name, errWorflow)
			WriteError(w, r, errWorflow)
			return
		}
	}

	if applicationStatus == "true" {
		var pipelineBuilds = []sdk.PipelineBuild{}

		version := 0
		if versionString != "" {
			var errStatus error
			version, errStatus = strconv.Atoi(versionString)
			if errStatus != nil {
				log.Warning("getApplicationHandler: Version %s is not an integer: %s\n", versionString, errStatus)
				WriteError(w, r, errStatus)
				return
			}
		}

		if version == 0 {
			var errBuilds error
			pipelineBuilds, errBuilds = pipeline.GetAllLastBuildByApplication(db, app.ID, branchName, 0)
			if errBuilds != nil {
				log.Warning("getApplicationHandler: Cannot load app status: %s\n", errBuilds)
				WriteError(w, r, errBuilds)
				return
			}
		} else {
			if branchName == "" {
				log.Warning("getApplicationHandler: branchName must be provided with version param\n")
				WriteError(w, r, sdk.ErrBranchNameNotProvided)
				return
			}
			var errPipBuilds error
			pipelineBuilds, errPipBuilds = pipeline.GetAllLastBuildByApplication(db, app.ID, branchName, version)
			if errPipBuilds != nil {
				log.Warning("getApplicationHandler: Cannot load app status by version: %s\n", errPipBuilds)
				WriteError(w, r, errPipBuilds)
				return
			}
		}
		app.PipelinesBuild = pipelineBuilds
	}

	app.Permission = permission.ApplicationPermission(app.ID, c.User)

	WriteJSON(w, r, app, http.StatusOK)
}

func getApplicationBranchHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	applicationName := vars["permApplicationName"]

	application, err := application.LoadApplicationByName(db, projectKey, applicationName)
	if err != nil {
		log.Warning("getApplicationBranchHandler: Cannot load application %s for project %s from db: %s\n", applicationName, projectKey, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var branches []sdk.VCSBranch
	if application.RepositoryFullname != "" && application.RepositoriesManager != nil {
		client, err := repositoriesmanager.AuthorizedClient(db, projectKey, application.RepositoriesManager.Name)
		if err != nil {
			log.Warning("getApplicationBranchHandler> Cannot get client got %s %s : %s", projectKey, application.RepositoriesManager.Name, err)
			WriteError(w, r, sdk.ErrNoReposManagerClientAuth)
			return
		}
		branches, err = client.Branches(application.RepositoryFullname)
		if err != nil {
			log.Warning("getApplicationBranchHandler> Cannot get branches from repository %s: %s", application.RepositoryFullname, err)
			WriteError(w, r, sdk.ErrNoReposManagerClientAuth)
			return
		}

	} else {
		branches, err = pipeline.GetBranches(db, application)
		if err != nil {
			log.Warning("getApplicationBranchHandler> Cannot get branches from builds: %s", err)
			WriteError(w, r, err)
			return
		}
	}

	WriteJSON(w, r, branches, http.StatusOK)
}

func addApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	projectData, err := project.LoadProject(db, key, c.User)
	if err != nil {
		log.Warning("addApplicationHandler: Cannot load %s: %s\n", key, err)
		WriteError(w, r, err)
		return
	}

	var app sdk.Application
	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("addApplicationHandler: Cannot read body: %s\n", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}
	err = json.Unmarshal(data, &app)
	if err != nil {
		log.Warning("addApplicationHandler: Cannot unmarshal request: %s\n", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	// check application name pattern
	regexp := regexp.MustCompile(sdk.NamePattern)
	if !regexp.MatchString(app.Name) {
		log.Warning("addApplicationHandler: Application name %s do not respect pattern %s", app.Name, sdk.NamePattern)
		WriteError(w, r, sdk.ErrInvalidApplicationPattern)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("addApplicationHandler> Cannot start transaction: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer tx.Rollback()

	err = application.InsertApplication(tx, projectData, &app)
	if err != nil {
		log.Warning("addApplicationHandler> Cannot insert pipeline: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = group.LoadGroupByProject(tx, projectData)
	if err != nil {
		log.Warning("addApplicationHandler> Cannot load group from project: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = group.InsertGroupsInApplication(tx, projectData.ProjectGroups, app.ID)
	if err != nil {
		log.Warning("addApplicationHandler> Cannot add groups on application: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("addApplicationHandler> Cannot commit transaction: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func deleteApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	applicationName := vars["permApplicationName"]

	cache.DeleteAll(cache.Key("application", projectKey, "*"))
	cache.DeleteAll(cache.Key("pipeline", projectKey, "*"))

	app, err := application.LoadApplicationByName(db, projectKey, applicationName)
	if err != nil {
		if err != sdk.ErrApplicationNotFound {
			log.Warning("deleteApplicationHandler> Cannot load application %s: %s\n", applicationName, err)
		}
		WriteError(w, r, err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteApplicationHandler> Cannot begin transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	err = application.DeleteApplication(tx, app.ID)
	if err != nil {
		log.Warning("deleteApplicationHandler> Cannot delete application: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("deleteApplicationHandler> Cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	cache.DeleteAll(cache.Key("application", projectKey, "*"))
	cache.DeleteAll(cache.Key("pipeline", projectKey, "*"))

	log.Notice("Application %s deleted.\n", applicationName)
	w.WriteHeader(http.StatusOK)
}

func cloneApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	applicationName := vars["permApplicationName"]

	projectData, errProj := project.LoadProject(db, projectKey, c.User)
	if errProj != nil {
		log.Warning("cloneApplicationHandler> Cannot load %s: %s\n", projectKey, errProj)
		WriteError(w, r, sdk.ErrNoProject)
		return
	}

	var newApp sdk.Application
	// Get body
	data, errRead := ioutil.ReadAll(r.Body)
	if errRead != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}
	if err := json.Unmarshal(data, &newApp); err != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	appToClone, errApp := application.LoadApplicationByName(db, projectKey, applicationName)
	if errApp != nil {
		log.Warning("cloneApplicationHandler> Cannot load application %s: %s\n", applicationName, errApp)
		WriteError(w, r, errApp)
		return
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		log.Warning("cloneApplicationHandler> Cannot start transaction : %s\n", errBegin)
		WriteError(w, r, errBegin)
		return
	}
	defer tx.Rollback()

	if err := cloneApplication(tx, projectData, &newApp, appToClone); err != nil {
		log.Warning("cloneApplicationHandler> Cannot insert new application %s: %s\n", newApp.Name, err)
		WriteError(w, r, err)
		return
	}

	lastModified, errLM := project.UpdateProjectDB(tx, projectData.Key, projectData.Name)
	if errLM != nil {
		log.Warning("cloneApplicationHandler> Cannot update project last modified date: %s\n", errLM)
		WriteError(w, r, errLM)
		return
	}
	projectData.LastModified = lastModified.Unix()

	if err := tx.Commit(); err != nil {
		log.Warning("cloneApplicationHandler> Cannot commit transaction : %s\n", err)
		WriteError(w, r, err)
		return
	}

	cache.DeleteAll(cache.Key("application", projectKey, "*"))
	cache.DeleteAll(cache.Key("pipeline", projectKey, "*"))

	WriteJSON(w, r, newApp, http.StatusOK)

}

// cloneApplication Clone an application with all her dependencies: pipelines, permissions, triggers
func cloneApplication(db gorp.SqlExecutor, project *sdk.Project, newApp *sdk.Application, appToClone *sdk.Application) error {
	newApp.Pipelines = appToClone.Pipelines
	newApp.ApplicationGroups = appToClone.ApplicationGroups

	// Create Application
	if err := application.InsertApplication(db, project, newApp); err != nil {
		return err
	}

	// Insert Permission
	if err := group.InsertGroupsInApplication(db, newApp.ApplicationGroups, newApp.ID); err != nil {
		return err
	}

	var variablesToDelete []string
	for _, v := range newApp.Variable {
		if v.Type == sdk.KeyVariable {
			variablesToDelete = append(variablesToDelete, fmt.Sprintf("%s.pub", v.Name))
		}
	}

	for _, vToDelete := range variablesToDelete {
		for i := range newApp.Variable {
			if vToDelete == newApp.Variable[i].Name {
				newApp.Variable = append(newApp.Variable[:i], newApp.Variable[i+1:]...)
				break
			}
		}
	}

	// Insert variable
	for _, v := range newApp.Variable {
		var errVar error
		// If variable is a key variable, generate a new one for this application
		if v.Type == sdk.KeyVariable {
			errVar = application.AddKeyPairToApplication(db, newApp, v.Name)
		} else {
			errVar = application.InsertVariable(db, newApp, v)
		}
		if errVar != nil {
			return errVar
		}
	}

	// Attach pipeline + Set pipeline parameters
	for _, appPip := range newApp.Pipelines {
		if err := application.AttachPipeline(db, newApp.ID, appPip.Pipeline.ID); err != nil {
			return err
		}

		if err := application.UpdatePipelineApplication(db, newApp, appPip.Pipeline.ID, appPip.Parameters); err != nil {
			return err
		}
	}

	// Load trigger to clone
	triggers, err := trigger.LoadTriggerByApp(db, appToClone.ID)
	if err != nil {
		return err
	}

	// Clone trigger
	for _, t := range triggers {
		// Insert new trigger
		if t.DestApplication.ID == appToClone.ID {
			t.DestApplication = *newApp
		}
		t.SrcApplication = *newApp
		if err := trigger.InsertTrigger(db, &t); err != nil {
			return err
		}
	}
	return nil
}

func updateApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	applicationName := vars["permApplicationName"]

	app, err := application.LoadApplicationByName(db, projectKey, applicationName)
	if err != nil {
		log.Warning("updateApplicationHandler> Cannot load application %s: %s\n", applicationName, err)
		WriteError(w, r, err)
		return
	}

	var appPost sdk.Application
	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("updateApplicationHandler> Cannot read body: %s\n", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}
	err = json.Unmarshal(data, &appPost)
	if err != nil {
		log.Warning("updateApplicationHandler> Cannot unmarshal request: %s\n", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	// check application name pattern
	regexp := regexp.MustCompile(sdk.NamePattern)
	if !regexp.MatchString(appPost.Name) {
		log.Warning("updateApplicationHandler: Application name %s do not respect pattern %s", appPost.Name, sdk.NamePattern)
		WriteError(w, r, sdk.ErrInvalidApplicationPattern)
		return
	}

	app.Name = appPost.Name

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateApplicationHandler> Cannot start transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	err = application.UpdateApplication(tx, app)
	if err != nil {
		log.Warning("updateApplicationHandler> Cannot delete application %s: %s\n", applicationName, err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("updateApplicationHandler> Cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	cache.DeleteAll(cache.Key("application", projectKey, "*"))
	cache.DeleteAll(cache.Key("pipeline", projectKey, "*"))

	WriteJSON(w, r, app, http.StatusOK)
}
