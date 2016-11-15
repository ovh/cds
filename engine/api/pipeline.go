package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/archivist"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/api/scheduler"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func rollbackPipelineHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	appName := vars["permApplicationName"]

	var request sdk.RunRequest

	// Get args in body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		WriteError(w, r, err)
		return
	}
	// Unmarshal args
	err = json.Unmarshal(data, &request)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	// Load application
	app, err := application.LoadApplicationByName(db, projectKey, appName, application.WithClearPassword())
	if err != nil {
		if err != sdk.ErrApplicationNotFound {
			log.Warning("rollbackPipelineHandler> Cannot load application %s: %s\n", appName, err)
		}
		WriteError(w, r, err)
		return
	}

	// Load pipeline
	pip, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		if err != sdk.ErrPipelineNotFound {
			log.Warning("rollbackPipelineHandler> Cannot load pipeline %s; %s\n", pipelineName, err)
		}
		WriteError(w, r, err)
		return
	}

	// Load Env
	var env *sdk.Environment
	if request.Env.Name != "" && request.Env.Name != sdk.DefaultEnv.Name {
		env, err = environment.LoadEnvironmentByName(db, projectKey, request.Env.Name)
		if err != nil {
			log.Warning("rollbackPipelineHandler> Cannot load environment %s; %s\n", request.Env.Name, err)
			WriteError(w, r, sdk.ErrNoEnvironment)
			return
		}

		if env.ID != sdk.DefaultEnv.ID && !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadExecute) {
			log.Warning("rollbackPipelineHandler> No enought right on this environment %s: \n", request.Env.Name)
			WriteError(w, r, sdk.ErrForbidden)
			return
		}

	} else {
		env = &sdk.DefaultEnv
	}

	if env.ID != sdk.DefaultEnv.ID && !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadExecute) {
		log.Warning("rollbackPipelineHandler> You do not have Execution Right on this environment %s\n", env.Name)
		WriteError(w, r, sdk.ErrNoEnvExecution)
		return
	}

	pbs, err := pipeline.LoadPipelineBuildHistoryByApplicationAndPipeline(db, app.ID, pip.ID, env.ID, 2, string(sdk.StatusSuccess), "", pipeline.WithParameters())
	if err != nil {
		log.Warning("rollbackPipelineHandler> Cannot load pipeline build history %s", err)
		WriteError(w, r, sdk.ErrNoPipelineBuild)
		return
	}

	if len(pbs) != 2 {
		log.Warning("rollbackPipelineHandler> There is no previous success for app %s(%d), pip %s(%d), env %s(%d): %d", app.Name, app.ID, pip.Name, pip.ID, env.Name, env.ID, len(pbs))
		WriteError(w, r, sdk.ErrNoPreviousSuccess)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("rollbackPipelineHandler> Cannot start tx: %s", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	trigger := pbs[1].Trigger
	trigger.TriggeredBy = c.User

	newPb, err := scheduler.Run(tx, projectKey, app, pipelineName, env.Name, pbs[1].Parameters, pbs[1].Version, trigger, c.User)
	if err != nil {
		log.Warning("rollbackPipelineHandler> Cannot run pipeline: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("rollbackPipelineHandler> Cannot commit tx: %s", err)
		WriteError(w, r, err)
		return
	}

	k := cache.Key("application", projectKey, "builds", "*")
	cache.DeleteAll(k)

	WriteJSON(w, r, newPb, http.StatusOK)
}

func loadDestEnvFromRunRequest(db *sql.DB, c *context.Context, request *sdk.RunRequest, projectKey string) (*sdk.Environment, error) {
	var envDest = &sdk.DefaultEnv
	var err error
	if request.Env.Name != "" && request.Env.Name != sdk.DefaultEnv.Name {
		envDest, err = environment.LoadEnvironmentByName(db, projectKey, request.Env.Name)
		if err != nil {
			log.Warning("loadDestEnvFromRunRequest> Cannot load destination environment: %s", err)
			return nil, sdk.ErrNoEnvironment
		}

		if envDest.ID != sdk.DefaultEnv.ID && !permission.AccessToEnvironment(envDest.ID, c.User, permission.PermissionReadExecute) {
			log.Warning("loadDestEnvFromRunRequest> No enought right on this environment %s: \n", request.Env.Name)
			return nil, sdk.ErrForbidden
		}
	}
	if envDest.ID != sdk.DefaultEnv.ID && !permission.AccessToEnvironment(envDest.ID, c.User, permission.PermissionReadExecute) {
		log.Warning("loadDestEnvFromRunRequest> You do not have Execution Right on this environment\n")
		return nil, sdk.ErrNoEnvExecution
	}
	return envDest, nil
}

func runPipelineWithLastParentHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	appName := vars["permApplicationName"]

	app, err := application.LoadApplicationByName(db, projectKey, appName, application.WithClearPassword())
	if err != nil {
		if err != sdk.ErrApplicationNotFound {
			log.Warning("runPipelineWithLastParentHandler> Cannot load application %s: %s\n", appName, err)
		}
		WriteError(w, r, err)
		return
	}

	// Get args in body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	var request sdk.RunRequest
	// Unmarshal args
	if err := json.Unmarshal(data, &request); err != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	//Check parent stuff
	if request.ParentApplicationID == 0 && request.ParentPipelineID == 0 {
		WriteError(w, r, sdk.ErrParentApplicationAndPipelineMandatory)
		return
	}
	envID := sdk.DefaultEnv.ID
	if request.ParentEnvironmentID != 0 {
		envID = request.ParentEnvironmentID
	}

	// Load pipeline
	pip, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		if err != sdk.ErrPipelineNotFound {
			log.Warning("runPipelineWithLastParentHandler> Cannot load pipeline %s; %s\n", pipelineName, err)
		}
		WriteError(w, r, err)
		return
	}

	// Check that pipeline is attached to application
	ok, err := application.PipelineAttached(db, app.ID, pip.ID)
	if !ok {
		log.Warning("runPipelineWithLastParentHandler> Pipeline %s is not attached to app %s\n", pipelineName, appName)
		WriteError(w, r, sdk.ErrPipelineNotAttached)
		return
	}
	if err != nil {
		log.Warning("runPipelineWithLastParentHandler> Cannot check if pipeline %s is attached to %s: %s\n", pipelineName, appName, err)
		WriteError(w, r, err)
		return
	}

	//Load environment
	envDest, err := loadDestEnvFromRunRequest(db, c, &request, projectKey)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	//Load triggers
	triggers, err := trigger.LoadTriggers(db, app.ID, pip.ID, envDest.ID)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	//Find parent trigger
	var trig *sdk.PipelineTrigger
	for i, t := range triggers {
		fmt.Printf("Trigger from app(%s[%d]) pip(%s[%d]) env(%s[%d]) to app(%s[%d]) pip(%s[%d]) env(%s[%d])\n", t.SrcApplication.Name, t.SrcApplication.ID, t.SrcPipeline.Name, t.SrcPipeline.ID, t.SrcEnvironment.Name, t.SrcEnvironment.ID, t.DestApplication.Name, t.DestApplication.ID, t.DestPipeline.Name, t.DestPipeline.ID, t.DestEnvironment.Name, t.DestEnvironment.ID)
		if t.SrcApplication.ID == request.ParentApplicationID &&
			t.SrcPipeline.ID == request.ParentPipelineID &&
			t.SrcEnvironment.ID == envID {
			trig = &triggers[i]
		}
	}

	//If trigger not found: exit
	if trig == nil {
		WriteError(w, r, sdk.ErrPipelineNotFound)
		return
	}

	//Branch
	var branch string
	for _, p := range request.Params {
		if p.Name == "git.branch" {
			branch = p.Value
		}
	}

	builds, err := pipeline.LoadPipelineBuildHistoryByApplicationAndPipeline(db, request.ParentApplicationID, request.ParentPipelineID, envID, 1, string(sdk.StatusSuccess), branch)
	if err != nil {
		log.Warning("runPipelineWithLastParentHandler> Unable to find any successfull pipeline build")
		WriteError(w, r, sdk.ErrNoParentBuildFound)
		return
	}
	if len(builds) == 0 {
		WriteError(w, r, sdk.ErrNoPipelineBuild)
	}

	request.ParentBuildNumber = builds[0].BuildNumber

	runPipelineHandlerFunc(w, r, db, c, &request)
}

