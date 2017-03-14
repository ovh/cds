package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/runabove/venom"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/artifact"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/stats"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func updateStepStatusHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	buildIDString := vars["id"]

	buildID, errID := strconv.ParseInt(buildIDString, 10, 64)
	if errID != nil {
		log.Warning("updateStepStatusHandler> buildID must be an integer: %s\n", errID)
		return sdk.ErrInvalidID
	}

	pbJob, errJob := pipeline.GetPipelineBuildJob(db, buildID)
	if errJob != nil {
		log.Warning("updateStepStatusHandler> Cannot get pipeline build job %d: %s\n", buildID, errJob)
		return errJob
	}

	var step sdk.StepStatus
	if err := UnmarshalBody(r, &step); err != nil {
		return err
	}

	found := false
	for i := range pbJob.Job.StepStatus {
		jobStep := &pbJob.Job.StepStatus[i]
		if step.StepOrder == jobStep.StepOrder {
			jobStep.Status = step.Status
			found = true
		}
	}
	if !found {
		pbJob.Job.StepStatus = append(pbJob.Job.StepStatus, step)
	}

	var errmarshal error
	pbJob.JobJSON, errmarshal = json.Marshal(pbJob.Job)
	if errmarshal != nil {
		log.Warning("updateStepStatusHandler> Cannot marshall job: %s\n", errmarshal)
		return errmarshal
	}
	if err := pipeline.UpdatePipelineBuildJob(db, pbJob); err != nil {
		log.Warning("updateStepStatusHandler> Cannot update pipeline build job: %s\n", err)
		return err
	}
	return nil
}

func getPipelineBuildTriggeredHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	buildNumberS := vars["build"]
	appName := vars["permApplicationName"]

	envName := r.FormValue("envName")

	buildNumber, err := strconv.ParseInt(buildNumberS, 10, 64)
	if err != nil {
		log.Warning("getPipelineBuildTriggeredHandler> BuildNumber %s is not an integer: %s\n", buildNumberS, err)
		return err
	}

	// Load Pipeline
	p, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		log.Warning("getPipelineBuildTriggeredHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
		return sdk.ErrPipelineNotFound
	}

	// Load Application
	a, err := application.LoadByName(db, projectKey, appName, c.User)
	if err != nil {
		log.Warning("getPipelineBuildTriggeredHandler> Cannot load application %s: %s\n", appName, err)
		return sdk.ErrApplicationNotFound
	}

	// Load Env
	env := &sdk.DefaultEnv
	if envName != sdk.DefaultEnv.Name && envName != "" {
		env, err = environment.LoadEnvironmentByName(db, projectKey, envName)
		if err != nil {
			log.Warning("getPipelineBuildTriggeredHandler> Cannot load environment %s: %s\n", envName, err)
			return sdk.ErrNoEnvironment
		}
	}

	// Load Children
	pbs, err := pipeline.LoadPipelineBuildChildren(db, p.ID, a.ID, buildNumber, env.ID)
	if err != nil {
		log.Warning("getPipelineBuildTriggeredHandler> Cannot load pipeline build children: %s\n", err)
		return sdk.ErrNoPipelineBuild
	}
	return WriteJSON(w, r, pbs, http.StatusOK)
}

func deleteBuildHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	buildNumberS := vars["build"]
	appName := vars["permApplicationName"]

	envName := r.FormValue("envName")

	// Check that pipeline exists
	p, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		log.Warning("deleteBuildHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
		return sdk.ErrPipelineNotFound
	}

	a, err := application.LoadByName(db, projectKey, appName, c.User)
	if err != nil {
		log.Warning("deleteBuildHandler> Cannot load application %s: %s\n", appName, err)
		return sdk.ErrApplicationNotFound
	}

	var env *sdk.Environment
	if envName == "" || envName == sdk.DefaultEnv.Name {
		env = &sdk.DefaultEnv
	} else {
		env, err = environment.LoadEnvironmentByName(db, projectKey, envName)
		if err != nil {
			log.Warning("deleteBuildHandler> Cannot load environment %s: %s\n", envName, err)
			return sdk.ErrUnknownEnv
		}
	}

	if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionRead) {
		log.Warning("deleteBuildHandler> No enought right on this environment %s: \n", envName)
		return sdk.ErrForbidden
	}

	var buildNumber int64
	buildNumber, err = strconv.ParseInt(buildNumberS, 10, 64)
	if err != nil {
		log.Warning("deleteBuildHandler> Cannot parse build number %s: %s\n", buildNumberS, err)
		return sdk.ErrWrongRequest
	}

	pbID, errPB := pipeline.LoadPipelineBuildID(db, a.ID, p.ID, env.ID, buildNumber)
	if errPB != nil {
		log.Warning("deleteBuildHandler> Cannot load pipeline build: %s", errPB)
		return errPB
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteBuildHandler> Cannot start transaction: %s\n", err)
		return err
	}
	defer tx.Rollback()

	if err := pipeline.DeletePipelineBuildByID(tx, pbID); err != nil {
		log.Warning("deleteBuildHandler> Cannot delete pipeline build: %s\n", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("deleteBuildHandler> Cannot commit transaction: %s\n", err)
		return err
	}

	return nil
}

func getBuildStateHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	buildNumberS := vars["build"]
	appName := vars["permApplicationName"]

	envName := r.FormValue("envName")
	withArtifacts := r.FormValue("withArtifacts")
	withTests := r.FormValue("withTests")

	// Check that pipeline exists
	p, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		log.Warning("getBuildStateHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
		return sdk.ErrPipelineNotFound
	}

	a, err := application.LoadByName(db, projectKey, appName, c.User)
	if err != nil {
		log.Warning("getBuildStateHandler> Cannot load application %s: %s\n", appName, err)
		return sdk.ErrApplicationNotFound
	}

	var env *sdk.Environment
	if envName == "" || envName == sdk.DefaultEnv.Name {
		env = &sdk.DefaultEnv
	} else {
		env, err = environment.LoadEnvironmentByName(db, projectKey, envName)
		if err != nil {
			log.Warning("getBuildStateHandler> Cannot load environment %s: %s\n", envName, err)
			return sdk.ErrUnknownEnv
		}
	}

	if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionRead) {
		log.Warning("getBuildStateHandler> No enought right on this environment %s: \n", envName)
		return sdk.ErrForbidden
	}

	// if buildNumber is 'last' fetch last build number
	var buildNumber int64
	if buildNumberS == "last" {
		lastBuildNumber, errg := pipeline.GetLastBuildNumberInTx(db, p.ID, a.ID, env.ID)
		if errg != nil {
			log.Warning("getBuildStateHandler> Cannot load last pipeline build number for %s-%s-%s: %s\n", a.Name, pipelineName, env.Name, errg)
			return sdk.ErrNotFound
		}
		buildNumber = lastBuildNumber
	} else {
		buildNumber, err = strconv.ParseInt(buildNumberS, 10, 64)
		if err != nil {
			log.Warning("getBuildStateHandler> Cannot parse build number %s: %s\n", buildNumberS, err)
			return sdk.ErrWrongRequest
		}
	}

	// load pipeline_build.id
	pb, err := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(db, a.ID, p.ID, env.ID, buildNumber)
	if err != nil {
		log.Warning("getBuildStateHandler> %s! Cannot load last pipeline build for %s-%s-%s[%s] (buildNUmber:%d): %s\n", c.User.Username, projectKey, appName, pipelineName, env.Name, buildNumber, err)
		return err
	}

	if withArtifacts == "true" {
		var errLoadArtifact error
		pb.Artifacts, errLoadArtifact = artifact.LoadArtifactsByBuildNumber(db, p.ID, a.ID, buildNumber, env.ID)
		if errLoadArtifact != nil {
			log.Warning("getBuildStateHandler> Cannot load artifacts: %s", errLoadArtifact)
			return errLoadArtifact
		}
	}

	if withTests == "true" {
		tests, errLoadTests := pipeline.LoadTestResults(db, pb.ID)
		if errLoadTests != nil {
			log.Warning("getBuildStateHandler> Cannot load tests: %s", errLoadTests)
			return errLoadTests
		}
		if len(tests.TestSuites) > 0 {
			pb.Tests = &tests
		}
	}
	pb.Translate(r.Header.Get("Accept-Language"))

	return WriteJSON(w, r, pb, http.StatusOK)
}

func addQueueResultHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get action name in URL
	vars := mux.Vars(r)
	idString := vars["id"]

	id, errInt := strconv.ParseInt(idString, 10, 64)
	if errInt != nil {
		return sdk.ErrInvalidID
	}

	// Load Build
	pbJob, errJob := pipeline.GetPipelineBuildJob(db, id)
	if errJob != nil {
		log.Warning("addQueueResultHandler> Cannot load queue (%d) from db: %s\n", id, errJob)
		return sdk.ErrNotFound
	}

	// Unmarshal into results
	var res sdk.Result
	if err := UnmarshalBody(r, &res); err != nil {
		return err
	}

	tx, errb := db.Begin()
	if errb != nil {
		log.Warning("addQueueResultHandler> Cannot begin tx: %s\n", errb)
		return sdk.ErrUnknownError
	}
	defer tx.Rollback()

	//Update worker status
	if err := worker.UpdateWorkerStatus(tx, c.Worker.ID, sdk.StatusWaiting); err != nil {
		log.Warning("addQueueResultHandler> Cannot update worker status (%s): %s\n", c.Worker.ID, err)
		// We want to update pipelineBuildJob status anyway
	}

	// Update action status
	log.Debug("addQueueResultHandler> Updating %d to %s in queue\n", id, res.Status)
	if err := pipeline.UpdatePipelineBuildJobStatus(tx, pbJob, res.Status); err != nil {
		log.Warning("addQueueResultHandler> Cannot update %d status: %s\n", id, err)
		return err
	}

	infos := []sdk.SpawnInfo{{
		RemoteTime: res.RemoteTime,
		Message:    sdk.SpawnMsg{ID: sdk.SpawnInfoWorkerEnd.ID, Args: []interface{}{c.Worker.Name, res.Duration}},
	}}

	if _, err := pipeline.AddSpawnInfosPipelineBuildJob(tx, pbJob.ID, infos); err != nil {
		log.Critical("addQueueResultHandler> Cannot save spawn info job %d: %s\n", pbJob.ID, err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("addQueueResultHandler> Cannot commit tx: %s\n", err)
		return sdk.ErrUnknownError
	}

	return nil
}

func takePipelineBuildJobHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get action name in URL
	vars := mux.Vars(r)
	idString := vars["id"]

	takeForm := &worker.TakeForm{}
	if err := UnmarshalBody(r, takeForm); err != nil {
		return sdk.WrapError(err, "takePipelineBuildJobHandler> Unable to parse take form")
	}

	// Load worker
	caller, err := worker.LoadWorker(db, c.Worker.ID)
	if err != nil {
		log.Warning("takePipelineBuildJobHandler> cannot load calling worker: %s\n", err)
		return err
	}
	if caller.Status != sdk.StatusChecking {
		log.Warning("takePipelineBuildJobHandler> worker %s is not available to for build (status = %s)\n", caller.ID, caller.Status)
		return sdk.ErrWrongRequest
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		log.Info("takePipelineBuildJobHandler> Cannot start transaction: %s\n", errBegin)
		return errBegin
	}
	defer tx.Rollback()

	// Take job
	id, errInt := strconv.ParseInt(idString, 10, 64)
	if errInt != nil {
		return sdk.ErrInvalidID
	}

	workerModel := caller.Name
	if caller.Model != 0 {
		wm, errModel := worker.LoadWorkerModelByID(db, caller.Model)
		if errModel != nil {
			return sdk.ErrNoWorkerModel
		}
		workerModel = wm.Name
	}

	infos := []sdk.SpawnInfo{{
		RemoteTime: takeForm.Time,
		Message:    sdk.SpawnMsg{ID: sdk.SpawnInfoJobTaken.ID, Args: []interface{}{c.Worker.Name}},
	}}

	if takeForm.BookedJobID != 0 && takeForm.BookedJobID == id {
		infos = append(infos, sdk.SpawnInfo{
			RemoteTime: takeForm.Time,
			Message:    sdk.SpawnMsg{ID: sdk.SpawnInfoWorkerForJob.ID, Args: []interface{}{c.Worker.Name}},
		})
	}

	pbJob, errTake := pipeline.TakePipelineBuildJob(tx, id, workerModel, caller.Name, infos)
	if errTake != nil {
		if errTake != pipeline.ErrAlreadyTaken {
			log.Warning("takePipelineBuildJobHandler> Cannot give ActionBuild %d: %s\n", id, errTake)
		}
		return errTake
	}

	if err := worker.SetToBuilding(tx, c.Worker.ID, pbJob.ID); err != nil {
		log.Warning("takePipelineBuildJobHandler> Cannot update worker status: %s\n", err)
		return err
	}

	log.Debug("takePipelineBuildJobHandler>Updated %d (PipelineAction %d) to %s\n", id, pbJob.Job.PipelineActionID, sdk.StatusBuilding)

	secrets, errSecret := loadActionBuildSecrets(db, pbJob.ID)
	if errSecret != nil {
		log.Warning("takePipelineBuildJobHandler> Cannot load action build secrets: %s\n", errSecret)
		return errSecret
	}

	pb, errPb := pipeline.LoadPipelineBuildByID(db, pbJob.PipelineBuildID)
	if errPb != nil {
		log.Warning("takePipelineBuildJobHandler> Cannot get pipeline build: %s\n", errPb)
		return errPb
	}

	if err := tx.Commit(); err != nil {
		log.Info("takePipelineBuildJobHandler> Cannot commit transaction: %s\n", err)
		return err
	}

	pbji := worker.PipelineBuildJobInfo{}
	pbji.PipelineBuildJob = *pbJob
	pbji.Secrets = secrets
	pbji.PipelineID = pb.Pipeline.ID
	pbji.BuildNumber = pb.BuildNumber
	return WriteJSON(w, r, pbji, http.StatusOK)
}

func preCheckHatchery(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) (int64, error) {
	if c.Hatchery == nil {
		log.Warning("this handler should be called only by hatcheries\n")
		return 0, sdk.ErrWrongRequest
	}

	// Get action name in URL
	vars := mux.Vars(r)
	idString := vars["id"]

	// Check ID Job
	id, erri := strconv.ParseInt(idString, 10, 64)
	if erri != nil {
		return id, sdk.ErrInvalidID
	}
	return id, nil
}

func bookPipelineBuildJobHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	id, errc := preCheckHatchery(w, r, db, c)
	if errc != nil {
		return errc
	}

	if h, err := pipeline.BookPipelineBuildJob(id, c.Hatchery); err != nil {
		if err == pipeline.ErrAlreadyBooked && h != nil {
			log.Warning("bookPipelineBuildJobHandler> job %d already booked by %s (%d): %s\n", id, h.Name, h.ID)
			return WriteJSON(w, r, "job already booked", http.StatusConflict)
		}
		log.Warning("bookPipelineBuildJobHandler> Cannot book job %d: %s\n", id, err)
		return err
	}
	return WriteJSON(w, r, nil, http.StatusOK)
}

func addSpawnInfosPipelineBuildJobHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	pbJobID, errc := preCheckHatchery(w, r, db, c)
	if errc != nil {
		return errc
	}
	var s []sdk.SpawnInfo
	if err := UnmarshalBody(r, &s); err != nil {
		return err
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		log.Info("addSpawnInfosPipelineBuildJobHandler> Cannot start transaction: %s\n", errBegin)
		return errBegin
	}
	defer tx.Rollback()

	if _, err := pipeline.AddSpawnInfosPipelineBuildJob(tx, pbJobID, s); err != nil {
		log.Warning("addSpawnInfosPipelineBuildJobHandler> Cannot save job %d: %s\n", pbJobID, err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("addSpawnInfosPipelineBuildJobHandler> Cannot commit tx: %s\n", err)
		return sdk.ErrUnknownError
	}

	return WriteJSON(w, r, nil, http.StatusOK)
}

