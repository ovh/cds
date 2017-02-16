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
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/hook"
	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/poller"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/api/scheduler"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func getApplicationsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]

	applications, err := application.LoadApplications(db, projectKey, false, false, c.User)
	if err != nil {
		log.Warning("getApplicationsHandler: Cannot load applications from db: %s\n", err)
		return err
	}

	return WriteJSON(w, r, applications, http.StatusOK)
}

func getApplicationTreeHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {

	vars := mux.Vars(r)
	projectKey := vars["key"]
	applicationName := vars["permApplicationName"]

	tree, err := application.LoadCDTree(db, projectKey, applicationName, c.User)
	if err != nil {
		log.Warning("getApplicationTreeHandler: Cannot load CD Tree for applications %s: %s\n", applicationName, err)
		return err
	}

	return WriteJSON(w, r, tree, http.StatusOK)
}

func getPipelineBuildBranchHistoryHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]

	err := r.ParseForm()
	if err != nil {
		log.Warning("getPipelineBranchHistoryHandler> Cannot parse form: %s\n", err)
		return sdk.ErrUnknownError
	}

	pageString := r.Form.Get("page")
	nbPerPageString := r.Form.Get("perPage")

	var nbPerPage int
	if nbPerPageString != "" {
		nbPerPage, err = strconv.Atoi(nbPerPageString)
		if err != nil {
			return err
		}
	} else {
		nbPerPage = 20
	}

	var page int
	if pageString != "" {
		page, err = strconv.Atoi(pageString)
		if err != nil {
			return err
		}
	} else {
		nbPerPage = 0
	}

	pbs, err := pipeline.GetBranchHistory(db, projectKey, appName, page, nbPerPage)
	if err != nil {
		log.Warning("getPipelineBranchHistoryHandler> Cannot get history by branch: %s", err)
		return fmt.Errorf("Cannot load pipeline branch history: %s", err)
	}

	return WriteJSON(w, r, pbs, http.StatusOK)
}

func getApplicationDeployHistoryHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]

	pbs, err := pipeline.GetDeploymentHistory(db, projectKey, appName)
	if err != nil {
		log.Warning("getPipelineDeployHistoryHandler> Cannot get history by env: %s", err)
		return fmt.Errorf("Cannot load pipeline deployment history: %s", err)
	}

	return WriteJSON(w, r, pbs, http.StatusOK)
}

func getApplicationBranchVersionHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	applicationName := vars["permApplicationName"]

	branch := r.FormValue("branch")

	app, err := application.LoadApplicationByName(db, projectKey, applicationName)
	if err != nil {
		log.Warning("getApplicationBranchVersionHandler: Cannot load application %s for project %s from db: %s\n", applicationName, projectKey, err)
		return err
	}

	versions, err := pipeline.GetVersions(db, app, branch)
	if err != nil {
		log.Warning("getApplicationBranchVersionHandler: Cannot load version for application %s on branch %s: %s\n", applicationName, branch, err)
		return err
	}

	return WriteJSON(w, r, versions, http.StatusOK)
}

func getApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	applicationName := vars["permApplicationName"]

	applicationStatus := r.FormValue("applicationStatus")
	withPollers := r.FormValue("withPollers")
	withHooks := r.FormValue("withHooks")
	withNotifs := r.FormValue("withNotifs")
	withWorkflow := r.FormValue("withWorkflow")
	withTriggers := r.FormValue("withTriggers")
	withSchedulers := r.FormValue("withSchedulers")
	branchName := r.FormValue("branchName")
	versionString := r.FormValue("version")

	app, errApp := application.LoadApplicationByName(db, projectKey, applicationName)
	if errApp != nil {
		log.Warning("getApplicationHandler: Cannot load application %s for project %s from db: %s\n", applicationName, projectKey, errApp)
		return errApp
	}

	if withPollers == "true" {
		var errPoller error
		app.RepositoryPollers, errPoller = poller.LoadByApplication(db, app.ID)
		if errPoller != nil {
			log.Warning("getApplicationHandler: Cannot load pollers for application %s: %s\n", applicationName, errPoller)
			return errPoller
		}

	}

	if withSchedulers == "true" {
		var errScheduler error
		app.Schedulers, errScheduler = scheduler.GetByApplication(db, app)
		if errScheduler != nil {
			log.Warning("getApplicationHandler: Cannot load schedulers for application %s: %s\n", applicationName, errScheduler)
			return errScheduler
		}
	}

	if withHooks == "true" {
		var errHook error
		app.Hooks, errHook = hook.LoadApplicationHooks(db, app.ID)
		if errHook != nil {
			log.Warning("getApplicationHandler: Cannot load hooks for application %s: %s\n", applicationName, errHook)
			return errHook
		}
	}

	if withNotifs == "true" {
		var errNotif error
		app.Notifications, errNotif = notification.LoadAllUserNotificationSettings(db, app.ID)
		if errNotif != nil {
			log.Warning("getApplicationHandler: Cannot load user notifications for application %s: %s\n", applicationName, errNotif)
			return errNotif
		}
	}

	if withTriggers == "true" {
		for i := range app.Pipelines {
			appPip := &app.Pipelines[i]
			var errTrig error
			appPip.Triggers, errTrig = trigger.LoadTriggersByAppAndPipeline(db, app.ID, appPip.Pipeline.ID)
			if errTrig != nil {
				log.Warning("getApplicationHandler: Cannot load triggers: %s\n", errTrig)
				return errTrig
			}
		}
	}

	if withWorkflow == "true" {
		var errWorflow error
		app.Workflows, errWorflow = application.LoadCDTree(db, projectKey, applicationName, c.User)
		if errWorflow != nil {
			log.Warning("getApplicationHandler: Cannot load CD Tree for applications %s: %s\n", app.Name, errWorflow)
			return errWorflow
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
				return errStatus
			}
		}

		if version == 0 {
			var errBuilds error
			pipelineBuilds, errBuilds = pipeline.GetAllLastBuildByApplication(db, app.ID, branchName, 0)
			if errBuilds != nil {
				log.Warning("getApplicationHandler: Cannot load app status: %s\n", errBuilds)
				return errBuilds
			}
		} else {
			if branchName == "" {
				log.Warning("getApplicationHandler: branchName must be provided with version param\n")
				return sdk.ErrBranchNameNotProvided
			}
			var errPipBuilds error
			pipelineBuilds, errPipBuilds = pipeline.GetAllLastBuildByApplication(db, app.ID, branchName, version)
			if errPipBuilds != nil {
				log.Warning("getApplicationHandler: Cannot load app status by version: %s\n", errPipBuilds)
				return errPipBuilds
			}
		}
		app.PipelinesBuild = pipelineBuilds
	}

	app.Permission = permission.ApplicationPermission(app.ID, c.User)

	return WriteJSON(w, r, app, http.StatusOK)
}

func getApplicationBranchHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	applicationName := vars["permApplicationName"]

	application, err := application.LoadApplicationByName(db, projectKey, applicationName)
	if err != nil {
		log.Warning("getApplicationBranchHandler: Cannot load application %s for project %s from db: %s\n", applicationName, projectKey, err)
		return err
	}

	var branches []sdk.VCSBranch
	if application.RepositoryFullname != "" && application.RepositoriesManager != nil {
		client, err := repositoriesmanager.AuthorizedClient(db, projectKey, application.RepositoriesManager.Name)
		if err != nil {
			log.Warning("getApplicationBranchHandler> Cannot get client got %s %s : %s", projectKey, application.RepositoriesManager.Name, err)
			return sdk.ErrNoReposManagerClientAuth
		}
		branches, err = client.Branches(application.RepositoryFullname)
		if err != nil {
			log.Warning("getApplicationBranchHandler> Cannot get branches from repository %s: %s", application.RepositoryFullname, err)
			return sdk.ErrNoReposManagerClientAuth
		}

	} else {
		branches, err = pipeline.GetBranches(db, application)
		if err != nil {
			log.Warning("getApplicationBranchHandler> Cannot get branches from builds: %s", err)
			return err
		}
	}

	return WriteJSON(w, r, branches, http.StatusOK)
}

func addApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	projectData, err := project.Load(db, key, c.User)
	if err != nil {
		log.Warning("addApplicationHandler: Cannot load %s: %s\n", key, err)
		return err
	}

	var app sdk.Application
	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("addApplicationHandler: Cannot read body: %s\n", err)
		return sdk.ErrWrongRequest
	}
	err = json.Unmarshal(data, &app)
	if err != nil {
		log.Warning("addApplicationHandler: Cannot unmarshal request: %s\n", err)
		return sdk.ErrWrongRequest
	}

	// check application name pattern
	regexp := regexp.MustCompile(sdk.NamePattern)
	if !regexp.MatchString(app.Name) {
		log.Warning("addApplicationHandler: Application name %s do not respect pattern %s", app.Name, sdk.NamePattern)
		return sdk.ErrInvalidApplicationPattern
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("addApplicationHandler> Cannot start transaction: %s\n", err)
		return err
	}

	defer tx.Rollback()

	err = application.InsertApplication(tx, projectData, &app)
	if err != nil {
		log.Warning("addApplicationHandler> Cannot insert pipeline: %s\n", err)
		return err
	}

	err = group.LoadGroupByProject(tx, projectData)
	if err != nil {
		log.Warning("addApplicationHandler> Cannot load group from project: %s\n", err)
		return err
	}

	err = group.InsertGroupsInApplication(tx, projectData.ProjectGroups, app.ID)
	if err != nil {
		log.Warning("addApplicationHandler> Cannot add groups on application: %s\n", err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("addApplicationHandler> Cannot commit transaction: %s\n", err)
		return err
	}
	return nil
}

func deleteApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
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
		return err
	}

	nb, errNb := pipeline.CountBuildingPipelineByApplication(db, app.ID)
	if errNb != nil {
		log.Warning("deleteApplicationHandler> Cannot count pipeline build for application %d: %s\n", app.ID, errNb)
		return errNb
	}

	if nb > 0 {
		log.Warning("deleteApplicationHandler> Cannot delete application [%d], there are building pipelines: %d\n", app.ID, nb)
		return sdk.ErrAppBuildingPipelines
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteApplicationHandler> Cannot begin transaction: %s\n", err)
		return err
	}
	defer tx.Rollback()

	err = application.DeleteApplication(tx, app.ID)
	if err != nil {
		log.Warning("deleteApplicationHandler> Cannot delete application: %s\n", err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("deleteApplicationHandler> Cannot commit transaction: %s\n", err)
		return err
	}

	cache.DeleteAll(cache.Key("application", projectKey, "*"))
	cache.DeleteAll(cache.Key("pipeline", projectKey, "*"))

	return nil
}

func cloneApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	applicationName := vars["permApplicationName"]

	projectData, errProj := project.Load(db, projectKey, c.User)
	if errProj != nil {
		log.Warning("cloneApplicationHandler> Cannot load %s: %s\n", projectKey, errProj)
		return sdk.ErrNoProject
	}

	envs, errE := environment.LoadEnvironments(db, projectKey, true, c.User)
	if errProj != nil {
		log.Warning("cloneApplicationHandler> Cannot load Environments %s: %s\n", projectKey, errProj)
		return errE

	}
	projectData.Environments = envs

	var newApp sdk.Application
	// Get body
	data, errRead := ioutil.ReadAll(r.Body)
	if errRead != nil {
		return sdk.ErrWrongRequest
	}
	if err := json.Unmarshal(data, &newApp); err != nil {
		return sdk.ErrWrongRequest
	}

	appToClone, errApp := application.LoadApplicationByName(db, projectKey, applicationName)
	if errApp != nil {
		log.Warning("cloneApplicationHandler> Cannot load application %s: %s\n", applicationName, errApp)
		return errApp
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		log.Warning("cloneApplicationHandler> Cannot start transaction : %s\n", errBegin)
		return errBegin
	}
	defer tx.Rollback()

	if err := cloneApplication(tx, projectData, &newApp, appToClone); err != nil {
		log.Warning("cloneApplicationHandler> Cannot insert new application %s: %s\n", newApp.Name, err)
		return err
	}

	lastModified, errLM := project.UpdateProjectDB(tx, projectData.Key, projectData.Name)
	if errLM != nil {
		log.Warning("cloneApplicationHandler> Cannot update project last modified date: %s\n", errLM)
		return errLM
	}
	projectData.LastModified = lastModified

	if err := tx.Commit(); err != nil {
		log.Warning("cloneApplicationHandler> Cannot commit transaction : %s\n", err)
		return err
	}

	cache.DeleteAll(cache.Key("application", projectKey, "*"))
	cache.DeleteAll(cache.Key("pipeline", projectKey, "*"))

	return WriteJSON(w, r, newApp, http.StatusOK)
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

	if err := sanity.CheckApplication(db, project, newApp); err != nil {
		log.Warning("cloneApplication> Cannot check application sanity: %s\n", err)
		return err
	}

	return nil
}

func updateApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	applicationName := vars["permApplicationName"]

	p, err := project.Load(db, projectKey, c.User)
	if err != nil {
		log.Warning("updateApplicationHandler> Cannot load project %s: %s\n", projectKey, err)
		return err
	}
	envs, err := environment.LoadEnvironments(db, projectKey, true, c.User)
	if err != nil {
		log.Warning("updateApplicationHandler> Cannot load environments %s: %s\n", projectKey, err)
		return err
	}
	p.Environments = envs

	app, err := application.LoadApplicationByName(db, projectKey, applicationName)
	if err != nil {
		log.Warning("updateApplicationHandler> Cannot load application %s: %s\n", applicationName, err)
		return err
	}

	var appPost sdk.Application
	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("updateApplicationHandler> Cannot read body: %s\n", err)
		return sdk.ErrWrongRequest
	}
	err = json.Unmarshal(data, &appPost)
	if err != nil {
		log.Warning("updateApplicationHandler> Cannot unmarshal request: %s\n", err)
		return sdk.ErrWrongRequest
	}

	// check application name pattern
	regexp := regexp.MustCompile(sdk.NamePattern)
	if !regexp.MatchString(appPost.Name) {
		log.Warning("updateApplicationHandler: Application name %s do not respect pattern %s", appPost.Name, sdk.NamePattern)
		return sdk.ErrInvalidApplicationPattern
	}

	app.Name = appPost.Name

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateApplicationHandler> Cannot start transaction: %s\n", err)
		return err
	}
	defer tx.Rollback()

	if err := application.UpdateApplication(tx, app); err != nil {
		log.Warning("updateApplicationHandler> Cannot delete application %s: %s\n", applicationName, err)
		return err
	}

	if err := sanity.CheckApplication(tx, p, app); err != nil {
		log.Warning("updateApplicationHandler: Cannot check application sanity: %s\n", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("updateApplicationHandler> Cannot commit transaction: %s\n", err)
		return err
	}

	cache.DeleteAll(cache.Key("application", projectKey, "*"))
	cache.DeleteAll(cache.Key("pipeline", projectKey, "*"))

	return WriteJSON(w, r, app, http.StatusOK)

}