func runPipelineHandlerFunc(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context, request *sdk.RunRequest) {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	appName := vars["permApplicationName"]

	// Load application to be send to scheduler.Run() from DB
	cache.DeleteAll(cache.Key("application", projectKey, "*"))
	cache.DeleteAll(cache.Key("pipeline", projectKey, "*"))

	app, err := application.LoadApplicationByName(db, projectKey, appName, application.WithClearPassword())
	if err != nil {
		if err != sdk.ErrApplicationNotFound {
			log.Warning("runPipelineHandler> Cannot load application %s: %s\n", appName, err)
		}
		WriteError(w, r, err)
		return
	}

	// Load pipeline
	pip, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		if err != sdk.ErrPipelineNotFound {
			log.Warning("runPipelineHandler> Cannot load pipeline %s; %s\n", pipelineName, err)
		}
		WriteError(w, r, err)
		return
	}

	// Check that pipeline is attached to application
	ok, err := application.PipelineAttached(db, app.ID, pip.ID)
	if !ok {
		log.Warning("runPipelineHandler> Pipeline %s is not attached to app %s\n", pipelineName, appName)
		WriteError(w, r, sdk.ErrPipelineNotAttached)
		return
	}
	if err != nil {
		log.Warning("runPipelineHandler> Cannot check if pipeline %s is attached to %s: %s\n", pipelineName, appName, err)
		WriteError(w, r, err)
		return
	}

	version := int64(0)
	// Load parent pipeline build + add parent variable
	var parentPipelineBuild *sdk.PipelineBuild
	envID := sdk.DefaultEnv.ID
	if request.ParentBuildNumber != 0 {
		if request.ParentEnvironmentID != 0 {
			envID = request.ParentEnvironmentID
		}
		pb, err := pipeline.LoadPipelineBuild(db, request.ParentPipelineID, request.ParentApplicationID, request.ParentBuildNumber, envID)
		if err != nil {
			if err != sdk.ErrNoPipelineBuild {
				log.Warning("runPipelineHandler> Cannot load parent pipeline build: %s\n", err)
				WriteError(w, r, err)
				return
			}
			log.Notice("Loading parent pipeline build %d from history", request.ParentPipelineID)
			pb, err = pipeline.LoadPipelineHistoryBuild(db, request.ParentPipelineID, request.ParentApplicationID, request.ParentBuildNumber, envID)
			if err != nil {
				log.Warning("runPipelineHandler> Cannot load parent pipeline build from history: %s\n", err)
				WriteError(w, r, err)
				return
			}

		}
		parentParams, err := scheduler.ParentBuildInfos(pb)
		if err != nil {
			log.Warning("runPipelineHandler> Cannot create parent build infos: %s\n", err)
			WriteError(w, r, err)
		}
		request.Params = append(request.Params, parentParams...)

		// Whether or not use parent build version is checked
		// in InsertPipelineBuild
		//if pip.Type != sdk.BuildPipeline {
		version = pb.Version
		//}
		//save the pointer of the parent pipeline_build for trigger struct
		parentPipelineBuild = &pb
	}

	envDest, err := loadDestEnvFromRunRequest(db, c, request, projectKey)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("runPipelineHandler> Cannot start tx: %s", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	// Schedule pipeline for build
	log.Info("runPipelineHandler> Scheduling %s/%s/%s[%s] with %d params, version 0",
		projectKey, app.Name, pipelineName, envDest.Name, len(request.Params))
	log.Debug("runPipelineHandler> Pipeline trigger by %s - %d", c.User.ID, request.ParentPipelineID)
	trigger := sdk.PipelineBuildTrigger{
		ManualTrigger:       true,
		TriggeredBy:         c.User,
		ParentPipelineBuild: parentPipelineBuild,
	}
	if parentPipelineBuild != nil {
		trigger.VCSChangesAuthor = parentPipelineBuild.Trigger.VCSChangesAuthor
		trigger.VCSChangesHash = parentPipelineBuild.Trigger.VCSChangesHash
		trigger.VCSChangesBranch = parentPipelineBuild.Trigger.VCSChangesBranch
	}

	pb, err := scheduler.Run(tx, projectKey, app, pipelineName, envDest.Name, request.Params, version, trigger, c.User)
	if err != nil {
		log.Warning("runPipelineHandler> Cannot run pipeline: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("runPipelineHandler> Cannot commit tx: %s", err)
		WriteError(w, r, err)
		return
	}

	k := cache.Key("application", projectKey, "builds", "*")
	cache.DeleteAll(k)

	WriteJSON(w, r, pb, http.StatusOK)
}

func runPipelineHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	var request sdk.RunRequest

	// Get args in body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	// Unmarshal args
	err = json.Unmarshal(data, &request)
	if err != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	runPipelineHandlerFunc(w, r, db, c, &request)
}

func updatePipelineActionHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	key := vars["key"]
	pipName := vars["permPipelineKey"]
	pipelineActionIDString := vars["pipelineActionID"]

	pipelineActionID, err := strconv.ParseInt(pipelineActionIDString, 10, 64)
	if err != nil {
		log.Warning("updatePipelineActionHandler>ID is not a int: %s\n", err)
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}

	var pipelineAction sdk.Action

	// Get args in body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("updatePipelineActionHandler>Cannot read body: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = json.Unmarshal(data, &pipelineAction)
	if err != nil {
		log.Warning("updatePipelineActionHandler>Cannot unmarshal request: %s\n", err)
		WriteError(w, r, err)
		return
	}

	if pipelineActionID != pipelineAction.PipelineActionID {
		log.Warning("updatePipelineActionHandler>Pipeline action does not match: %s\n", err)
		WriteError(w, r, err)
		return
	}

	args, err := json.Marshal(pipelineAction.Parameters)
	if err != nil {
		log.Warning("updatePipelineActionHandler>Cannot marshal parameters: %s\n", err)
		WriteError(w, r, err)
		return
	}

	pipelineData, err := pipeline.LoadPipeline(db, key, pipName, false)
	if err != nil {
		log.Warning("updatePipelineActionHandler>Cannot load pipeline %s: %s\n", pipName, err)
		WriteError(w, r, err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updatePipelineActionHandler> Cannot start transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	err = pipeline.UpdatePipelineAction(tx, pipelineAction, string(args))
	if err != nil {
		log.Warning("updatePipelineActionHandler> Cannot update in database: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = pipeline.UpdatePipelineLastModified(tx, pipelineData.ID)
	if err != nil {
		log.Warning("updatePipelineActionHandler> Cannot update pipeline last_modified: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("updatePipelineActionHandler> Cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	k := cache.Key("application", key, "*")
	cache.DeleteAll(k)

	w.WriteHeader(http.StatusOK)
}

func deletePipelineActionHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	pipName := vars["permPipelineKey"]
	pipelineActionIDString := vars["pipelineActionID"]

	pipelineActionID, err := strconv.ParseInt(pipelineActionIDString, 10, 64)
	if err != nil {
		log.Warning("deletepipelineActionHandler>ID is not a int: %s\n", err)
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}

	pipelineData, err := pipeline.LoadPipeline(db, key, pipName, false)
	if err != nil {
		log.Warning("deletepipelineActionHandler>Cannot load pipeline %s: %s\n", pipName, err)
		WriteError(w, r, err)
		return
	}

	log.Notice("deletePipelineActionHandler> Deleting action %d in %s/%s\n", pipelineActionID, vars["key"], vars["permPipelineKey"])

	// Select all pipeline build where given pipelineAction has been run
	query := `SELECT pipeline_build.id FROM pipeline_build
						JOIN action_build ON action_build.pipeline_build_id = pipeline_build.id
						WHERE action_build.pipeline_action_id = $1`
	var ids []int64
	rows, err := db.Query(query, pipelineActionID)
	if err != nil {
		log.Warning("deletePipelineActionHandler> cannot retrieves pipeline build: %s\n", err)
		WriteError(w, r, err)
		return
	}

	for rows.Next() {
		var id int64
		err = rows.Scan(&id)
		if err != nil {
			rows.Close()
			log.Warning("deletePipelineActionHandler> cannot retrieves pipeline build: %s\n", err)
			WriteError(w, r, err)
			return
		}
		ids = append(ids, id)
	}
	rows.Close()
	log.Notice("deletePipelineActionHandler> Got %d PipelineBuild to archive\n", len(ids))

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deletePipelineActionHandler> Cannot begin transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	// For each pipeline build, archive it to get out of relationnal
	for _, id := range ids {
		err = archivist.ArchiveBuild(tx, id)
		if err != nil {
			log.Warning("deletePipelineActionHandler> cannot archive pipeline build: %s\n", err)
			WriteError(w, r, err)
			return
		}
	}

	err = pipeline.DeletePipelineAction(tx, pipelineActionID)
	if err != nil {
		log.Warning("deletePipelineActionHandler> Cannot delete pipeline action: %s", err)
		WriteError(w, r, err)
		return
	}

	err = pipeline.UpdatePipelineLastModified(tx, pipelineData.ID)
	if err != nil {
		log.Warning("deletePipelineActionHandler> Cannot update pipeline last_modified: %s", err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("deletePipelineActionHandler> Cannot commit transaction: %s", err)
		WriteError(w, r, err)
		return
	}

	k := cache.Key("application", key, "*")
	cache.DeleteAll(k)

	w.WriteHeader(http.StatusOK)
}

func updatePipelineHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	name := vars["permPipelineKey"]

	var p sdk.Pipeline
	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("updatePipelineHandler: Cannot read body: %s", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	err = json.Unmarshal(data, &p)
	if err != nil {
		log.Warning("updatePipelineHandler: Cannot unmarshal body: %s", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	// check pipeline name pattern
	regexp := regexp.MustCompile(sdk.NamePattern)
	if !regexp.MatchString(p.Name) {
		log.Warning("updatePipelineHandler: Pipeline name %s do not respect pattern %s", p.Name, sdk.NamePattern)
		WriteError(w, r, sdk.ErrInvalidPipelinePattern)
		return
	}

	pipelineDB, err := pipeline.LoadPipeline(db, key, name, false)
	if err != nil {
		log.Warning("updatePipelineHandler> cannot load pipeline %s: %s\n", name, err)
		WriteError(w, r, err)
		return
	}

	pipelineDB.Name = p.Name
	pipelineDB.Type = p.Type

	err = pipeline.UpdatePipeline(db, pipelineDB)
	if err != nil {
		log.Warning("updatePipelineHandler> cannot update pipeline %s: %s\n", name, err)
		WriteError(w, r, err)
		return
	}

	cache.DeleteAll(cache.Key("application", key, "*"))
	cache.Delete(cache.Key("pipeline", key, name))

	w.WriteHeader(http.StatusOK)
}

func getApplicationUsingPipelineHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	name := vars["permPipelineKey"]

	pipelineData, err := pipeline.LoadPipeline(db, key, name, false)
	if err != nil {
		log.Warning("getApplicationUsingPipelineHandler> Cannot load pipeline %s: %s\n", name, err)
		WriteError(w, r, err)
		return
	}
	applications, err := application.LoadApplicationByPipeline(db, pipelineData.ID)
	if err != nil {
		log.Warning("getApplicationUsingPipelineHandler> Cannot load applications using pipeline %s: %s\n", name, err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, applications, http.StatusOK)
}

func addPipeline(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	project, err := project.LoadProject(db, key, c.User)
	if err != nil {
		log.Warning("AddPipeline: Cannot load %s: %s\n", key, err)
		WriteError(w, r, sdk.ErrNoProject)
		return
	}

	var p sdk.Pipeline
	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	err = json.Unmarshal(data, &p)
	if err != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	// check pipeline name pattern
	regexp := regexp.MustCompile(sdk.NamePattern)
	if !regexp.MatchString(p.Name) {
		log.Warning("AddPipeline: Pipeline name %s do not respect pattern %s", p.Name, sdk.NamePattern)
		WriteError(w, r, sdk.ErrInvalidPipelinePattern)
		return
	}

	// Check that pipeline does not already exists
	exist, err := pipeline.ExistPipeline(db, project.ID, p.Name)
	if err != nil {
		log.Warning("addPipeline> cannot check if pipeline exist: %s\n", err)
		WriteError(w, r, err)
		return
	}
	if exist {
		log.Warning("addPipeline> Pipeline %s already exists\n", p.Name)
		WriteError(w, r, sdk.ErrConflict)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("addPipelineHandler> Cannot start transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	p.ProjectID = project.ID
	err = pipeline.InsertPipeline(tx, &p)
	if err != nil {
		log.Warning("addPipelineHandler> Cannot insert pipeline: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = group.LoadGroupByProject(tx, project)
	if err != nil {
		log.Warning("addPipelineHandler> Cannot load groupfrom project: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = group.InsertGroupsInPipeline(tx, project.ProjectGroups, p.ID)
	if err != nil {
		log.Warning("addPipelineHandler> Cannot add groups on pipeline: %s\n", err)
		WriteError(w, r, err)
		return
	}

	for _, param := range p.Parameter {
		err = pipeline.InsertParameterInPipeline(tx, p.ID, &param)
		if err != nil {
			log.Warning("addPipelineHandler> Cannot add parameter %s: %s\n", param.Name, err)
			WriteError(w, r, err)
			return
		}
	}

	for _, app := range p.AttachedApplication {
		err = application.AttachPipeline(tx, app.ID, p.ID)
		if err != nil {
			log.Warning("addPipelineHandler> Cannot attach pipeline %d to %d: %s\n", app.ID, p.ID, err)
			WriteError(w, r, err)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("addPipelineHandler> Cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	k := cache.Key("application", key, "*")
	cache.DeleteAll(k)

	w.WriteHeader(http.StatusOK)
}

func getPipelineHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]

	p, err := pipeline.LoadPipeline(db, projectKey, pipelineName, true)
	if err != nil {
		log.Warning("getPipelineHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
		WriteError(w, r, err)
		return
	}

	p.Permission = permission.PipelinePermission(p.ID, c.User)

	WriteJSON(w, r, p, http.StatusOK)
}

func getPipelineTypeHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	WriteJSON(w, r, sdk.AvailablePipelineType, http.StatusOK)
}

func getPipelinesHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	project, err := project.LoadProject(db, key, c.User)
	if err != nil {
		if err != sdk.ErrNoProject {
			log.Warning("getPipelinesHandler: Cannot load %s: %s\n", key, err)
		}
		WriteError(w, r, err)
		return
	}

	pip, err := pipeline.LoadPipelines(db, project.ID, true, c.User)
	if err != nil {
		if err != sdk.ErrPipelineNotFound {
			log.Warning("getPipelinesHandler>Cannot load pipelines: %s\n", err)
		}
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, pip, http.StatusOK)
}

func getPipelineHistoryHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	appName := vars["permApplicationName"]

	err := r.ParseForm()
	if err != nil {
		log.Warning("getPipelineHistoryHandler> Cannot parse form: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}
	envName := r.Form.Get("envName")
	limitString := r.Form.Get("limit")
	status := r.Form.Get("status")
	stage := r.Form.Get("stage")
	branchName := r.Form.Get("branchName")

	var limit int
	if limitString != "" {
		limit, err = strconv.Atoi(limitString)
		if err != nil {
			WriteError(w, r, err)
			return
		}
	} else {
		limit = 20
	}

	p, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		if err != sdk.ErrPipelineNotFound {
			log.Warning("getPipelineHistoryHandler> Cannot load pipelines: %s\n", err)
		}
		WriteError(w, r, err)
		return
	}

	a, err := application.LoadApplicationByName(db, projectKey, appName)
	if err != nil {
		if err != sdk.ErrApplicationNotFound {
			log.Warning("getPipelineHistoryHandler> Cannot load application %s: %s\n", appName, err)
		}
		WriteError(w, r, err)
		return
	}

	var env *sdk.Environment
	if envName == "" || envName == sdk.DefaultEnv.Name {
		env = &sdk.DefaultEnv
	} else {
		env, err = environment.LoadEnvironmentByName(db, projectKey, envName)
		if err != nil {
			if err != sdk.ErrNoEnvironment {
				log.Warning("getPipelineHistoryHandler> Cannot load environment %s: %s\n", envName, err)
			}
			WriteError(w, r, err)
			return
		}
	}

	if env.ID != sdk.DefaultEnv.ID && !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionRead) {
		log.Warning("getPipelineHistoryHandler> No enought right on this environment %s: \n", envName)
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	args := []pipeline.FuncArg{}
	if stage == "true" {
		args = append(args, pipeline.WithStages())
	}

	pbs, err := pipeline.LoadPipelineBuildHistoryByApplicationAndPipeline(db, a.ID, p.ID, env.ID, limit, status, branchName, args...)
	if err != nil {
		log.Warning("getPipelineHistoryHandler> cannot load pipeline %s history: %s\n", p.Name, err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, pbs, http.StatusOK)
}

func deletePipeline(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]

	p, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		if err != sdk.ErrPipelineNotFound {
			log.Warning("deletePipeline> Cannot load pipeline %s: %s\n", pipelineName, err)
		}
		WriteError(w, r, err)
		return
	}

	used, err := application.CountPipeline(db, p.ID)
	if err != nil {
		log.Warning("deletePipeline> Cannot check if pipeline is used by an application: %s\n", err)
		WriteError(w, r, err)
		return
	}

	if used {
		log.Warning("deletePipeline> Cannot delete a pipeline used by at least 1 application\n")
		WriteError(w, r, sdk.ErrPipelineHasApplication)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deletePipeline> Cannot begin transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	err = pipeline.DeletePipeline(tx, p.ID, c.User.ID)
	if err != nil {
		log.Warning("deletePipeline> Cannot delete pipeline %s: %s\n", pipelineName, err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("deletePipeline> Cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	cache.DeleteAll(cache.Key("application", projectKey, "*"))
	cache.Delete(cache.Key("pipeline", projectKey, pipelineName))

	log.Notice("Pipeline %s removed.\n", pipelineName)
	w.WriteHeader(http.StatusOK)
}

func addJoinedActionToPipelineHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	stageIDString := vars["stageID"]

	stageID, err := strconv.ParseInt(stageIDString, 10, 60)
	if err != nil {
		log.Warning("addJoinedActionToPipelineHandler> Stage ID must be an int: %s\n", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	// Get args in body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	a, err := sdk.NewAction("").FromJSON(data)
	if err != nil {
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	proj, err := project.LoadProject(db, projectKey, c.User)
	if err != nil {
		log.Warning("addJoinedActionToPipelineHandler> Cannot load project %s: %s\n", projectKey, err)
		WriteError(w, r, sdk.ErrNoProject)
		return
	}

	pip, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		log.Warning("addJoinedActionToPipelineHandler> Cannot load pipeline %s for project %s: %s\n", pipelineName, projectKey, err)
		WriteError(w, r, sdk.ErrPipelineNotFound)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	// Insert joined action
	a.Type = sdk.JoinedAction
	a.Enabled = true
	err = action.InsertAction(tx, a, false)
	if err != nil {
		log.Warning("addJoinedActionToPipelineHandler> Cannot insert action: %s\n", err)
		WriteError(w, r, err)
		return
	}

	// then attach it to pipeline
	pipelineActionID, err := pipeline.InsertPipelineAction(tx, projectKey, pipelineName, a.ID, "[]", stageID)
	if err != nil {
		log.Warning("addActionToPipelineHandler> Cannot insert in database: %s\n", err)
		WriteError(w, r, err)
		return
	}
	a.PipelineActionID = pipelineActionID

	warnings, err := sanity.CheckAction(tx, proj, pip, a.ID)
	if err != nil {
		log.Warning("addActionToPipelineHandler> Cannot check action %d requirements: %s\n", a.ID, err)
		WriteError(w, r, err)
		return
	}

	err = sanity.InsertActionWarnings(tx, proj.ID, pip.ID, a.ID, warnings)
	if err != nil {
		log.Warning("addActionToPipelineHandler> Cannot insert warning for action %d: %s\n", a.ID, err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		WriteError(w, r, err)
		return
	}

	cache.DeleteAll(cache.Key("application", projectKey, "*"))
	cache.Delete(cache.Key("pipeline", projectKey, pipelineName))

	WriteJSON(w, r, a, http.StatusOK)
}

func updateJoinedAction(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	actionIDString := vars["actionID"]
	key := vars["key"]
	pipName := vars["permPipelineKey"]

	proj, err := project.LoadProject(db, key, c.User)
	if err != nil {
		log.Warning("updateJoinedAction> Cannot load project %s: %s\n", key, err)
		WriteError(w, r, sdk.ErrNoProject)
		return
	}

	pip, err := pipeline.LoadPipeline(db, key, pipName, false)
	if err != nil {
		log.Warning("updateJoinedAction> Cannot load pipeline %s for project %s: %s\n", pipName, key, err)
		WriteError(w, r, sdk.ErrPipelineNotFound)
		return
	}

	actionID, err := strconv.ParseInt(actionIDString, 10, 60)
	if err != nil {
		log.Warning("updateJoinedAction> Action ID must be an int: %s\n", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	// Get args in body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("updateJoinedAction> Unable to parse payload: %s", err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	a, err := sdk.NewAction("").FromJSON(data)
	if err != nil {
		log.Warning("updateJoinedAction> Unable to parse json %s: %s\n", actionIDString, err)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}
	a.ID = actionID

	clearJoinedAction, err := action.LoadActionByID(db, actionID)
	if err != nil {
		log.Warning("updateJoinedAction> Cannot load action %d: %s\n", actionID, err)
		WriteError(w, r, err)
		return
	}

	if clearJoinedAction.Type != sdk.JoinedAction {
		log.Warning("updateJoinedAction> Tried to update a %s action, aborting", clearJoinedAction.Type)
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateJoinedAction> Cannot begin tx: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	err = action.UpdateActionDB(tx, a, c.User.ID)
	if err != nil {
		log.Warning("updateJoinedAction> cannot update action: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = pipeline.UpdatePipelineLastModified(tx, pip.ID)
	if err != nil {
		log.Warning("updateJoinedAction> cannot update pipeline last_modified date: %s\n", err)
		WriteError(w, r, err)
		return
	}

	warnings, err := sanity.CheckAction(tx, proj, pip, a.ID)
	if err != nil {
		log.Warning("updateJoinedAction> Cannot check action %d requirements: %s\n", a.ID, err)
		WriteError(w, r, err)
		return
	}

	err = sanity.InsertActionWarnings(tx, proj.ID, pip.ID, a.ID, warnings)
	if err != nil {
		log.Warning("updateJoinedAction> Cannot insert warning for action %d: %s\n", a.ID, err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("updateJoinedAction> Cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	cache.DeleteAll(cache.Key("application", projectKey, "*"))
	cache.Delete(cache.Key("pipeline", projectKey, pip.Name))

	WriteJSON(w, r, a, http.StatusOK)
}

func deleteJoinedAction(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipName := vars["permPipelineKey"]
	actionIDString := vars["actionID"]

	actionID, err := strconv.ParseInt(actionIDString, 10, 60)
	if err != nil {
		log.Warning("deleteJoinedAction> Action ID must be an int: %s\n", err)
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}

	pip, err := pipeline.LoadPipeline(db, projectKey, pipName, false)
	if err != nil {
		log.Warning("deleteJoinedAction> Cannot load pipeline %s for project %s: %s\n", pipName, projectKey, err)
		WriteError(w, r, sdk.ErrPipelineNotFound)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteJoinedAction> Cannot start transaction: %s", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	err = action.DeleteAction(db, actionID, c.User.ID)
	if err != nil {
		log.Warning("deleteJoinedAction> Cannot delete joined action: %s\n", err)
		WriteError(w, r, err)
		return
	}

	err = pipeline.UpdatePipelineLastModified(tx, pip.ID)
	if err != nil {
		log.Warning("deleteJoinedAction> Cannot update last_modified pipeline date: %s", err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("deleteJoinedAction> Cannot commit transation: %s", err)
		WriteError(w, r, err)
		return
	}

	k := cache.Key("application", projectKey, "*")
	cache.DeleteAll(k)

}

func getJoinedActionAudithandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	actionIDString := vars["actionID"]

	actionID, err := strconv.Atoi(actionIDString)
	if err != nil {
		log.Warning("getJoinedActionAudithandler> Action ID must be an int: %s\n", err)
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}

	audit, err := action.LoadAuditAction(db, actionID, false)
	if err != nil {
		log.Warning("getJoinedActionAudithandler> Cannot load audit for action %d: %s\n", actionID, err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, audit, http.StatusOK)
}

func getJoinedAction(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	actionIDString := vars["actionID"]
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]

	actionID, err := strconv.ParseInt(actionIDString, 10, 60)
	if err != nil {
		log.Warning("getJoinedAction> Action ID must be an int: %s\n", err)
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}

	a, err := action.LoadPipelineActionByID(db, projectKey, pipelineName, actionID)
	if err != nil {
		log.Warning("getJoinedAction> Cannot load joined action: %s\n", err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, a, http.StatusOK)
}

func getBuildingPipelines(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	var err error
	var pbs, recent []sdk.PipelineBuild

	if c.User.Admin {
	} else {
		pbs, err = pipeline.LoadUserBuildingPipelines(db, c.User.ID)
	}
	if err != nil {
		log.Warning("getBuildingPipelines> cannot load Building pipelines: %s", err)
		WriteError(w, r, err)
		return
	}

	if c.User.Admin {
		recent, err = pipeline.LoadRecentPipelineBuild(db)
	} else {
		recent, err = pipeline.LoadUserRecentPipelineBuild(db, c.User.ID)
	}
	if err != nil {
		log.Warning("getBuildingPipelines> cannot load recent pipelines: %s", err)
		WriteError(w, r, err)
		return
	}
	pbs = append(pbs, recent...)
	WriteJSON(w, r, pbs, http.StatusOK)
}

func getPipelineBuildingCommit(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	hash := vars["hash"]

	pbs, err := pipeline.LoadPipelineBuildByHash(db, hash)
	if err != nil {
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, pbs, http.StatusOK)
}

func stopPipelineBuildHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]
	pipName := vars["permPipelineKey"]
	buildNumberS := vars["build"]

	err := r.ParseForm()
	if err != nil {
		log.Warning("stopPipelineBuildHandler> Cannot parse form: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}
	envName := r.Form.Get("envName")

	buildNumber, err := strconv.ParseInt(buildNumberS, 10, 64)
	if err != nil {
		log.Warning("stopPipelineBuildHandler> buildNumber is not a int: %s\n", err)
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}

	// Load pipeline
	pip, err := pipeline.LoadPipeline(db, projectKey, pipName, false)
	if err != nil {
		log.Warning("stopPipelineBuildHandler> Cannot load pipeline: %s\n", err)
		WriteError(w, r, err)
		return
	}

	// Load application
	app, err := application.LoadApplicationByName(db, projectKey, appName)
	if err != nil {
		log.Warning("stopPipelineBuildHandler> Cannot load application: %s\n", err)
		WriteError(w, r, err)
		return
	}

	// Load environment
	if pip.Type != sdk.BuildPipeline && (envName == "" || envName == sdk.DefaultEnv.Name) {
		WriteError(w, r, sdk.ErrNoEnvironmentProvided)
		return
	}
	env := &sdk.DefaultEnv

	if pip.Type != sdk.BuildPipeline {
		env, err = environment.LoadEnvironmentByName(db, projectKey, envName)
		if err != nil {
			log.Warning("stopPipelineBuildHandler> Cannot load environment %s: %s\n", envName, err)
			WriteError(w, r, err)
			return
		}

		if env.ID != sdk.DefaultEnv.ID && !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadExecute) {
			log.Warning("stopPipelineBuildHandler> No enought right on this environment %s: \n", env.Name)
			WriteError(w, r, sdk.ErrForbidden)
			return
		}
	}

	if env.ID != sdk.DefaultEnv.ID && !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadExecute) {
		log.Warning("stopPipelineBuildHandler> You do not have Execution Right on this environment %s\n", env.Name)
		WriteError(w, r, sdk.ErrNoEnvExecution)
		return
	}

	pb, err := pipeline.LoadPipelineBuild(db, pip.ID, app.ID, buildNumber, env.ID)
	if err != nil {
		errFinal := err
		if err == sdk.ErrNoPipelineBuild {
			errFinal = sdk.ErrBuildArchived
		}
		log.Warning("stopPipelineBuildHandler> Cannot load pipeline Build: %s\n", errFinal)
		WriteError(w, r, errFinal)
		return
	}

	err = pipeline.StopPipelineBuild(db, pb.ID)
	if err != nil {
		log.Warning("stopPipelineBuildHandler> Cannot stop pb: %s\n", err)
		WriteError(w, r, err)
		return
	}

	k := cache.Key("application", projectKey, "builds", "*")
	cache.DeleteAll(k)
}

func restartPipelineBuildHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]
	pipName := vars["permPipelineKey"]
	buildNumberS := vars["build"]

	err := r.ParseForm()
	if err != nil {
		log.Warning("restartPipelineBuildHandler> Cannot parse form: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}
	envName := r.Form.Get("envName")

	buildNumber, err := strconv.ParseInt(buildNumberS, 10, 64)
	if err != nil {
		log.Warning("restartPipelineBuildHandler> buildNumber is not a int: %s\n", err)
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}

	// Load pipeline
	pip, err := pipeline.LoadPipeline(db, projectKey, pipName, false)
	if err != nil {
		log.Warning("restartPipelineBuildHandler> Cannot load pipeline: %s\n", err)
		WriteError(w, r, err)
		return
	}

	// Load application
	app, err := application.LoadApplicationByName(db, projectKey, appName)
	if err != nil {
		log.Warning("restartPipelineBuildHandler> Cannot load application: %s\n", err)
		WriteError(w, r, err)
		return
	}

	// Load environment
	if pip.Type != sdk.BuildPipeline && (envName == "" || envName == sdk.DefaultEnv.Name) {
		WriteError(w, r, sdk.ErrNoEnvironmentProvided)
		return
	}
	env := &sdk.DefaultEnv

	if pip.Type != sdk.BuildPipeline {
		env, err = environment.LoadEnvironmentByName(db, projectKey, envName)
		if err != nil {
			log.Warning("restartPipelineBuildHandler> Cannot load environment %s: %s\n", envName, err)
			WriteError(w, r, err)
			return
		}

		if env.ID != sdk.DefaultEnv.ID && !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadExecute) {
			log.Warning("restartPipelineBuildHandler> No enought right on this environment %s: \n", envName)
			WriteError(w, r, sdk.ErrForbidden)
			return
		}
	}

	pb, err := pipeline.LoadPipelineBuild(db, pip.ID, app.ID, buildNumber, env.ID, pipeline.WithParameters())
	if err != nil {
		errFinal := err
		if err == sdk.ErrNoPipelineBuild {
			errFinal = sdk.ErrBuildArchived
		}
		log.Warning("restartPipelineBuildHandler> Cannot load pipeline Build: %s\n", errFinal)
		WriteError(w, r, errFinal)
		return
	}

	if env.ID != sdk.DefaultEnv.ID && !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadExecute) {
		log.Warning("restartPipelineBuildHandler> You do not have Execution Right on this environment %s\n", env.Name)
		WriteError(w, r, sdk.ErrNoEnvExecution)
		return
	}

	err = pipeline.RestartPipelineBuild(db, pb)
	if err != nil {
		log.Warning("restartPipelineBuildHandler> cannot restart pb: %s\n", err)
		WriteError(w, r, err)
		return
	}

	k := cache.Key("application", projectKey, "builds", "*")
	cache.DeleteAll(k)

	WriteJSON(w, r, pb, http.StatusOK)
}

func getPipelineCommitsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]
	pipName := vars["permPipelineKey"]

	if err := r.ParseForm(); err != nil {
		log.Warning("getPipelineCommitsHandler> Cannot parse form: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}
	envName := r.Form.Get("envName")
	hash := r.Form.Get("hash")

	// Load pipeline
	pip, err := pipeline.LoadPipeline(db, projectKey, pipName, false)
	if err != nil {
		log.Warning("getPipelineCommitsHandler> Cannot load pipeline: %s\n", err)
		WriteError(w, r, err)
		return
	}

	//Load the environment
	var env *sdk.Environment
	if envName == "" || envName == sdk.DefaultEnv.Name {
		env = &sdk.DefaultEnv
	} else {
		env, err = environment.LoadEnvironmentByName(db, projectKey, envName)
		if err != nil {
			log.Warning("getPipelineCommitsHandler> Cannot load environment %s: %s\n", envName, err)
			WriteError(w, r, err)
			return
		}
	}

	if env.ID != sdk.DefaultEnv.ID && !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionRead) {
		log.Warning("getPipelineCommitsHandler> No enought right on this environment %s: \n", envName)
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	//Load the application
	application, err := application.LoadApplicationByName(db, projectKey, appName)
	if err != nil {
		WriteError(w, r, sdk.ErrApplicationNotFound)
		return
	}

	commits := []sdk.VCSCommit{}

	//Check it the application is attached to a repository
	if application.RepositoriesManager == nil {
		log.Warning("getPipelineCommitsHandler> Project not attached to a repository managers %s: \n", envName)
		WriteJSON(w, r, commits, http.StatusOK)
		return
	}

	pbs, err := pipeline.LoadPipelineBuildHistoryByApplicationAndPipeline(db, application.ID, pip.ID, env.ID, 1, string(sdk.StatusSuccess), "")
	if err != nil {
		log.Warning("getPipelineCommitsHandler> Cannot load pipeline build %s: \n", err)
		WriteError(w, r, err)
		return
	}

	if len(pbs) != 1 {
		log.Warning("getPipelineCommitsHandler> There is no previous build")
		WriteJSON(w, r, commits, http.StatusOK)
		return
	}

	b, e := repositoriesmanager.CheckApplicationIsAttached(db, application.RepositoriesManager.Name, projectKey, appName)
	if e != nil {
		log.Warning("getPipelineCommitsHandler> Cannot check app (%s,%s,%s): %s", application.RepositoriesManager.Name, projectKey, appName, e)
		WriteError(w, r, e)
		return
	}

	if !b && application.RepositoryFullname == "" {
		log.Warning("getPipelineCommitsHandler> No repository on the application %s", appName)
		WriteJSON(w, r, commits, http.StatusOK)
		return
	}

	//Get the RepositoriesManager Client
	client, err := repositoriesmanager.AuthorizedClient(db, projectKey, application.RepositoriesManager.Name)
	if err != nil {
		log.Warning("getPipelineCommitsHandler> Cannot get client: %s", err)
		WriteError(w, r, sdk.ErrNoReposManagerClientAuth)
		return
	}

	if pbs[0].Trigger.VCSChangesHash == "" {
		log.Warning("getPipelineCommitsHandler>No hash on the previous run %d", pbs[0].ID)
		WriteJSON(w, r, commits, http.StatusOK)
		return
	}

	//If we are lucky, return a true diff
	commits, err = client.Commits(application.RepositoryFullname, pbs[0].Trigger.VCSChangesHash, hash)
	if err != nil {
		log.Warning("getPipelineBuildCommitsHandler> Cannot get commits: %s", err)
		WriteError(w, r, err)
		return
	}
	WriteJSON(w, r, commits, http.StatusOK)

}

func getPipelineBuildCommitsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]
	pipName := vars["permPipelineKey"]
	buildNumber, err := strconv.Atoi(vars["build"])
	if err != nil {
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Warning("getPipelineBuildCommitsHandler> Cannot parse form: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}
	envName := r.Form.Get("envName")

	// Load pipeline
	pip, err := pipeline.LoadPipeline(db, projectKey, pipName, false)
	if err != nil {
		log.Warning("getPipelineBuildCommitsHandler> Cannot load pipeline: %s\n", err)
		WriteError(w, r, err)
		return
	}

	//Load the environment
	var env *sdk.Environment
	if envName == "" || envName == sdk.DefaultEnv.Name {
		env = &sdk.DefaultEnv
	} else {
		env, err = environment.LoadEnvironmentByName(db, projectKey, envName)
		if err != nil {
			if err != sdk.ErrNoEnvironment {
				log.Warning("getPipelineBuildCommitsHandler> Cannot load environment %s: %s\n", envName, err)
			}
			WriteError(w, r, err)
			return
		}
	}

	if env.ID != sdk.DefaultEnv.ID && !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionRead) {
		log.Warning("getPipelineHistoryHandler> No enought right on this environment %s: \n", envName)
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	//Load the application
	application, err := application.LoadApplicationByName(db, projectKey, appName)
	if err != nil {
		WriteError(w, r, sdk.ErrApplicationNotFound)
		return
	}

	//Check it the application is attached to a repository
	if application.RepositoriesManager == nil {
		WriteError(w, r, sdk.ErrNoReposManagerClientAuth)
		return
	}

	b, e := repositoriesmanager.CheckApplicationIsAttached(db, application.RepositoriesManager.Name, projectKey, appName)
	if e != nil {
		log.Warning("getPipelineBuildCommitsHandler> Cannot check app (%s,%s,%s): %s", application.RepositoriesManager.Name, projectKey, appName, e)
		WriteError(w, r, e)
		return
	}

	if !b && application.RepositoryFullname == "" {
		WriteError(w, r, sdk.ErrNoReposManagerClientAuth)
		return
	}

	//Get the RepositoriesManager Client
	client, err := repositoriesmanager.AuthorizedClient(db, projectKey, application.RepositoriesManager.Name)
	if err != nil {
		log.Warning("getPipelineBuildCommitsHandler> Cannot get client: %s", err)
		WriteError(w, r, sdk.ErrNoReposManagerClientAuth)
		return
	}

	//Get the commit hash for the pipeline build number and the hash for the previous pipeline build for the same branch
	//buildNumber, pipelineID, applicationID, environmentID
	cur, prev, err := pipeline.CurrentAndPreviousPipelineBuildNumberAndHash(db, int64(buildNumber), pip.ID, application.ID, env.ID)
	if err != nil {
		log.Warning("getPipelineBuildCommitsHandler> Cannot get build number and hashes (buildNumber=%d, pipelineID=%d, applicationID=%d, envID=%d)  : %s ", buildNumber, pip.ID, application.ID, env.ID, err)
		WriteError(w, r, err)
		return
	}

	if prev != nil && cur.Hash != "" && prev.Hash != "" {
		//If we are lucky, return a true diff
		commits, err := client.Commits(application.RepositoryFullname, prev.Hash, cur.Hash)
		if err != nil {
			log.Warning("getPipelineBuildCommitsHandler> Cannot get commits: %s", err)
			WriteError(w, r, err)
			return
		}
		WriteJSON(w, r, commits, http.StatusOK)
		return
	}

	if cur.Hash != "" {
		//If we only get current pipeline build hash
		c, err := client.Commit(application.RepositoryFullname, cur.Hash)
		if err != nil {
			log.Warning("getPipelineBuildCommitsHandler> Cannot get commits: %s", err)
			WriteError(w, r, err)
			return
		}
		WriteJSON(w, r, []sdk.VCSCommit{c}, http.StatusOK)
	} else {
		//If we only have the current branch, search for the branch
		br, err := client.Branch(application.RepositoryFullname, cur.Branch)
		if err != nil {
			log.Warning("getPipelineBuildCommitsHandler> Cannot get branch: %s", err)
			WriteError(w, r, err)
			return
		}
		if br.LatestCommit == "" {
			log.Warning("getPipelineBuildCommitsHandler> Branch or lastest commit not found")
			WriteError(w, r, sdk.ErrNoBranch)
			return
		}
		//and return the last commit of the branch
		log.Debug("get the last commit : %s", br.LatestCommit)
		c, err := client.Commit(application.RepositoryFullname, br.LatestCommit)
		if err != nil {
			log.Warning("getPipelineBuildCommitsHandler> Cannot get commits: %s", err)
			WriteError(w, r, err)
			return
		}
		WriteJSON(w, r, []sdk.VCSCommit{c}, http.StatusOK)
	}
	return
}
