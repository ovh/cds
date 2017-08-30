package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/businesscontext"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database"
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
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func rollbackPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	appName := vars["permApplicationName"]

	var request sdk.RunRequest
	if err := UnmarshalBody(r, &request); err != nil {
		return err
	}

	//Load the project
	proj, errproj := project.Load(db, projectKey, c.User)
	if errproj != nil {
		return sdk.WrapError(errproj, "rollbackPipelineHandler> Unable to load project %s", projectKey)
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

	if err := tx.Commit(); err != nil {
		log.Warning("rollbackPipelineHandler> Cannot commit tx: %s", err)
		return err
	}

	go func() {
		db := database.GetDBMap()
		if _, err := pipeline.UpdatePipelineBuildCommits(db, proj, pip, app, env, newPb); err != nil {
			log.Warning("scheduler.Run> Unable to update pipeline build commits : %s", err)
		}
	}()

	k := cache.Key("application", projectKey, "builds", "*")
	cache.DeleteAll(k)

	return WriteJSON(w, r, newPb, http.StatusOK)
}

func loadDestEnvFromRunRequest(db *gorp.DbMap, c *businesscontext.Ctx, request *sdk.RunRequest, projectKey string) (*sdk.Environment, error) {
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

func runPipelineWithLastParentHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	appName := vars["permApplicationName"]

	app, errl := application.LoadByName(db, projectKey, appName, c.User, application.LoadOptions.WithRepositoryManager, application.LoadOptions.WithTriggers, application.LoadOptions.WithVariablesWithClearPassword)
	if errl != nil {
		if errl != sdk.ErrApplicationNotFound {
			log.Warning("runPipelineWithLastParentHandler> Cannot load application %s: %s\n", appName, errl)
		}
		return errl
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
		return sdk.WrapError(sdk.ErrNoParentBuildFound, "runPipelineWithLastParentHandler> Unable to find any successful pipeline build")
	}
	if len(builds) == 0 {
		return sdk.ErrNoPipelineBuild
	}

	request.ParentBuildNumber = builds[0].BuildNumber

	return runPipelineHandlerFunc(w, r, db, c, &request)
}

func runPipelineHandlerFunc(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx, request *sdk.RunRequest) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	appName := vars["permApplicationName"]

	//Load the project
	proj, errproj := project.Load(db, projectKey, c.User)
	if errproj != nil {
		return sdk.WrapError(errproj, "runPipelineHandler> Unable to load project %s", projectKey)
	}

	app, errln := application.LoadByName(db, projectKey, appName, c.User, application.LoadOptions.WithRepositoryManager, application.LoadOptions.WithTriggers, application.LoadOptions.WithVariablesWithClearPassword)
	if errln != nil {
		if errln != sdk.ErrApplicationNotFound {
			log.Warning("runPipelineHandler> Cannot load application %s: %s\n", appName, errln)
		}
		return errln
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
		pb, errlp := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(db, request.ParentApplicationID, request.ParentPipelineID, envID, request.ParentBuildNumber)
		if errlp != nil {
			return sdk.WrapError(errlp, "runPipelineHandler> Cannot load parent pipeline build")
		}
		parentParams := queue.ParentBuildInfos(pb)
		request.Params = append(request.Params, parentParams...)

		version = pb.Version
		parentPipelineBuild = pb
	} else if request.ParentVersion != 0 {
		if request.ParentEnvironmentID != 0 {
			envID = request.ParentEnvironmentID
		}
		pbs, errP := pipeline.LoadPipelineBuildByApplicationPipelineEnvVersion(db, request.ParentApplicationID, request.ParentPipelineID, envID, request.ParentVersion, 1)
		if errP != nil {
			return sdk.WrapError(errP, "runPipelineHandler> Cannot load parent pipeline build by version")
		}
		if len(pbs) == 0 {
			return sdk.WrapError(sdk.ErrNoParentBuildFound, "runPipelineHandler> No parent build found")
		}
		parentParams := queue.ParentBuildInfos(&pbs[0])
		request.Params = append(request.Params, parentParams...)

		version = pbs[0].Version
		parentPipelineBuild = &pbs[0]
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
	log.Debug("runPipelineHandler> Scheduling %s/%s/%s[%s] with %d params, version 0",
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

	env, errenv := environment.LoadEnvironmentByName(db, projectKey, envDest.Name)
	if errenv != nil {
		return sdk.WrapError(errenv, "runPipelineHandler> Unable to load env %s %s", projectKey, envDest.Name)
	}

	pb, err := queue.RunPipeline(tx, projectKey, app, pipelineName, envDest.Name, request.Params, version, trigger, c.User)
	if err != nil {
		return sdk.WrapError(err, "runPipelineHandler> Cannot run pipeline")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "runPipelineHandler> Cannot commit tx")
	}

	go func() {
		db := database.GetDBMap()
		if _, err := pipeline.UpdatePipelineBuildCommits(db, proj, pip, app, env, pb); err != nil {
			log.Warning("runPipelineHandler> Unable to update pipeline build commits : %s", err)
		}
	}()

	return WriteJSON(w, r, pb, http.StatusOK)
}

func runPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	var request sdk.RunRequest
	if err := UnmarshalBody(r, &request); err != nil {
		return err
	}

	return runPipelineHandlerFunc(w, r, db, c, &request)
}

// DEPRECATED
func updatePipelineActionHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	key := vars["key"]
	pipName := vars["permPipelineKey"]
	pipelineActionIDString := vars["pipelineActionID"]

	pipelineActionID, errp := strconv.ParseInt(pipelineActionIDString, 10, 64)
	if errp != nil {
		return sdk.WrapError(sdk.ErrInvalidID, "updatePipelineActionHandler>ID %s is not a int", pipelineActionID)
	}

	var job sdk.Job
	if err := UnmarshalBody(r, &job); err != nil {
		return err
	}

	if pipelineActionID != job.PipelineActionID {
		return fmt.Errorf("updatePipelineActionHandler>Pipeline action does not match")
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

	//Load the project
	proj, errproj := project.Load(db, key, c.User)
	if errproj != nil {
		return sdk.WrapError(errproj, "updatePipelineActionHandler> Unable to load project %s", key)
	}

	err = pipeline.UpdatePipelineLastModified(tx, proj, pipelineData, c.User)
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

func deletePipelineActionHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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

	log.Info("deletePipelineActionHandler> Deleting action %d in %s/%s\n", pipelineActionID, vars["key"], vars["permPipelineKey"])

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

	//Load the project
	proj, errproj := project.Load(db, key, c.User)
	if errproj != nil {
		return sdk.WrapError(errproj, "updatePipelineActionHandler> Unable to load project %s", key)
	}

	err = pipeline.UpdatePipelineLastModified(tx, proj, pipelineData, c.User)
	if err != nil {
		log.Warning("deletePipelineActionHandler> Cannot update pipeline last_modified: %s", err)
		return err
	}
	err = tx.Commit()
	if err != nil {
		log.Warning("deletePipelineActionHandler> Cannot commit transaction: %s", err)
		return err
	}

	return nil
}

func updatePipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	name := vars["permPipelineKey"]

	proj, errP := project.Load(db, key, c.User)
	if errP != nil {
		return sdk.WrapError(errP, "updatePipelineHandler> Cannot load project")
	}

	var p sdk.Pipeline
	if err := UnmarshalBody(r, &p); err != nil {
		return sdk.WrapError(err, "updatePipelineHandler> Cannot read body")
	}

	// check pipeline name pattern
	regexp := regexp.MustCompile(sdk.NamePattern)
	if !regexp.MatchString(p.Name) {
		return sdk.WrapError(sdk.ErrInvalidPipelinePattern, "updatePipelineHandler: Pipeline name %s do not respect pattern", p.Name)
	}

	pipelineDB, err := pipeline.LoadPipeline(db, key, name, false)
	if err != nil {
		return sdk.WrapError(err, "updatePipelineHandler> cannot load pipeline %s", name)
	}

	pipelineDB.Name = p.Name
	pipelineDB.Type = p.Type

	tx, errB := db.Begin()
	if errB != nil {
		sdk.WrapError(errB, "updatePipelineHandler> Cannot start transaction")
	}
	defer tx.Rollback()

	if err := pipeline.UpdatePipeline(tx, pipelineDB); err != nil {
		return sdk.WrapError(err, "updatePipelineHandler> cannot update pipeline %s", name)
	}

	if err := pipeline.UpdatePipelineLastModified(tx, proj, pipelineDB, c.User); err != nil {
		return sdk.WrapError(err, "updatePipelineHandler> Cannot update pipeline last modified date")
	}

	if err := project.UpdateLastModified(tx, c.User, proj); err != nil {
		return sdk.WrapError(err, "updatePipelineHandler> cannot update project last modified date")
	}

	// Update applications
	apps, errA := application.LoadByPipeline(tx, p.ID, c.User)
	if errA != nil {
		return sdk.WrapError(errA, "updatePipelineHandler> Cannot load application using pipeline %s", p.Name)
	}

	for _, app := range apps {
		if err := application.UpdateLastModified(tx, &app, c.User); err != nil {
			return sdk.WrapError(err, "updatePipelineHandler> Cannot update application last modified date")
		}
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "updatePipelineHandler> Cannot commit transaction")
	}

	cache.DeleteAll(cache.Key("application", key, "*"))
	cache.Delete(cache.Key("pipeline", key, name))

	return WriteJSON(w, r, pipelineDB, http.StatusOK)
}

func getApplicationUsingPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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

func addPipeline(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get project name in URL
	vars := mux.Vars(r)
	key := vars["permProjectKey"]

	proj, errl := project.Load(db, key, c.User, project.LoadOptions.Default)
	if errl != nil {
		return sdk.WrapError(errl, "AddPipeline: Cannot load %s", key)
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
	exist, err := pipeline.ExistPipeline(db, proj.ID, p.Name)
	if err != nil {
		return sdk.WrapError(err, "cannot check if pipeline exist")
	}
	if exist {
		log.Warning("addPipeline> Pipeline %s already exists", p.Name)
		return sdk.ErrConflict
	}

	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "Cannot start transaction")
	}
	defer tx.Rollback()

	p.ProjectID = proj.ID
	if err := pipeline.InsertPipeline(tx, proj, &p, c.User); err != nil {
		return sdk.WrapError(err, "Cannot insert pipeline")
	}

	if err := group.LoadGroupByProject(tx, proj); err != nil {
		return sdk.WrapError(err, "Cannot load groupfrom project")
	}

	for _, g := range proj.ProjectGroups {
		p.GroupPermission = append(p.GroupPermission, g)
	}

	if err := group.InsertGroupsInPipeline(tx, proj.ProjectGroups, p.ID); err != nil {
		return sdk.WrapError(err, "Cannot add groups on pipeline")
	}

	for _, app := range p.AttachedApplication {
		if _, err := application.AttachPipeline(tx, app.ID, p.ID); err != nil {
			return sdk.WrapError(err, "Cannot attach pipeline %d to %d", app.ID, p.ID)
		}

		if err := application.UpdateLastModified(tx, &app, c.User); err != nil {
			return sdk.WrapError(err, "Cannot update application last modified date")
		}
	}

	if err := project.UpdateLastModified(tx, c.User, proj); err != nil {
		return sdk.WrapError(err, "Cannot update project last modified date")
	}
	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "Cannot commit transaction")
	}

	p.Permission = permission.PermissionReadWriteExecute

	return WriteJSON(w, r, p, http.StatusOK)
}

func getPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	withApp := FormBool(r, "withApplications")

	p, err := pipeline.LoadPipeline(db, projectKey, pipelineName, true)
	if err != nil {
		log.Warning("getPipelineHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
		return err
	}

	p.Permission = permission.PipelinePermission(p.ID, c.User)

	if withApp {
		apps, errA := application.LoadByPipeline(db, p.ID, c.User)
		if errA != nil {
			return sdk.WrapError(errA, "getApplicationUsingPipelineHandler> Cannot load applications using pipeline %s", p.Name)
		}
		p.AttachedApplication = apps
	}

	return WriteJSON(w, r, p, http.StatusOK)
}

func getPipelineTypeHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	return WriteJSON(w, r, sdk.AvailablePipelineType, http.StatusOK)
}

func getPipelinesHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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

func getPipelineHistoryHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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

func deletePipeline(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	key := vars["key"]
	pipelineName := vars["permPipelineKey"]

	proj, errP := project.Load(db, key, c.User)
	if errP != nil {
		return sdk.WrapError(errP, "deletePipeline> Cannot load project")
	}

	p, err := pipeline.LoadPipeline(db, proj.Key, pipelineName, false)
	if err != nil {
		return sdk.WrapError(err, "deletePipeline> Cannot load pipeline %s", pipelineName)
	}

	used, err := application.CountPipeline(db, p.ID)
	if err != nil {
		return sdk.WrapError(err, "deletePipeline> Cannot check if pipeline is used by an application")
	}

	if used {
		return sdk.WrapError(sdk.ErrPipelineHasApplication, "deletePipeline> Cannot delete a pipeline used by at least 1 application")
	}

	tx, errT := db.Begin()
	if errT != nil {
		return sdk.WrapError(errT, "deletePipeline> Cannot begin transaction")
	}
	defer tx.Rollback()

	if err := pipeline.DeletePipeline(tx, p.ID, c.User.ID); err != nil {
		log.Warning("deletePipeline> Cannot delete pipeline %s: %s\n", pipelineName, err)
		return err
	}

	if err := project.UpdateLastModified(db, c.User, proj); err != nil {
		return sdk.WrapError(err, "deletePipeline> Cannot update project last modified date")
	}

	if err := tx.Commit(); err != nil {
		log.Warning("deletePipeline> Cannot commit transaction: %s\n", err)
		return err
	}

	cache.DeleteAll(cache.Key("application", proj.Key, "*"))
	cache.Delete(cache.Key("pipeline", proj.Key, pipelineName))

	return nil
}

func addJobToPipelineHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	stageIDString := vars["stageID"]

	stageID, errInt := strconv.ParseInt(stageIDString, 10, 60)
	if errInt != nil {
		return sdk.WrapError(sdk.ErrInvalidID, "addJoinedActionToPipelineHandler> Stage ID must be an int", stageID)
	}

	var job sdk.Job
	if err := UnmarshalBody(r, &job); err != nil {
		return err
	}

	proj, errP := project.Load(db, projectKey, c.User, project.LoadOptions.Default)
	if errP != nil {
		return sdk.WrapError(errP, "addJoinedActionToPipelineHandler> Cannot load project %s", projectKey)
	}

	pip, errPip := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if errPip != nil {
		return sdk.WrapError(errPip, "addJoinedActionToPipelineHandler> Cannot load pipeline %s for project %s", pipelineName, projectKey)
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		return errBegin
	}
	defer tx.Rollback()

	reqs, errlb := action.LoadAllBinaryRequirements(tx)
	if errlb != nil {
		return sdk.WrapError(errlb, "updateJoinedAction> cannot load all binary requirements")
	}

	job.Enabled = true
	job.Action.Enabled = true
	if err := pipeline.InsertJob(tx, &job, stageID, pip); err != nil {
		return sdk.WrapError(err, "addJoinedActionToPipelineHandler> Cannot insert job")
	}

	warnings, errC := sanity.CheckAction(tx, proj, pip, job.Action.ID)
	if errC != nil {
		return sdk.WrapError(errC, "addActionToPipelineHandler> Cannot check action %d requirements", job.Action.ID)
	}

	if err := sanity.InsertActionWarnings(tx, proj.ID, pip.ID, job.Action.ID, warnings); err != nil {
		return sdk.WrapError(err, "addActionToPipelineHandler> Cannot insert warning for action %d", job.Action.ID)
	}

	if err := worker.ComputeRegistrationNeeds(tx, reqs, job.Action.Requirements); err != nil {
		return sdk.WrapError(err, "addActionToPipelineHandler> Cannot compute registration needs")
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	cache.DeleteAll(cache.Key("application", projectKey, "*"))
	cache.Delete(cache.Key("pipeline", projectKey, pipelineName))

	return WriteJSON(w, r, job, http.StatusOK)
}

