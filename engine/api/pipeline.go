package main

import (
	"net/http"
	"regexp"
	"strconv"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/queue"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/sanity"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func rollbackPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	appName := vars["permApplicationName"]

	var request sdk.RunRequest
	if err := UnmarshalBody(r, &request); err != nil {
		return err
	}

	// Load application
	app, err := application.LoadByName(db, projectKey, appName, c.User, application.LoadOptions.WithRepositoryManager, application.LoadOptions.WithTriggers, application.LoadOptions.WithVariablesWithClearPassword)
	if err != nil {
		if err != sdk.ErrApplicationNotFound {
			log.Warning("rollbackPipelineHandler> Cannot load application %s: %s\n", appName, err)
		}
		return err
	}

	// Load pipeline
	pip, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		if err != sdk.ErrPipelineNotFound {
			log.Warning("rollbackPipelineHandler> Cannot load pipeline %s; %s\n", pipelineName, err)
		}
		return err
	}

	// Load Env
	var env *sdk.Environment
	if request.Env.Name != "" && request.Env.Name != sdk.DefaultEnv.Name {
		env, err = environment.LoadEnvironmentByName(db, projectKey, request.Env.Name)
		if err != nil {
			log.Warning("rollbackPipelineHandler> Cannot load environment %s; %s\n", request.Env.Name, err)
			return sdk.ErrNoEnvironment
		}

		if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadExecute) {
			log.Warning("rollbackPipelineHandler> No enought right on this environment %s: \n", request.Env.Name)
			return sdk.ErrForbidden
		}

	} else {
		env = &sdk.DefaultEnv
	}

	if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadExecute) {
		log.Warning("rollbackPipelineHandler> You do not have Execution Right on this environment %s\n", env.Name)
		return sdk.ErrNoEnvExecution
	}

	pbs, err := pipeline.LoadPipelineBuildsByApplicationAndPipeline(db, app.ID, pip.ID, env.ID, 2, string(sdk.StatusSuccess), "")
	if err != nil {
		log.Warning("rollbackPipelineHandler> Cannot load pipeline build history %s", err)
		return sdk.ErrNoPipelineBuild
	}

	if len(pbs) != 2 {
		log.Warning("rollbackPipelineHandler> There is no previous success for app %s(%d), pip %s(%d), env %s(%d): %d", app.Name, app.ID, pip.Name, pip.ID, env.Name, env.ID, len(pbs))
		return sdk.ErrNoPreviousSuccess
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("rollbackPipelineHandler> Cannot start tx: %s", err)
		return err
	}
	defer tx.Rollback()

	trigger := pbs[1].Trigger
	trigger.TriggeredBy = c.User

	newPb, err := queue.RunPipeline(tx, projectKey, app, pipelineName, env.Name, pbs[1].Parameters, pbs[1].Version, trigger, c.User)
	if err != nil {
		log.Warning("rollbackPipelineHandler> Cannot run pipeline: %s\n", err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("rollbackPipelineHandler> Cannot commit tx: %s", err)
		return err
	}

	k := cache.Key("application", projectKey, "builds", "*")
	cache.DeleteAll(k)

	return WriteJSON(w, r, newPb, http.StatusOK)
}

func loadDestEnvFromRunRequest(db *gorp.DbMap, c *context.Ctx, request *sdk.RunRequest, projectKey string) (*sdk.Environment, error) {
	var envDest = &sdk.DefaultEnv
	var err error
	if request.Env.Name != "" && request.Env.Name != sdk.DefaultEnv.Name {
		envDest, err = environment.LoadEnvironmentByName(db, projectKey, request.Env.Name)
		if err != nil {
			log.Warning("loadDestEnvFromRunRequest> Cannot load destination environment: %s", err)
			return nil, sdk.ErrNoEnvironment
		}
	}
	if !permission.AccessToEnvironment(envDest.ID, c.User, permission.PermissionReadExecute) {
		log.Warning("loadDestEnvFromRunRequest> You do not have Execution Right on this environment\n")
		return nil, sdk.ErrForbidden
	}
	return envDest, nil
}

func runPipelineWithLastParentHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	appName := vars["permApplicationName"]

	app, err := application.LoadByName(db, projectKey, appName, c.User, application.LoadOptions.WithRepositoryManager, application.LoadOptions.WithTriggers, application.LoadOptions.WithVariablesWithClearPassword)
	if err != nil {
		if err != sdk.ErrApplicationNotFound {
			log.Warning("runPipelineWithLastParentHandler> Cannot load application %s: %s\n", appName, err)
		}
		return err
	}

	var request sdk.RunRequest
	if err := UnmarshalBody(r, &request); err != nil {
		return err
	}

	//Check parent stuff
	if request.ParentApplicationID == 0 && request.ParentPipelineID == 0 {
		return sdk.ErrParentApplicationAndPipelineMandatory
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
		return err
	}

	// Check that pipeline is attached to application
	ok, err := application.IsAttached(db, app.ProjectID, app.ID, pip.Name)
	if !ok {
		log.Warning("runPipelineWithLastParentHandler> Pipeline %s is not attached to app %s\n", pipelineName, appName)
		return sdk.ErrPipelineNotAttached
	}
	if err != nil {
		log.Warning("runPipelineWithLastParentHandler> Cannot check if pipeline %s is attached to %s: %s\n", pipelineName, appName, err)
		return err
	}

	//Load environment
	envDest, err := loadDestEnvFromRunRequest(db, c, &request, projectKey)
	if err != nil {
		return err
	}

	//Load triggers
	triggers, err := trigger.LoadTriggers(db, app.ID, pip.ID, envDest.ID)
	if err != nil {
		return err
	}

	//Find parent trigger
	var trig *sdk.PipelineTrigger
	for i, t := range triggers {
		log.Debug("Trigger from app(%s[%d]) pip(%s[%d]) env(%s[%d]) to app(%s[%d]) pip(%s[%d]) env(%s[%d])\n", t.SrcApplication.Name, t.SrcApplication.ID, t.SrcPipeline.Name, t.SrcPipeline.ID, t.SrcEnvironment.Name, t.SrcEnvironment.ID, t.DestApplication.Name, t.DestApplication.ID, t.DestPipeline.Name, t.DestPipeline.ID, t.DestEnvironment.Name, t.DestEnvironment.ID)
		if t.SrcApplication.ID == request.ParentApplicationID &&
			t.SrcPipeline.ID == request.ParentPipelineID &&
			t.SrcEnvironment.ID == envID {
			trig = &triggers[i]
		}
	}

	//If trigger not found: exit
	if trig == nil {
		return sdk.ErrPipelineNotFound
	}

	//Branch
	var branch string
	for _, p := range request.Params {
		if p.Name == "git.branch" {
			branch = p.Value
		}
	}

	builds, err := pipeline.LoadPipelineBuildsByApplicationAndPipeline(db, request.ParentApplicationID, request.ParentPipelineID, envID, 1, string(sdk.StatusSuccess), branch)
	if err != nil {
		log.Warning("runPipelineWithLastParentHandler> Unable to find any successfull pipeline build")
		return sdk.ErrNoParentBuildFound
	}
	if len(builds) == 0 {
		return sdk.ErrNoPipelineBuild
	}

	request.ParentBuildNumber = builds[0].BuildNumber

	return runPipelineHandlerFunc(w, r, db, c, &request)
}

func runPipelineHandlerFunc(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx, request *sdk.RunRequest) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	appName := vars["permApplicationName"]

	app, err := application.LoadByName(db, projectKey, appName, c.User, application.LoadOptions.WithRepositoryManager, application.LoadOptions.WithTriggers, application.LoadOptions.WithVariablesWithClearPassword)
	if err != nil {
		if err != sdk.ErrApplicationNotFound {
			log.Warning("runPipelineHandler> Cannot load application %s: %s\n", appName, err)
		}
		return err
	}

	var pip *sdk.Pipeline
	for _, p := range app.Pipelines {
		if p.Pipeline.Name == pipelineName {
			pip = &p.Pipeline
			break
		}
	}

	if pip == nil {
		log.Warning("runPipelineHandler> Pipeline %s is not attached to app %s\n", pipelineName, appName)
		return sdk.ErrPipelineNotAttached
	}

	version := int64(0)
	// Load parent pipeline build + add parent variable
	var parentPipelineBuild *sdk.PipelineBuild
	envID := sdk.DefaultEnv.ID
	if request.ParentBuildNumber != 0 {
		if request.ParentEnvironmentID != 0 {
			envID = request.ParentEnvironmentID
		}
		pb, err := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(db, request.ParentApplicationID, request.ParentPipelineID, envID, request.ParentBuildNumber)
		if err != nil {
			return sdk.WrapError(err, "runPipelineHandler> Cannot load parent pipeline build")
		}
		parentParams := queue.ParentBuildInfos(pb)
		request.Params = append(request.Params, parentParams...)

		version = pb.Version
		parentPipelineBuild = pb
	}

	envDest, err := loadDestEnvFromRunRequest(db, c, request, projectKey)
	if err != nil {
		return sdk.WrapError(err, "runPipelineHandler> Unable to load dest environment")
	}

	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "runPipelineHandler> Cannot start tx")
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

	pb, err := queue.RunPipeline(tx, projectKey, app, pipelineName, envDest.Name, request.Params, version, trigger, c.User)
	if err != nil {
		return sdk.WrapError(err, "runPipelineHandler> Cannot run pipeline")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "runPipelineHandler> Cannot commit tx")
	}

	return WriteJSON(w, r, pb, http.StatusOK)
}

func runPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	var request sdk.RunRequest
	if err := UnmarshalBody(r, &request); err != nil {
		return err
	}

	return runPipelineHandlerFunc(w, r, db, c, &request)
}

// DEPRECATED
func updatePipelineActionHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	pipName := vars["permPipelineKey"]
	pipelineActionIDString := vars["pipelineActionID"]

	pipelineActionID, err := strconv.ParseInt(pipelineActionIDString, 10, 64)
	if err != nil {
		log.Warning("updatePipelineActionHandler>ID is not a int: %s\n", err)
		return sdk.ErrInvalidID
	}

	var job sdk.Job
	if err := UnmarshalBody(r, &job); err != nil {
		return err
	}

	if pipelineActionID != job.PipelineActionID {
		log.Warning("updatePipelineActionHandler>Pipeline action does not match: %s\n", err)
		return err
	}

	pipelineData, err := pipeline.LoadPipeline(db, key, pipName, false)
	if err != nil {
		log.Warning("updatePipelineActionHandler>Cannot load pipeline %s: %s\n", pipName, err)
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updatePipelineActionHandler> Cannot start transaction: %s\n", err)
		return err
	}
	defer tx.Rollback()

	err = pipeline.UpdatePipelineAction(tx, job)
	if err != nil {
		log.Warning("updatePipelineActionHandler> Cannot update in database: %s\n", err)
		return err
	}

	err = pipeline.UpdatePipelineLastModified(tx, pipelineData)
	if err != nil {
		log.Warning("updatePipelineActionHandler> Cannot update pipeline last_modified: %s\n", err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("updatePipelineActionHandler> Cannot commit transaction: %s\n", err)
		return err
	}

	k := cache.Key("application", key, "*")
	cache.DeleteAll(k)

	return nil
}

func deletePipelineActionHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	pipName := vars["permPipelineKey"]
	pipelineActionIDString := vars["pipelineActionID"]

	pipelineActionID, err := strconv.ParseInt(pipelineActionIDString, 10, 64)
	if err != nil {
		log.Warning("deletepipelineActionHandler>ID is not a int: %s\n", err)
		return sdk.ErrInvalidID
	}

	pipelineData, err := pipeline.LoadPipeline(db, key, pipName, false)
	if err != nil {
		log.Warning("deletepipelineActionHandler>Cannot load pipeline %s: %s\n", pipName, err)
		return err
	}

	log.Notice("deletePipelineActionHandler> Deleting action %d in %s/%s\n", pipelineActionID, vars["key"], vars["permPipelineKey"])

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deletePipelineActionHandler> Cannot begin transaction: %s\n", err)
		return err
	}
	defer tx.Rollback()

	err = pipeline.DeletePipelineAction(tx, pipelineActionID)
	if err != nil {
		log.Warning("deletePipelineActionHandler> Cannot delete pipeline action: %s", err)
		return err
	}

	err = pipeline.UpdatePipelineLastModified(tx, pipelineData)
	if err != nil {
		log.Warning("deletePipelineActionHandler> Cannot update pipeline last_modified: %s", err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("deletePipelineActionHandler> Cannot commit transaction: %s", err)
		return err
	}

	k := cache.Key("application", key, "*")
	cache.DeleteAll(k)

	return nil
}

func updatePipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	name := vars["permPipelineKey"]

	var p sdk.Pipeline
	if err := UnmarshalBody(r, &p); err != nil {
		return err
	}

	// check pipeline name pattern
	regexp := regexp.MustCompile(sdk.NamePattern)
	if !regexp.MatchString(p.Name) {
		log.Warning("updatePipelineHandler: Pipeline name %s do not respect pattern %s", p.Name, sdk.NamePattern)
		return sdk.ErrInvalidPipelinePattern
	}

	pipelineDB, err := pipeline.LoadPipeline(db, key, name, false)
	if err != nil {
		log.Warning("updatePipelineHandler> cannot load pipeline %s: %s\n", name, err)
		return err
	}

	pipelineDB.Name = p.Name
	pipelineDB.Type = p.Type

	err = pipeline.UpdatePipeline(db, pipelineDB)
	if err != nil {
		log.Warning("updatePipelineHandler> cannot update pipeline %s: %s\n", name, err)
		return err
	}

	cache.DeleteAll(cache.Key("application", key, "*"))
	cache.Delete(cache.Key("pipeline", key, name))

	return WriteJSON(w, r, pipelineDB, http.StatusOK)
}

func getApplicationUsingPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	name := vars["permPipelineKey"]

	pipelineData, err := pipeline.LoadPipeline(db, key, name, false)
	if err != nil {
		log.Warning("getApplicationUsingPipelineHandler> Cannot load pipeline %s: %s\n", name, err)
		return err
	}
	applications, err := application.LoadByPipeline(db, pipelineData.ID, c.User)
	if err != nil {
		log.Warning("getApplicationUsingPipelineHandler> Cannot load applications using pipeline %s: %s\n", name, err)
		return err
	}

	return WriteJSON(w, r, applications, http.StatusOK)
}

func addPipeline(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	project, err := project.Load(db, key, c.User, project.LoadOptions.Default)
	if err != nil {
		log.Warning("AddPipeline: Cannot load %s: %s\n", key, err)
		return err
	}

	var p sdk.Pipeline
	if err := UnmarshalBody(r, &p); err != nil {
		return err
	}

	// check pipeline name pattern
	if regexp := regexp.MustCompile(sdk.NamePattern); !regexp.MatchString(p.Name) {
		log.Warning("AddPipeline: Pipeline name %s do not respect pattern %s", p.Name, sdk.NamePattern)
		return sdk.ErrInvalidPipelinePattern
	}

	// Check that pipeline does not already exists
	exist, err := pipeline.ExistPipeline(db, project.ID, p.Name)
	if err != nil {
		log.Warning("addPipeline> cannot check if pipeline exist: %s\n", err)
		return err
	}
	if exist {
		log.Warning("addPipeline> Pipeline %s already exists\n", p.Name)
		return sdk.ErrConflict
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("addPipelineHandler> Cannot start transaction: %s\n", err)
		return err
	}
	defer tx.Rollback()

	p.ProjectID = project.ID
	if err := pipeline.InsertPipeline(tx, &p); err != nil {
		log.Warning("addPipelineHandler> Cannot insert pipeline: %s\n", err)
		return err
	}

	if err := group.LoadGroupByProject(tx, project); err != nil {
		log.Warning("addPipelineHandler> Cannot load groupfrom project: %s\n", err)
		return err
	}

	for _, g := range project.ProjectGroups {
		p.GroupPermission = append(p.GroupPermission, g)
	}

	if err := group.InsertGroupsInPipeline(tx, project.ProjectGroups, p.ID); err != nil {
		log.Warning("addPipelineHandler> Cannot add groups on pipeline: %s\n", err)
		return err
	}

	for _, app := range p.AttachedApplication {
		if _, err := application.AttachPipeline(tx, app.ID, p.ID); err != nil {
			log.Warning("addPipelineHandler> Cannot attach pipeline %d to %d: %s\n", app.ID, p.ID, err)
			return err
		}

		if err := application.UpdateLastModified(tx, &app, c.User); err != nil {
			log.Warning("addPipelineHandler> Cannot update application last modified date: %s\n", err)
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		log.Warning("addPipelineHandler> Cannot commit transaction: %s\n", err)
		return err
	}

	p.Permission = permission.PermissionReadWriteExecute

	return WriteJSON(w, r, p, http.StatusOK)
}

func getPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]

	p, err := pipeline.LoadPipeline(db, projectKey, pipelineName, true)
	if err != nil {
		log.Warning("getPipelineHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
		return err
	}

	p.Permission = permission.PipelinePermission(p.ID, c.User)

	return WriteJSON(w, r, p, http.StatusOK)
}

func getPipelineTypeHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	return WriteJSON(w, r, sdk.AvailablePipelineType, http.StatusOK)
}

func getPipelinesHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	project, err := project.Load(db, key, c.User, project.LoadOptions.Default)
	if err != nil {
		if err != sdk.ErrNoProject {
			log.Warning("getPipelinesHandler: Cannot load %s: %s\n", key, err)
		}
		return err
	}

	pip, err := pipeline.LoadPipelines(db, project.ID, true, c.User)
	if err != nil {
		if err != sdk.ErrPipelineNotFound {
			log.Warning("getPipelinesHandler>Cannot load pipelines: %s\n", err)
		}
		return err
	}

	return WriteJSON(w, r, pip, http.StatusOK)
}

func getPipelineHistoryHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	appName := vars["permApplicationName"]

	if err := r.ParseForm(); err != nil {
		return sdk.WrapError(err, "getPipelineHistoryHandler> Cannot parse form")
	}
	envName := r.Form.Get("envName")
	limitString := r.Form.Get("limit")
	status := r.Form.Get("status")
	branchName := r.Form.Get("branchName")

	var limit int
	if limitString != "" {
		var erra error
		limit, erra = strconv.Atoi(limitString)
		if erra != nil {
			return erra
		}
	} else {
		limit = 20
	}

	p, errlp := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if errlp != nil {
		return sdk.WrapError(errlp, "getPipelineHistoryHandler> Cannot load pipelines")
	}

	a, errln := application.LoadByName(db, projectKey, appName, c.User)
	if errln != nil {
		return sdk.WrapError(errln, "getPipelineHistoryHandler> Cannot load application %s", appName)
	}

	var env *sdk.Environment
	if envName == "" || envName == sdk.DefaultEnv.Name {
		env = &sdk.DefaultEnv
	} else {
		var errle error
		env, errle = environment.LoadEnvironmentByName(db, projectKey, envName)
		if errle != nil {
			return sdk.WrapError(errle, "getPipelineHistoryHandler> Cannot load environment %s", envName)
		}
	}

	if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionRead) {
		return sdk.WrapError(sdk.ErrForbidden, "getPipelineHistoryHandler> No enought right on this environment %s", envName)
	}

	pbs, errl := pipeline.LoadPipelineBuildsByApplicationAndPipeline(db, a.ID, p.ID, env.ID, limit, status, branchName)
	if errl != nil {
		return sdk.WrapError(errl, "getPipelineHistoryHandler> cannot load pipeline %s history", p.Name)
	}

	return WriteJSON(w, r, pbs, http.StatusOK)
}

func deletePipeline(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]

	p, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		if err != sdk.ErrPipelineNotFound {
			log.Warning("deletePipeline> Cannot load pipeline %s: %s\n", pipelineName, err)
		}
		return err
	}

	used, err := application.CountPipeline(db, p.ID)
	if err != nil {
		log.Warning("deletePipeline> Cannot check if pipeline is used by an application: %s\n", err)
		return err
	}

	if used {
		log.Warning("deletePipeline> Cannot delete a pipeline used by at least 1 application\n")
		return sdk.ErrPipelineHasApplication
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deletePipeline> Cannot begin transaction: %s\n", err)
		return err
	}
	defer tx.Rollback()

	err = pipeline.DeletePipeline(tx, p.ID, c.User.ID)
	if err != nil {
		log.Warning("deletePipeline> Cannot delete pipeline %s: %s\n", pipelineName, err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("deletePipeline> Cannot commit transaction: %s\n", err)
		return err
	}

	cache.DeleteAll(cache.Key("application", projectKey, "*"))
	cache.Delete(cache.Key("pipeline", projectKey, pipelineName))

	return nil
}

func addJobToPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	stageIDString := vars["stageID"]

	stageID, errInt := strconv.ParseInt(stageIDString, 10, 60)
	if errInt != nil {
		log.Warning("addJoinedActionToPipelineHandler> Stage ID must be an int: %s\n", errInt)
		return sdk.ErrInvalidID

	}

	var job sdk.Job
	if err := UnmarshalBody(r, &job); err != nil {
		return err
	}

	proj, errP := project.Load(db, projectKey, c.User, project.LoadOptions.Default)
	if errP != nil {
		log.Warning("addJoinedActionToPipelineHandler> Cannot load project %s: %s\n", projectKey, errP)
		return errP
	}

	pip, errPip := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if errPip != nil {
		log.Warning("addJoinedActionToPipelineHandler> Cannot load pipeline %s for project %s: %s\n", pipelineName, projectKey, errPip)
		return errPip
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		return errBegin
	}
	defer tx.Rollback()

	job.Enabled = true
	job.Action.Enabled = true
	if err := pipeline.InsertJob(tx, &job, stageID, pip); err != nil {
		log.Warning("addJoinedActionToPipelineHandler> Cannot insert job: %s\n", err)
		return err
	}

	warnings, errC := sanity.CheckAction(tx, proj, pip, job.Action.ID)
	if errC != nil {
		log.Warning("addActionToPipelineHandler> Cannot check action %d requirements: %s\n", job.Action.ID, errC)
		return errC
	}

	if err := sanity.InsertActionWarnings(tx, proj.ID, pip.ID, job.Action.ID, warnings); err != nil {
		log.Warning("addActionToPipelineHandler> Cannot insert warning for action %d: %s\n", job.Action.ID, err)
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	cache.DeleteAll(cache.Key("application", projectKey, "*"))
	cache.Delete(cache.Key("pipeline", projectKey, pipelineName))

	return WriteJSON(w, r, job, http.StatusOK)
}

func updateJoinedAction(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	actionIDString := vars["actionID"]
	key := vars["key"]
	pipName := vars["permPipelineKey"]

	proj, err := project.Load(db, key, c.User, project.LoadOptions.Default)
	if err != nil {
		log.Warning("updateJoinedAction> Cannot load project %s: %s\n", key, err)
		return sdk.ErrNoProject

	}

	pip, err := pipeline.LoadPipeline(db, key, pipName, false)
	if err != nil {
		log.Warning("updateJoinedAction> Cannot load pipeline %s for project %s: %s\n", pipName, key, err)
		return sdk.ErrPipelineNotFound

	}

	actionID, err := strconv.ParseInt(actionIDString, 10, 60)
	if err != nil {
		log.Warning("updateJoinedAction> Action ID must be an int: %s\n", err)
		return sdk.ErrWrongRequest

	}

	var a sdk.Action
	if err := UnmarshalBody(r, &a); err != nil {
		return err
	}
	a.ID = actionID

	clearJoinedAction, err := action.LoadActionByID(db, actionID)
	if err != nil {
		log.Warning("updateJoinedAction> Cannot load action %d: %s\n", actionID, err)
		return err

	}

	if clearJoinedAction.Type != sdk.JoinedAction {
		log.Warning("updateJoinedAction> Tried to update a %s action, aborting", clearJoinedAction.Type)
		return sdk.ErrForbidden

	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("updateJoinedAction> Cannot begin tx: %s\n", err)
		return err

	}
	defer tx.Rollback()

	if err := action.UpdateActionDB(tx, &a, c.User.ID); err != nil {
		log.Warning("updateJoinedAction> cannot update action: %s\n", err)
		return err

	}

	if err := pipeline.UpdatePipelineLastModified(tx, pip); err != nil {
		log.Warning("updateJoinedAction> cannot update pipeline last_modified date: %s\n", err)
		return err

	}

	warnings, err := sanity.CheckAction(tx, proj, pip, a.ID)
	if err != nil {
		log.Warning("updateJoinedAction> Cannot check action %d requirements: %s\n", a.ID, err)
		return err

	}

	err = sanity.InsertActionWarnings(tx, proj.ID, pip.ID, a.ID, warnings)
	if err != nil {
		log.Warning("updateJoinedAction> Cannot insert warning for action %d: %s\n", a.ID, err)
		return err

	}

	err = tx.Commit()
	if err != nil {
		log.Warning("updateJoinedAction> Cannot commit transaction: %s\n", err)
		return err

	}

	cache.DeleteAll(cache.Key("application", projectKey, "*"))
	cache.Delete(cache.Key("pipeline", projectKey, pip.Name))

	return WriteJSON(w, r, a, http.StatusOK)
}

func deleteJoinedAction(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipName := vars["permPipelineKey"]
	actionIDString := vars["actionID"]

	actionID, err := strconv.ParseInt(actionIDString, 10, 60)
	if err != nil {
		log.Warning("deleteJoinedAction> Action ID must be an int: %s\n", err)
		return sdk.ErrInvalidID

	}

	pip, err := pipeline.LoadPipeline(db, projectKey, pipName, false)
	if err != nil {
		log.Warning("deleteJoinedAction> Cannot load pipeline %s for project %s: %s\n", pipName, projectKey, err)
		return sdk.ErrPipelineNotFound

	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteJoinedAction> Cannot start transaction: %s", err)
		return err

	}
	defer tx.Rollback()

	err = action.DeleteAction(db, actionID, c.User.ID)
	if err != nil {
		log.Warning("deleteJoinedAction> Cannot delete joined action: %s\n", err)
		return err

	}

	err = pipeline.UpdatePipelineLastModified(tx, pip)
	if err != nil {
		log.Warning("deleteJoinedAction> Cannot update last_modified pipeline date: %s", err)
		return err

	}

	err = tx.Commit()
	if err != nil {
		log.Warning("deleteJoinedAction> Cannot commit transation: %s", err)
		return err

	}

	k := cache.Key("application", projectKey, "*")
	cache.DeleteAll(k)
	return nil
}

func getJoinedActionAudithandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	actionIDString := vars["actionID"]

	actionID, err := strconv.Atoi(actionIDString)
	if err != nil {
		log.Warning("getJoinedActionAudithandler> Action ID must be an int: %s\n", err)
		return sdk.ErrInvalidID

	}

	audit, err := action.LoadAuditAction(db, actionID, false)
	if err != nil {
		log.Warning("getJoinedActionAudithandler> Cannot load audit for action %d: %s\n", actionID, err)
		return err

	}

	return WriteJSON(w, r, audit, http.StatusOK)
}

func getJoinedAction(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	actionIDString := vars["actionID"]
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]

	actionID, err := strconv.ParseInt(actionIDString, 10, 60)
	if err != nil {
		log.Warning("getJoinedAction> Action ID must be an int: %s\n", err)
		return sdk.ErrInvalidID

	}

	a, err := action.LoadPipelineActionByID(db, projectKey, pipelineName, actionID)
	if err != nil {
		log.Warning("getJoinedAction> Cannot load joined action: %s\n", err)
		return err

	}

	return WriteJSON(w, r, a, http.StatusOK)
}

func getBuildingPipelines(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	var err error
	var pbs, recent []sdk.PipelineBuild

	if c.User.Admin {
		recent, err = pipeline.LoadRecentPipelineBuild(db)
	} else {
		recent, err = pipeline.LoadUserRecentPipelineBuild(db, c.User.ID)
	}
	if err != nil {
		log.Warning("getBuildingPipelines> cannot load recent pipelines: %s", err)
		return err

	}
	pbs = append(pbs, recent...)
	return WriteJSON(w, r, pbs, http.StatusOK)
}

func getPipelineBuildingCommit(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	hash := vars["hash"]

	pbs, err := pipeline.LoadPipelineBuildByHash(db, hash)
	if err != nil {
		return err

	}

	return WriteJSON(w, r, pbs, http.StatusOK)
}

func stopPipelineBuildHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]
	pipName := vars["permPipelineKey"]
	buildNumberS := vars["build"]

	err := r.ParseForm()
	if err != nil {
		log.Warning("stopPipelineBuildHandler> Cannot parse form: %s\n", err)
		return sdk.ErrUnknownError

	}
	envName := r.Form.Get("envName")

	buildNumber, err := strconv.ParseInt(buildNumberS, 10, 64)
	if err != nil {
		log.Warning("stopPipelineBuildHandler> buildNumber is not a int: %s\n", err)
		return sdk.ErrInvalidID

	}

	// Load pipeline
	pip, err := pipeline.LoadPipeline(db, projectKey, pipName, false)
	if err != nil {
		log.Warning("stopPipelineBuildHandler> Cannot load pipeline: %s\n", err)
		return err
	}

	// Load application
	app, err := application.LoadByName(db, projectKey, appName, c.User)
	if err != nil {
		log.Warning("stopPipelineBuildHandler> Cannot load application: %s\n", err)
		return err
	}

	// Load environment
	if pip.Type != sdk.BuildPipeline && (envName == "" || envName == sdk.DefaultEnv.Name) {
		return sdk.ErrNoEnvironmentProvided
	}
	env := &sdk.DefaultEnv

	if pip.Type != sdk.BuildPipeline {
		env, err = environment.LoadEnvironmentByName(db, projectKey, envName)
		if err != nil {
			log.Warning("stopPipelineBuildHandler> Cannot load environment %s: %s\n", envName, err)
			return err
		}
	}

	if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadExecute) {
		log.Warning("stopPipelineBuildHandler> You do not have Execution Right on this environment %s\n", env.Name)
		return sdk.ErrForbidden
	}

	pb, err := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(db, app.ID, pip.ID, env.ID, buildNumber)
	if err != nil {
		errFinal := err
		if err == sdk.ErrNoPipelineBuild {
			errFinal = sdk.ErrBuildArchived
		}
		log.Warning("stopPipelineBuildHandler> Cannot load pipeline Build: %s\n", errFinal)
		return errFinal
	}

	// Disable building worker
	for _, s := range pb.Stages {
		if s.Status != sdk.StatusBuilding {
			continue
		}

		for _, pipJob := range s.PipelineBuildJobs {
			if err := worker.DisableBuildingWorker(db, pipJob.ID); err != nil {
				log.Warning("stopPipelineBuildHandler> Cannot stop worker for pipeline build [%d-%d]: %s\n", pb.ID, pipJob.ID, err)
				return err
			}
		}
	}

	if err := pipeline.StopPipelineBuild(db, pb); err != nil {
		log.Warning("stopPipelineBuildHandler> Cannot stop pb: %s\n", err)
		return err
	}

	k := cache.Key("application", projectKey, "builds", "*")
	cache.DeleteAll(k)

	return nil
}

func restartPipelineBuildHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]
	pipName := vars["permPipelineKey"]
	buildNumberS := vars["build"]

	err := r.ParseForm()
	if err != nil {
		log.Warning("restartPipelineBuildHandler> Cannot parse form: %s\n", err)
		return sdk.ErrUnknownError

	}
	envName := r.Form.Get("envName")

	buildNumber, err := strconv.ParseInt(buildNumberS, 10, 64)
	if err != nil {
		log.Warning("restartPipelineBuildHandler> buildNumber is not a int: %s\n", err)
		return sdk.ErrInvalidID

	}

	// Load pipeline
	pip, err := pipeline.LoadPipeline(db, projectKey, pipName, false)
	if err != nil {
		log.Warning("restartPipelineBuildHandler> Cannot load pipeline: %s\n", err)
		return err

	}

	// Load application
	app, err := application.LoadByName(db, projectKey, appName, c.User)
	if err != nil {
		log.Warning("restartPipelineBuildHandler> Cannot load application: %s\n", err)
		return err

	}

	// Load environment
	if pip.Type != sdk.BuildPipeline && (envName == "" || envName == sdk.DefaultEnv.Name) {
		return sdk.ErrNoEnvironmentProvided

	}
	env := &sdk.DefaultEnv

	if pip.Type != sdk.BuildPipeline {
		env, err = environment.LoadEnvironmentByName(db, projectKey, envName)
		if err != nil {
			log.Warning("restartPipelineBuildHandler> Cannot load environment %s: %s\n", envName, err)
			return err

		}

		if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadExecute) {
			log.Warning("restartPipelineBuildHandler> No enought right on this environment %s: \n", envName)
			return sdk.ErrForbidden

		}
	}

	pb, err := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(db, app.ID, pip.ID, env.ID, buildNumber)
	if err != nil {
		errFinal := err
		if err == sdk.ErrNoPipelineBuild {
			errFinal = sdk.ErrBuildArchived
		}
		log.Warning("restartPipelineBuildHandler> Cannot load pipeline Build: %s\n", errFinal)
		return errFinal
	}

	if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadExecute) {
		log.Warning("restartPipelineBuildHandler> You do not have Execution Right on this environment %s\n", env.Name)
		return sdk.ErrNoEnvExecution
	}

	tx, errbegin := db.Begin()
	if errbegin != nil {
		log.Warning("restartPipelineBuildHandler> Cannot start transaction: %s\n", errbegin)
		return sdk.ErrNoEnvExecution
	}
	defer tx.Rollback()

	if err := pipeline.RestartPipelineBuild(tx, pb); err != nil {
		log.Warning("restartPipelineBuildHandler> cannot restart pb: %s\n", err)
		return err

	}

	if err := tx.Commit(); err != nil {
		log.Warning("restartPipelineBuildHandler> Cannot commit transaction: %s\n", err)
		return sdk.ErrNoEnvExecution

	}

	k := cache.Key("application", projectKey, "builds", "*")
	cache.DeleteAll(k)

	return WriteJSON(w, r, pb, http.StatusOK)
}

func getPipelineCommitsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]
	pipName := vars["permPipelineKey"]

	if err := r.ParseForm(); err != nil {
		log.Warning("getPipelineCommitsHandler> Cannot parse form: %s\n", err)
		return sdk.ErrUnknownError

	}
	envName := r.Form.Get("envName")
	hash := r.Form.Get("hash")

	// Load pipeline
	pip, err := pipeline.LoadPipeline(db, projectKey, pipName, false)
	if err != nil {
		log.Warning("getPipelineCommitsHandler> Cannot load pipeline: %s\n", err)
		return err

	}

	//Load the environment
	var env *sdk.Environment
	if envName == "" || envName == sdk.DefaultEnv.Name {
		env = &sdk.DefaultEnv
	} else {
		env, err = environment.LoadEnvironmentByName(db, projectKey, envName)
		if err != nil {
			log.Warning("getPipelineCommitsHandler> Cannot load environment %s: %s\n", envName, err)
			return err

		}
	}

	if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionRead) {
		log.Warning("getPipelineCommitsHandler> No enought right on this environment %s: \n", envName)
		return sdk.ErrForbidden
	}

	//Load the application
	app, err := application.LoadByName(db, projectKey, appName, c.User, application.LoadOptions.WithRepositoryManager)
	if err != nil {
		return err
	}

	commits := []sdk.VCSCommit{}

	//Check it the application is attached to a repository
	if app.RepositoriesManager == nil {
		log.Warning("getPipelineCommitsHandler> Application %s/%s not attached to a repository manager", projectKey, appName)
		return WriteJSON(w, r, commits, http.StatusOK)
	}

	pbs, err := pipeline.LoadPipelineBuildsByApplicationAndPipeline(db, app.ID, pip.ID, env.ID, 1, string(sdk.StatusSuccess), "")
	if err != nil {
		log.Warning("getPipelineCommitsHandler> Cannot load pipeline build %s: \n", err)
		return err

	}

	if len(pbs) != 1 {
		log.Warning("getPipelineCommitsHandler> There is no previous build")
		return WriteJSON(w, r, commits, http.StatusOK)

	}

	b, e := repositoriesmanager.CheckApplicationIsAttached(db, app.RepositoriesManager.Name, projectKey, appName)
	if e != nil {
		log.Warning("getPipelineCommitsHandler> Cannot check app (%s,%s,%s): %s", app.RepositoriesManager.Name, projectKey, appName, e)
		return e
	}

	if !b && app.RepositoryFullname == "" {
		log.Warning("getPipelineCommitsHandler> No repository on the application %s", appName)
		return WriteJSON(w, r, commits, http.StatusOK)
	}

	//Get the RepositoriesManager Client
	client, err := repositoriesmanager.AuthorizedClient(db, projectKey, app.RepositoriesManager.Name)
	if err != nil {
		log.Warning("getPipelineCommitsHandler> Cannot get client: %s", err)
		return sdk.ErrNoReposManagerClientAuth
	}

	if pbs[0].Trigger.VCSChangesHash == "" {
		log.Warning("getPipelineCommitsHandler>No hash on the previous run %d", pbs[0].ID)
		return WriteJSON(w, r, commits, http.StatusOK)
	}

	//If we are lucky, return a true diff
	commits, err = client.Commits(app.RepositoryFullname, pbs[0].Trigger.VCSChangesBranch, pbs[0].Trigger.VCSChangesHash, hash)
	if err != nil {
		log.Warning("getPipelineBuildCommitsHandler> Cannot get commits: %s", err)
		return err
	}
	return WriteJSON(w, r, commits, http.StatusOK)

}

func getPipelineBuildCommitsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]
	pipName := vars["permPipelineKey"]
	buildNumber, err := strconv.Atoi(vars["build"])
	if err != nil {
		return sdk.ErrInvalidID

	}

	if err := r.ParseForm(); err != nil {
		log.Warning("getPipelineBuildCommitsHandler> Cannot parse form: %s\n", err)
		return sdk.ErrUnknownError

	}
	envName := r.Form.Get("envName")

	// Load pipeline
	pip, err := pipeline.LoadPipeline(db, projectKey, pipName, false)
	if err != nil {
		log.Warning("getPipelineBuildCommitsHandler> Cannot load pipeline: %s\n", err)
		return err

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
			return err
		}
	}

	if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionRead) {
		log.Warning("getPipelineHistoryHandler> No enought right on this environment %s: \n", envName)
		return sdk.ErrForbidden
	}

	//Load the application
	app, err := application.LoadByName(db, projectKey, appName, c.User, application.LoadOptions.WithRepositoryManager)
	if err != nil {
		return sdk.ErrApplicationNotFound
	}

	//Check it the application is attached to a repository
	if app.RepositoriesManager == nil {
		return sdk.ErrNoReposManagerClientAuth
	}

	b, e := repositoriesmanager.CheckApplicationIsAttached(db, app.RepositoriesManager.Name, projectKey, appName)
	if e != nil {
		log.Warning("getPipelineBuildCommitsHandler> Cannot check app (%s,%s,%s): %s", app.RepositoriesManager.Name, projectKey, appName, e)
		return e
	}

	if !b && app.RepositoryFullname == "" {
		return sdk.ErrNoReposManagerClientAuth
	}

	//Get the RepositoriesManager Client
	client, err := repositoriesmanager.AuthorizedClient(db, projectKey, app.RepositoriesManager.Name)
	if err != nil {
		log.Warning("getPipelineBuildCommitsHandler> Cannot get client: %s", err)
		return sdk.ErrNoReposManagerClientAuth
	}

	//Get the commit hash for the pipeline build number and the hash for the previous pipeline build for the same branch
	//buildNumber, pipelineID, applicationID, environmentID
	cur, prev, err := pipeline.CurrentAndPreviousPipelineBuildNumberAndHash(db, int64(buildNumber), pip.ID, app.ID, env.ID)

	if err != nil {
		log.Warning("getPipelineBuildCommitsHandler> Cannot get build number and hashes (buildNumber=%d, pipelineID=%d, applicationID=%d, envID=%d)  : %s ", buildNumber, pip.ID, app.ID, env.ID, err)
		return err
	}

	if prev == nil {
		log.Info("getPipelineBuildCommitsHandler> No previous build was found for branch %s", cur.Branch)
	} else {
		log.Info("getPipelineBuildCommitsHandler> Current Build number: %d - Current Hash: %s - Previous Build number: %d - Previous Hash: %s", cur.BuildNumber, cur.Hash, prev.BuildNumber, prev.Hash)
	}

	//If there is not difference between the previous build and the current build
	if prev != nil && cur.Hash == prev.Hash {
		return WriteJSON(w, r, []sdk.VCSCommit{}, http.StatusOK)

	}

	if prev != nil && cur.Hash != "" && prev.Hash != "" {
		//If we are lucky, return a true diff
		commits, err := client.Commits(app.RepositoryFullname, cur.Branch, prev.Hash, cur.Hash)
		if err != nil {
			log.Warning("getPipelineBuildCommitsHandler> Cannot get commits: %s", err)
			return err

		}
		return WriteJSON(w, r, commits, http.StatusOK)

	}

	if cur.Hash != "" {
		//If we only get current pipeline build hash
		log.Info("getPipelineBuildCommitsHandler>  Looking for every commit until %s ", cur.Hash)
		c, err := client.Commits(app.RepositoryFullname, cur.Branch, "", cur.Hash)
		if err != nil {
			log.Warning("getPipelineBuildCommitsHandler> Cannot get commits: %s", err)
			return err

		}
		return WriteJSON(w, r, c, http.StatusOK)
	}

	//If we only have the current branch, search for the branch
	br, err := client.Branch(app.RepositoryFullname, cur.Branch)
	if err != nil {
		log.Warning("getPipelineBuildCommitsHandler> Cannot get branch: %s", err)
		return err

	}
	if br.LatestCommit == "" {
		log.Warning("getPipelineBuildCommitsHandler> Branch or lastest commit not found")
		return sdk.ErrNoBranch
	}
	//and return the last commit of the branch
	log.Debug("get the last commit : %s", br.LatestCommit)
	cm, err := client.Commit(app.RepositoryFullname, br.LatestCommit)
	if err != nil {
		log.Warning("getPipelineBuildCommitsHandler> Cannot get commits: %s", err)
		return err
	}
	return WriteJSON(w, r, []sdk.VCSCommit{cm}, http.StatusOK)
}
