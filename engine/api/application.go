package main

import (
	"fmt"
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
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/poller"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/api/scheduler"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func getApplicationsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["permProjectKey"]

	applications, err := application.LoadAll(db, projectKey, c.User)
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

	tree, err := workflow.LoadCDTree(db, projectKey, applicationName, c.User)
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

	app, err := application.LoadByName(db, projectKey, applicationName, c.User, application.LoadOptions.WithTriggers)
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

	applicationStatus := FormBool(r, "applicationStatus")
	withPollers := FormBool(r, "withPollers")
	withHooks := FormBool(r, "withHooks")
	withNotifs := FormBool(r, "withNotifs")
	withWorkflow := FormBool(r, "withWorkflow")
	withTriggers := FormBool(r, "withTriggers")
	withSchedulers := FormBool(r, "withSchedulers")
	branchName := r.FormValue("branchName")
	versionString := r.FormValue("version")

	loadOptions := []application.LoadOptionFunc{
		application.LoadOptions.WithVariables,
		application.LoadOptions.WithRepositoryManager,
		application.LoadOptions.WithVariables,
		application.LoadOptions.WithPipelines,
	}
	if withHooks {
		loadOptions = append(loadOptions, application.LoadOptions.WithHooks)
	}
	if withTriggers {
		loadOptions = append(loadOptions, application.LoadOptions.WithTriggers)
	}
	if withNotifs {
		loadOptions = append(loadOptions, application.LoadOptions.WithNotifs)
	}

	app, errApp := application.LoadByName(db, projectKey, applicationName, c.User, loadOptions...)
	if errApp != nil {
		log.Warning("getApplicationHandler: Cannot load application %s for project %s from db: %s\n", applicationName, projectKey, errApp)
		return errApp
	}

	if err := application.LoadGroupByApplication(db, app); err != nil {
		return sdk.WrapError(err, "getApplicationHandler> Unable to load groups by application")
	}

	if withPollers {
		var errPoller error
		app.RepositoryPollers, errPoller = poller.LoadByApplication(db, app.ID)
		if errPoller != nil {
			log.Warning("getApplicationHandler: Cannot load pollers for application %s: %s\n", applicationName, errPoller)
			return errPoller
		}
	}

	if withSchedulers {
		var errScheduler error
		app.Schedulers, errScheduler = scheduler.GetByApplication(db, app)
		if errScheduler != nil {
			log.Warning("getApplicationHandler: Cannot load schedulers for application %s: %s\n", applicationName, errScheduler)
			return errScheduler
		}
	}

	if withWorkflow {
		var errWorflow error
		app.Workflows, errWorflow = workflow.LoadCDTree(db, projectKey, applicationName, c.User)
		if errWorflow != nil {
			log.Warning("getApplicationHandler: Cannot load CD Tree for applications %s: %s\n", app.Name, errWorflow)
			return errWorflow
		}
	}

	if applicationStatus {
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

	app, err := application.LoadByName(db, projectKey, applicationName, c.User, application.LoadOptions.Default)
	if err != nil {
		log.Warning("getApplicationBranchHandler: Cannot load application %s for project %s from db: %s\n", applicationName, projectKey, err)
		return err
	}

	var branches []sdk.VCSBranch
	if app.RepositoryFullname != "" && app.RepositoriesManager != nil {
		client, err := repositoriesmanager.AuthorizedClient(db, projectKey, app.RepositoriesManager.Name)
		if err != nil {
			log.Warning("getApplicationBranchHandler> Cannot get client got %s %s : %s", projectKey, app.RepositoriesManager.Name, err)
			return sdk.ErrNoReposManagerClientAuth
		}
		branches, err = client.Branches(app.RepositoryFullname)
		if err != nil {
			log.Warning("getApplicationBranchHandler> Cannot get branches from repository %s: %s", app.RepositoryFullname, err)
			return sdk.ErrNoReposManagerClientAuth
		}
	} else {
		branches, err = pipeline.GetBranches(db, app)
		if err != nil {
			log.Warning("getApplicationBranchHandler> Cannot get branches from builds: %s", err)
			return err
		}
	}

	//Yo analyze branch and delete pipeline_build for old branches...

	return WriteJSON(w, r, branches, http.StatusOK)
}

func addApplicationHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	proj, err := project.Load(db, key, c.User)
	if err != nil {
		log.Warning("addApplicationHandler: Cannot load %s: %s\n", key, err)
		return err
	}

	var app sdk.Application
	if err := UnmarshalBody(r, &app); err != nil {
		return err
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

	err = application.Insert(tx, proj, &app)
	if err != nil {
		log.Warning("addApplicationHandler> Cannot insert pipeline: %s\n", err)
		return err
	}

	err = group.LoadGroupByProject(tx, proj)
	if err != nil {
		log.Warning("addApplicationHandler> Cannot load group from project: %s\n", err)
		return err
	}

	if err := application.AddGroup(tx, proj, &app, proj.ProjectGroups...); err != nil {
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

	app, err := application.LoadByName(db, projectKey, applicationName, c.User)
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

	proj, errProj := project.Load(db, projectKey, c.User)
	if errProj != nil {
		log.Warning("cloneApplicationHandler> Cannot load %s: %s\n", projectKey, errProj)
		return sdk.ErrNoProject
	}

	envs, errE := environment.LoadEnvironments(db, projectKey, true, c.User)
	if errProj != nil {
		log.Warning("cloneApplicationHandler> Cannot load Environments %s: %s\n", projectKey, errProj)
		return errE

	}
	proj.Environments = envs

	var newApp sdk.Application
	if err := UnmarshalBody(r, &newApp); err != nil {
		return err
	}

	appToClone, errApp := application.LoadByName(db, projectKey, applicationName, c.User, application.LoadOptions.Default, application.LoadOptions.WithGroups)
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

	if err := cloneApplication(tx, proj, &newApp, appToClone, c.User); err != nil {
		log.Warning("cloneApplicationHandler> Cannot insert new application %s: %s\n", newApp.Name, err)
		return err
	}

	if err := project.UpdateLastModified(tx, c.User, proj); err != nil {
		log.Warning("cloneApplicationHandler: Cannot update last modified date: %s\n", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("cloneApplicationHandler> Cannot commit transaction : %s\n", err)
		return err
	}

	cache.DeleteAll(cache.Key("application", projectKey, "*"))
	cache.DeleteAll(cache.Key("pipeline", projectKey, "*"))

	return WriteJSON(w, r, newApp, http.StatusOK)
}

// cloneApplication Clone an application with all her dependencies: pipelines, permissions, triggers
func cloneApplication(db gorp.SqlExecutor, proj *sdk.Project, newApp *sdk.Application, appToClone *sdk.Application, u *sdk.User) error {
	newApp.Pipelines = appToClone.Pipelines
	newApp.ApplicationGroups = appToClone.ApplicationGroups

	// Create Application
	if err := application.Insert(db, proj, newApp); err != nil {
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
			errVar = application.AddKeyPairToApplication(db, newApp, v.Name, u)
		} else {
			errVar = application.InsertVariable(db, newApp, v, u)
		}
		if errVar != nil {
			return errVar
		}
	}

	// Attach pipeline + Set pipeline parameters
	for _, appPip := range newApp.Pipelines {
		if _, err := application.AttachPipeline(db, newApp.ID, appPip.Pipeline.ID); err != nil {
			return err
		}

		if err := application.UpdatePipelineApplication(db, newApp, appPip.Pipeline.ID, appPip.Parameters, u); err != nil {
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

	//Reload trigger
	for i := range newApp.Pipelines {
		appPip := &newApp.Pipelines[i]
		var errTrig error
		appPip.Triggers, errTrig = trigger.LoadTriggersByAppAndPipeline(db, newApp.ID, appPip.Pipeline.ID)
		if errTrig != nil {
			log.Warning("cloneApplication> Cannot load triggers: %s\n", errTrig)
			return errTrig
		}
	}

	// Insert Permission
	if err := application.AddGroup(db, proj, newApp, newApp.ApplicationGroups...); err != nil {
		return err
	}

	if err := sanity.CheckApplication(db, proj, newApp); err != nil {
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

	p, err := project.Load(db, projectKey, c.User, project.LoadOptions.Default)
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

	app, err := application.LoadByName(db, projectKey, applicationName, c.User, application.LoadOptions.Default)
	if err != nil {
		log.Warning("updateApplicationHandler> Cannot load application %s: %s\n", applicationName, err)
		return err
	}

	var appPost sdk.Application
	if err := UnmarshalBody(r, &appPost); err != nil {
		return err
	}

	// check application name pattern
	regexp := regexp.MustCompile(sdk.NamePattern)
	if !regexp.MatchString(appPost.Name) {
		log.Warning("updateApplicationHandler: Application name %s do not respect pattern %s", appPost.Name, sdk.NamePattern)
		return sdk.ErrInvalidApplicationPattern
	}

	//Update name and Metadata
	app.Name = appPost.Name
	app.Metadata = appPost.Metadata

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateApplicationHandler> Cannot start transaction: %s\n", err)
		return err
	}
	defer tx.Rollback()
	if err := application.Update(tx, app); err != nil {
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