func updateJoinedAction(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	actionIDString := vars["actionID"]
	key := vars["key"]
	pipName := vars["permPipelineKey"]

	proj, errl := project.Load(db, key, c.User, project.LoadOptions.Default)
	if errl != nil {
		return sdk.WrapError(sdk.ErrNoProject, "updateJoinedAction> Cannot load project %s: %s", key, errl)
	}

	pip, errlp := pipeline.LoadPipeline(db, key, pipName, false)
	if errlp != nil {
		return sdk.WrapError(sdk.ErrPipelineNotFound, "updateJoinedAction> Cannot load pipeline %s for project %s: %s", pipName, key, errlp)
	}

	actionID, errp := strconv.ParseInt(actionIDString, 10, 60)
	if errp != nil {
		return sdk.WrapError(sdk.ErrWrongRequest, "updateJoinedAction> Action ID %s must be an int", actionID)
	}

	var a sdk.Action
	if err := UnmarshalBody(r, &a); err != nil {
		return err
	}
	a.ID = actionID

	clearJoinedAction, err := action.LoadActionByID(db, actionID)
	if err != nil {
		return sdk.WrapError(err, "updateJoinedAction> Cannot load action %d", actionID)
	}

	if clearJoinedAction.Type != sdk.JoinedAction {
		return sdk.WrapError(sdk.ErrForbidden, "updateJoinedAction> Tried to update a %s action, aborting", clearJoinedAction.Type)
	}

	tx, errb := db.Begin()
	if errb != nil {
		return sdk.WrapError(errb, "updateJoinedAction> Cannot begin tx")
	}
	defer tx.Rollback()

	reqs, errlb := action.LoadAllBinaryRequirements(tx)
	if errlb != nil {
		return sdk.WrapError(errlb, "updateJoinedAction> cannot load all binary requirements")
	}

	log.Debug("updateJoinedAction> UpdateActionDB %d", a.ID)
	if err := action.UpdateActionDB(tx, &a, c.User.ID); err != nil {
		return sdk.WrapError(err, "updateJoinedAction> cannot update action")
	}

	if err := pipeline.UpdatePipelineLastModified(tx, proj, pip, c.User); err != nil {
		return sdk.WrapError(err, "updateJoinedAction> cannot update pipeline last_modified date")
	}

	log.Debug("updateJoinedAction> CheckAction %d", a.ID)
	warnings, errc := sanity.CheckAction(tx, proj, pip, a.ID)
	if errc != nil {
		return sdk.WrapError(errc, "updateJoinedAction> Cannot check action %d requirements", a.ID)
	}

	if err := sanity.InsertActionWarnings(tx, proj.ID, pip.ID, a.ID, warnings); err != nil {
		return sdk.WrapError(err, "updateJoinedAction> Cannot insert warning for action %d", a.ID)
	}

	if err := worker.ComputeRegistrationNeeds(tx, reqs, a.Requirements); err != nil {
		return sdk.WrapError(err, "updateJoinedAction> Cannot compute registration needs")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "updateJoinedAction> Cannot commit transaction")
	}

	cache.DeleteAll(cache.Key("application", projectKey, "*"))
	cache.Delete(cache.Key("pipeline", projectKey, pip.Name))

	return WriteJSON(w, r, a, http.StatusOK)
}