func loadActionBuildSecrets(db *gorp.DbMap, pbJobID int64) ([]sdk.Variable, error) {
	query := `SELECT pipeline.project_id, pipeline_build.application_id, pipeline_build.environment_id
	FROM pipeline_build
	JOIN pipeline_build_job ON pipeline_build_job.pipeline_build_id = pipeline_build.id
	JOIN pipeline ON pipeline.id = pipeline_build.pipeline_id
	WHERE pipeline_build_job.id = $1`

	var projectID, appID, envID int64
	var secrets []sdk.Variable
	if err := db.QueryRow(query, pbJobID).Scan(&projectID, &appID, &envID); err != nil {
		return nil, err
	}

	// Load project secrets
	pv, err := project.GetAllVariableInProject(db, projectID, project.WithClearPassword())
	if err != nil {
		return nil, err
	}
	for _, s := range pv {
		if !sdk.NeedPlaceholder(s.Type) {
			continue
		}
		if s.Value == sdk.PasswordPlaceholder {
			log.Critical("loadActionBuildSecrets> Loaded an placeholder for %s !\n", s.Name)
			return nil, fmt.Errorf("Loaded placeholder for %s\n", s.Name)
		}
		s.Name = "cds.proj." + s.Name
		secrets = append(secrets, s)
	}

	// Load application secrets
	pv, err = application.GetAllVariableByID(db, appID, application.WithClearPassword())
	if err != nil {
		return nil, err
	}
	for _, s := range pv {
		if !sdk.NeedPlaceholder(s.Type) {
			continue
		}
		if s.Value == sdk.PasswordPlaceholder {
			log.Critical("loadActionBuildSecrets> Loaded an placeholder for %s !\n", s.Name)
			return nil, fmt.Errorf("Loaded placeholder for %s\n", s.Name)
		}
		s.Name = "cds.app." + s.Name
		secrets = append(secrets, s)
	}

	// Load environment secrets
	pv, err = environment.GetAllVariableByID(db, envID, environment.WithClearPassword())
	if err != nil {
		return nil, err
	}
	for _, s := range pv {
		if !sdk.NeedPlaceholder(s.Type) {
			continue
		}
		if s.Value == sdk.PasswordPlaceholder {
			log.Critical("loadActionBuildSecrets> Loaded an placeholder for %s !\n", s.Name)
			return nil, fmt.Errorf("Loaded placeholder for %s\n", s.Name)
		}
		s.Name = "cds.env." + s.Name
		secrets = append(secrets, s)
	}

	return secrets, nil
}

func getQueueHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	if c.Worker != nil && c.Worker.ID != "" {
		// Load calling worker
		caller, errW := worker.LoadWorker(db, c.Worker.ID)
		if errW != nil {
			log.Warning("getQueueHandler> cannot load calling worker: %s\n", errW)
			return errW
		}
		if caller.Status != sdk.StatusWaiting {
			log.Info("getQueueHandler> worker %s is not available to build (status = %s)\n", caller.ID, caller.Status)
			return sdk.ErrInvalidWorkerStatus
		}
	}

	var queue []sdk.PipelineBuildJob
	var errQ error
	switch c.Agent {
	case sdk.HatcheryAgent:
		queue, errQ = pipeline.LoadGroupWaitingQueue(db, c.Hatchery.GroupID)
	case sdk.WorkerAgent:
		queue, errQ = pipeline.LoadGroupWaitingQueue(db, c.Worker.GroupID)
	default:
		queue, errQ = pipeline.LoadUserWaitingQueue(db, c.User)
	}

	if errQ != nil {
		log.Warning("getQueueHandler> Cannot load queue from db: %s\n", errQ)
		return errQ
	}

	if log.IsDebug() {
		for _, pbJob := range queue {
			log.Debug("getQueueHandler> PipelineBuildJob : %d %s [%s]", pbJob.ID, pbJob.Job.Action.Name, pbJob.Status)
		}
	}

	return WriteJSON(w, r, queue, http.StatusOK)
}

func requirementsErrorHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("requirementsErrorHandler> %s\n", err)
		return err
	}

	if c.Worker.ID != "" {
		// Load calling worker
		caller, err := worker.LoadWorker(db, c.Worker.ID)
		if err != nil {
			log.Warning("requirementsErrorHandler> cannot load calling worker: %s\n", err)
			return sdk.ErrWrongRequest
		}

		log.Warning("%s (%s) > %s", c.Worker.ID, caller.Name, string(body))
	}
	return nil
}

func addBuildVariableHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	buildNumberS := vars["build"]
	appName := vars["app"]

	var env *sdk.Environment
	envName := r.FormValue("envName")
	if envName == "" || envName == sdk.DefaultEnv.Name {
		env = &sdk.DefaultEnv
	} else {
		var err error
		env, err = environment.LoadEnvironmentByName(db, projectKey, envName)
		if err != nil {
			log.Warning("addBuildVariableHandler> Cannot load environment %s: %s\n", envName, err)
			return sdk.ErrUnknownEnv
		}

	}

	if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadExecute) {
		log.Warning("addBuildVariableHandler> No enought right on this environment %s: \n", envName)
		return sdk.ErrForbidden
	}

	// Check that pipeline exists
	p, errLP := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if errLP != nil {
		log.Warning("addBuildVariableHandler> Cannot load pipeline %s: %s\n", pipelineName, errLP)
		return errLP
	}

	// Check that application exists
	a, errLA := application.LoadByName(db, projectKey, appName, c.User)
	if errLA != nil {
		log.Warning("addBuildVariableHandler> Cannot load application %s: %s\n", appName, errLA)
		return errLA
	}

	// if buildNumber is 'last' fetch last build number
	buildNumber, errP := strconv.ParseInt(buildNumberS, 10, 64)
	if errP != nil {
		log.Warning("addBuildVariableHandler> Cannot parse build number %s: %s\n", buildNumberS, errP)
		return errP
	}

	// load pipeline_build.id
	pbID, errPB := pipeline.LoadPipelineBuildID(db, a.ID, p.ID, env.ID, buildNumber)
	if errPB != nil {
		log.Warning("addBuildVariableHandler> Cannot load pipeline build %d: %s\n", buildNumber, errPB)
		return errPB
	}

	// Unmarshal into results
	var v sdk.Variable
	if err := UnmarshalBody(r, &v); err != nil {
		return err
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		log.Warning("addBuildVariableHandler> Cannot start transaction: %s\n", errBegin)
		return errBegin
	}
	defer tx.Rollback()

	if err := pipeline.InsertBuildVariable(tx, pbID, v); err != nil {
		log.Warning("addBuildVariableHandler> Cannot add build variable: %s\n", err)
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Warning("addBuildVariableHandler> Cannot commit transaction: %s\n", err)
		return err
	}

	return nil
}

func addBuildTestResultsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	buildNumberS := vars["build"]
	appName := vars["app"]

	var env *sdk.Environment
	envName := r.FormValue("envName")
	if envName == "" || envName == sdk.DefaultEnv.Name {
		env = &sdk.DefaultEnv
	} else {
		var errle error
		env, errle = environment.LoadEnvironmentByName(db, projectKey, envName)
		if errle != nil {
			log.Warning("addBuildTestResultsHandler> Cannot load environment %s: %s\n", envName, errle)
			return sdk.ErrUnknownEnv
		}

	}

	if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadExecute) {
		log.Warning("addBuildTestResultsHandler> No enought right on this environment %s: \n", envName)
		return sdk.ErrForbidden
	}

	// Check that pipeline exists
	p, errlp := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if errlp != nil {
		log.Warning("addBuildTestResultsHandler> Cannot load pipeline %s: %s\n", pipelineName, errlp)
		return sdk.ErrNotFound
	}

	// Check that application exists
	a, errln := application.LoadByName(db, projectKey, appName, c.User)
	if errln != nil {
		log.Warning("addBuildTestResultsHandler> Cannot load application %s: %s\n", appName, errln)
		return sdk.ErrNotFound
	}

	buildNumber, errpi := strconv.ParseInt(buildNumberS, 10, 64)
	if errpi != nil {
		log.Warning("addBuildTestResultsHandler> Cannot parse build number %s: %s\n", buildNumberS, errpi)
		return errpi
	}

	// load pipeline_build.id
	pb, errl := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(db, a.ID, p.ID, env.ID, buildNumber)
	if errl != nil {
		log.Warning("addBuiltTestResultsHandler> Cannot loadpipelinebuild for %s/%s[%s] %d: %s\n", a.Name, p.Name, envName, buildNumber, errl)
		return errl
	}

	// Unmarshal into results
	var new venom.Tests
	if err := UnmarshalBody(r, &new); err != nil {
		return err
	}

	// Load existing and merge
	tests, err := pipeline.LoadTestResults(db, pb.ID)
	if err != nil {
		log.Warning("addBuildTestResultsHandler> Cannot load test results: %s\n", err)
		return err
	}

	for k := range new.TestSuites {
		for i := range tests.TestSuites {
			if tests.TestSuites[i].Name == new.TestSuites[k].Name {
				// testsuite with same name already exists,
				// Create a unique name
				new.TestSuites[k].Name = fmt.Sprintf("%s.%d", new.TestSuites[k].Name, pb.ID)
				break
			}
		}
		tests.TestSuites = append(tests.TestSuites, new.TestSuites[k])
	}

	// update total values
	tests.Total = 0
	tests.TotalOK = 0
	tests.TotalKO = 0
	tests.TotalSkipped = 0
	for _, ts := range tests.TestSuites {
		tests.Total += ts.Total
		tests.TotalKO += ts.Failures + ts.Errors
		tests.TotalOK += ts.Total - ts.Skipped - ts.Failures - ts.Errors
		tests.TotalSkipped += ts.Skipped
	}

	if err := pipeline.UpdateTestResults(db, pb.ID, tests); err != nil {
		log.Warning("addBuildTestsResultsHandler> Cannot insert tests results: %s\n", err)
		return err
	}

	stats.TestEvent(db, p.ProjectID, a.ID, tests)
	return nil
}

func getBuildTestResultsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {

	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	buildNumberS := vars["build"]
	appName := vars["app"]

	var err error
	var env *sdk.Environment
	envName := r.FormValue("envName")
	if envName == "" || envName == sdk.DefaultEnv.Name {
		env = &sdk.DefaultEnv
	} else {
		env, err = environment.LoadEnvironmentByName(db, projectKey, envName)
		if err != nil {
			log.Warning("getBuildTestResultsHandler> Cannot load environment %s: %s\n", envName, err)
			return sdk.ErrUnknownEnv
		}

	}

	if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionRead) {
		log.Warning("getBuildTestResultsHandler> No enought right on this environment %s: \n", envName)
		return sdk.ErrForbidden
	}

	// Check that pipeline exists
	p, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		log.Warning("getBuildTestResultsHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
		return sdk.ErrNotFound
	}

	// Check that application exists
	a, err := application.LoadByName(db, projectKey, appName, c.User)
	if err != nil {
		log.Warning("getBuildTestResultsHandler> Cannot load application %s: %s\n", appName, err)
		return sdk.ErrNotFound
	}

	// if buildNumber is 'last' fetch last build number
	var buildNumber int64
	if buildNumberS == "last" {
		var errlb error
		bn, errlb := pipeline.GetLastBuildNumberInTx(db, p.ID, a.ID, env.ID)
		if errlb != nil {
			log.Warning("getBuildTestResultsHandler> Cannot load last build number for %s: %s\n", pipelineName, errlb)
			return sdk.ErrNoPipelineBuild
		}
		buildNumber = bn
	} else {
		var errpi error
		buildNumber, errpi = strconv.ParseInt(buildNumberS, 10, 64)
		if errpi != nil {
			log.Warning("getBuildTestResultsHandler> Cannot parse build number %s: %s\n", buildNumberS, errpi)
			return errpi
		}
	}

	// load pipeline_build.id
	pb, errlpb := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(db, a.ID, p.ID, env.ID, buildNumber)
	if errlpb != nil {
		log.Warning("getBuildTestResultsHandler> Cannot load pipeline build: %s\n", errlpb)
		return errlpb
	}

	tests, errltr := pipeline.LoadTestResults(db, pb.ID)
	if errltr != nil {
		log.Warning("getBuildTestResultsHandler> Cannot load test results: %s\n", errltr)
		return errltr
	}

	return WriteJSON(w, r, tests, http.StatusOK)
}