func deleteJoinedAction(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipName := vars["permPipelineKey"]
	actionIDString := vars["actionID"]

	actionID, errp := strconv.ParseInt(actionIDString, 10, 60)
	if errp != nil {
		return sdk.WrapError(sdk.ErrInvalidID, "deleteJoinedAction> Action ID %s must be an int", actionIDString)
	}

	pip, errload := pipeline.LoadPipeline(db, projectKey, pipName, false)
	if errload != nil {
		return sdk.WrapError(sdk.ErrPipelineNotFound, "deleteJoinedAction> Cannot load pipeline %s for project %s, err:%s", pipName, projectKey, errload)
	}

	tx, errb := db.Begin()
	if errb != nil {
		return sdk.WrapError(errb, "deleteJoinedAction> Cannot start transaction")
	}
	defer tx.Rollback()

	if err := action.DeleteAction(db, actionID, c.User.ID); err != nil {
		return sdk.WrapError(err, "deleteJoinedAction> Cannot delete joined action")
	}

	//Load the project
	proj, errproj := project.Load(db, projectKey, c.User)
	if errproj != nil {
		return sdk.WrapError(errproj, "deleteJoinedAction> Unable to load project %s", projectKey)
	}

	if err := pipeline.UpdatePipelineLastModified(tx, proj, pip, c.User); err != nil {
		return sdk.WrapError(err, "deleteJoinedAction> cannot update pipeline last_modified date")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "deleteJoinedAction> Cannot commit transaction")
	}

	k := cache.Key("application", projectKey, "*")
	cache.DeleteAll(k)
	return nil
}

func getJoinedActionAudithandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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

func getJoinedAction(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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

func getBuildingPipelines(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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

func getPipelineBuildingCommit(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	hash := vars["hash"]

	pbs, err := pipeline.LoadPipelineBuildByHash(db, hash)
	if err != nil {
		return err

	}

	return WriteJSON(w, r, pbs, http.StatusOK)
}

func stopPipelineBuildHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]
	pipName := vars["permPipelineKey"]

	if err := r.ParseForm(); err != nil {
		return sdk.WrapError(sdk.ErrUnknownError, "stopPipelineBuildHandler> Cannot parse form")
	}
	envName := r.Form.Get("envName")

	buildNumber, err := requestVarInt(r, "build")
	if err != nil {
		return sdk.WrapError(err, "stopPipelineBuildHandler> invalid build number")
	}

	pip, err := pipeline.LoadPipeline(db, projectKey, pipName, false)
	if err != nil {
		return sdk.WrapError(err, "stopPipelineBuildHandler> Cannot load pipeline")
	}

	app, err := application.LoadByName(db, projectKey, appName, c.User)
	if err != nil {
		return sdk.WrapError(err, "stopPipelineBuildHandler> Cannot load application")
	}

	if pip.Type != sdk.BuildPipeline && (envName == "" || envName == sdk.DefaultEnv.Name) {
		return sdk.ErrNoEnvironmentProvided
	}
	env := &sdk.DefaultEnv

	if pip.Type != sdk.BuildPipeline {
		env, err = environment.LoadEnvironmentByName(db, projectKey, envName)
		if err != nil {
			return sdk.WrapError(err, "stopPipelineBuildHandler> Cannot load environment %s", envName)
		}
	}

	if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadExecute) {
		return sdk.WrapError(sdk.ErrForbidden, "stopPipelineBuildHandler> You do not have Execution Right on this environment %s", env.Name)
	}

	pb, err := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(db, app.ID, pip.ID, env.ID, buildNumber)
	if err != nil {
		errFinal := err
		if err == sdk.ErrNoPipelineBuild {
			errFinal = sdk.ErrBuildArchived
		}
		return sdk.WrapError(errFinal, "stopPipelineBuildHandler> Cannot load pipeline Build")
	}

	if err := pipeline.StopPipelineBuild(db, pb); err != nil {
		return sdk.WrapError(err, "stopPipelineBuildHandler> Cannot stop pipeline build")
	}

	k := cache.Key("application", projectKey, "builds", "*")
	cache.DeleteAll(k)

	return nil
}

func restartPipelineBuildHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
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

func getPipelineCommitsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]
	pipName := vars["permPipelineKey"]

	if err := r.ParseForm(); err != nil {
		return sdk.WrapError(sdk.ErrUnknownError, "getPipelineCommitsHandler> Cannot parse form")

	}
	envName := r.Form.Get("envName")
	hash := r.Form.Get("hash")

	// Load pipeline
	pip, errpip := pipeline.LoadPipeline(db, projectKey, pipName, false)
	if errpip != nil {
		return sdk.WrapError(errpip, "getPipelineCommitsHandler> Cannot load pipeline")
	}

	//Load the environment
	var env *sdk.Environment
	if envName == "" || envName == sdk.DefaultEnv.Name {
		env = &sdk.DefaultEnv
	} else {
		var err error
		env, err = environment.LoadEnvironmentByName(db, projectKey, envName)
		if err != nil {
			return sdk.WrapError(err, "getPipelineCommitsHandler> Cannot load environment %s", envName)
		}
	}

	if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionRead) {
		return sdk.WrapError(sdk.ErrForbidden, "getPipelineCommitsHandler> No enought right on this environment %s (user=%s)", envName, c.User.Username)
	}

	//Load the application
	app, errapp := application.LoadByName(db, projectKey, appName, c.User, application.LoadOptions.WithRepositoryManager)
	if errapp != nil {
		return sdk.WrapError(errapp, "getPipelineCommitsHandler> Unable to load application %s", appName)
	}

	commits := []sdk.VCSCommit{}

	//Check it the application is attached to a repository
	if app.RepositoriesManager == nil {
		log.Warning("getPipelineCommitsHandler> Application %s/%s not attached to a repository manager", projectKey, appName)
		return WriteJSON(w, r, commits, http.StatusOK)
	}

	pbs, errpb := pipeline.LoadPipelineBuildsByApplicationAndPipeline(db, app.ID, pip.ID, env.ID, 1, string(sdk.StatusSuccess), "")
	if errpb != nil {
		return sdk.WrapError(errpb, "getPipelineCommitsHandler> Cannot load pipeline build")
	}

	if len(pbs) != 1 {
		log.Debug("getPipelineCommitsHandler> There is no previous build")
		return WriteJSON(w, r, commits, http.StatusOK)
	}

	b, e := repositoriesmanager.CheckApplicationIsAttached(db, app.RepositoriesManager.Name, projectKey, appName)
	if e != nil {
		log.Warning("getPipelineCommitsHandler> Cannot check app (%s,%s,%s): %s", app.RepositoriesManager.Name, projectKey, appName, e)
		return e
	}

	if !b && app.RepositoryFullname == "" {
		log.Debug("getPipelineCommitsHandler> No repository on the application %s", appName)
		return WriteJSON(w, r, commits, http.StatusOK)
	}

	//Get the RepositoriesManager Client
	client, errclient := repositoriesmanager.AuthorizedClient(db, projectKey, app.RepositoriesManager.Name)
	if errclient != nil {
		return sdk.WrapError(errclient, "getPipelineCommitsHandler> Cannot get client")
	}

	if pbs[0].Trigger.VCSChangesHash == "" {
		log.Debug("getPipelineCommitsHandler>No hash on the previous run %d", pbs[0].ID)
		return WriteJSON(w, r, commits, http.StatusOK)
	}

	//If we are lucky, return a true diff
	var errcommits error
	commits, errcommits = client.Commits(app.RepositoryFullname, pbs[0].Trigger.VCSChangesBranch, pbs[0].Trigger.VCSChangesHash, hash)
	if errcommits != nil {
		return sdk.WrapError(errcommits, "getPipelineBuildCommitsHandler> Cannot get commits")
	}

	return WriteJSON(w, r, commits, http.StatusOK)
}

func getPipelineBuildCommitsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *businesscontext.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	appName := vars["permApplicationName"]
	pipName := vars["permPipelineKey"]
	buildNumber, err := strconv.Atoi(vars["build"])
	if err != nil {
		return sdk.ErrInvalidID

	}

	proj, errproj := project.Load(db, projectKey, c.User)
	if errproj != nil {
		return sdk.WrapError(errproj, "getPipelineBuildCommitsHandler> Unable to load project %s", projectKey)
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

	//Load the pipeline build
	pb, errpb := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(db, app.ID, pip.ID, env.ID, int64(buildNumber))
	if errpb != nil {
		return sdk.WrapError(errpb, "getPipelineBuildCommitsHandler>")
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

	cm, err := pipeline.UpdatePipelineBuildCommits(db, proj, pip, app, env, pb)
	if err != nil {
		return sdk.WrapError(err, "getPipelineBuildCommitsHandler> UpdatePipelineBuildCommits failed")
	}
	return WriteJSON(w, r, cm, http.StatusOK)
}
