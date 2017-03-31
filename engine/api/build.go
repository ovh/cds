package main

import (
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
	buildID, errr := requestVarInt(r, "id")
	if errr != nil {
		return sdk.WrapError(errr, "updateStepStatusHandler> Invalid id")
	}

	pbJob, errJob := pipeline.GetPipelineBuildJob(db, buildID)
	if errJob != nil {
		return sdk.WrapError(errJob, "updateStepStatusHandler> Cannot get pipeline build job %d", buildID)
	}

	var step sdk.StepStatus
	if err := UnmarshalBody(r, &step); err != nil {
		return sdk.WrapError(err, "updateStepStatusHandler> Error while unmarshal job")
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

	if err := pipeline.UpdatePipelineBuildJob(db, pbJob); err != nil {
		log.Warning("updateStepStatusHandler> Cannot update pipeline build job: %s", err)
		return err
	}
	return nil
}

func getPipelineBuildTriggeredHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	appName := vars["permApplicationName"]

	envName := r.FormValue("envName")

	buildNumber, err := requestVarInt(r, "build")
	if err != nil {
		return sdk.WrapError(err, "getPipelineBuildTriggeredHandler> invalid build number")
	}

	// Load Pipeline
	p, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		return sdk.WrapError(err, "getPipelineBuildTriggeredHandler> Cannot load pipeline %s", pipelineName)
	}

	// Load Application
	a, err := application.LoadByName(db, projectKey, appName, c.User)
	if err != nil {
		return sdk.WrapError(err, "getPipelineBuildTriggeredHandler> Cannot load application %s", appName)
	}

	// Load Env
	env := &sdk.DefaultEnv
	if envName != sdk.DefaultEnv.Name && envName != "" {
		env, err = environment.LoadEnvironmentByName(db, projectKey, envName)
		if err != nil {
			return sdk.WrapError(err, "getPipelineBuildTriggeredHandler> Cannot load environment %s", envName)
		}
	}

	// Load Children
	pbs, err := pipeline.LoadPipelineBuildChildren(db, p.ID, a.ID, buildNumber, env.ID)
	if err != nil {
		return sdk.WrapError(sdk.ErrNoPipelineBuild, "getPipelineBuildTriggeredHandler> Cannot load pipeline build children: %s", err)
	}
	return WriteJSON(w, r, pbs, http.StatusOK)
}

func deleteBuildHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	appName := vars["permApplicationName"]

	envName := r.FormValue("envName")

	buildNumber, err := requestVarInt(r, "build")
	if err != nil {
		return sdk.WrapError(err, "getPipelineBuildTriggeredHandler> invalid build number")
	}

	// Check that pipeline exists
	p, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		return sdk.WrapError(err, "deleteBuildHandler> Cannot load pipeline %s", pipelineName)
	}

	a, err := application.LoadByName(db, projectKey, appName, c.User)
	if err != nil {
		return sdk.WrapError(err, "deleteBuildHandler> Cannot load application %s", appName)
	}

	var env *sdk.Environment
	if envName == "" || envName == sdk.DefaultEnv.Name {
		env = &sdk.DefaultEnv
	} else {
		env, err = environment.LoadEnvironmentByName(db, projectKey, envName)
		if err != nil {
			return sdk.WrapError(err, "deleteBuildHandler> Cannot load environment %s", envName)
		}
	}

	if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionRead) {
		return sdk.WrapError(sdk.ErrForbidden, "deleteBuildHandler> No enought right on this environment %s", envName)
	}

	pbID, errPB := pipeline.LoadPipelineBuildID(db, a.ID, p.ID, env.ID, buildNumber)
	if errPB != nil {
		return sdk.WrapError(errPB, "deleteBuildHandler> Cannot load pipeline build")
	}

	tx, err := db.Begin()
	if err != nil {
		return sdk.WrapError(err, "deleteBuildHandler> Cannot start transaction")
	}
	defer tx.Rollback()

	if err := pipeline.DeletePipelineBuildByID(tx, pbID); err != nil {
		return sdk.WrapError(sdk.ErrUnknownError, "deleteBuildHandler> Cannot delete pipeline build: %s", err)
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "deleteBuildHandler> Cannot commit transaction")
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
		return sdk.WrapError(err, "getBuildStateHandler> Cannot load pipeline %s", pipelineName)
	}

	a, err := application.LoadByName(db, projectKey, appName, c.User)
	if err != nil {
		return sdk.WrapError(err, "getBuildStateHandler> Cannot load application %s", appName)
	}

	var env *sdk.Environment
	if envName == "" || envName == sdk.DefaultEnv.Name {
		env = &sdk.DefaultEnv
	} else {
		env, err = environment.LoadEnvironmentByName(db, projectKey, envName)
		if err != nil {
			return sdk.WrapError(err, "getBuildStateHandler> Cannot load environment %s", envName)
		}
	}

	if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionRead) {
		return sdk.WrapError(sdk.ErrForbidden, "getBuildStateHandler> No enought right on this environment %s: ", envName)
	}

	// if buildNumber is 'last' fetch last build number
	var buildNumber int64
	if buildNumberS == "last" {
		lastBuildNumber, errg := pipeline.GetLastBuildNumberInTx(db, p.ID, a.ID, env.ID)
		if errg != nil {
			return sdk.WrapError(sdk.ErrNotFound, "getBuildStateHandler> Cannot load last pipeline build number for %s-%s-%s: %s", a.Name, pipelineName, env.Name, errg)
		}
		buildNumber = lastBuildNumber
	} else {
		buildNumber, err = strconv.ParseInt(buildNumberS, 10, 64)
		if err != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "getBuildStateHandler> Cannot parse build number %s: %s", buildNumberS, err)
		}
	}

	// load pipeline_build.id
	pb, err := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(db, a.ID, p.ID, env.ID, buildNumber)
	if err != nil {
		return sdk.WrapError(err, "getBuildStateHandler> %s! Cannot load last pipeline build for %s-%s-%s[%s] (buildNUmber:%d)", c.User.Username, projectKey, appName, pipelineName, env.Name, buildNumber)
	}

	if withArtifacts == "true" {
		var errLoadArtifact error
		pb.Artifacts, errLoadArtifact = artifact.LoadArtifactsByBuildNumber(db, p.ID, a.ID, buildNumber, env.ID)
		if errLoadArtifact != nil {
			return sdk.WrapError(errLoadArtifact, "getBuildStateHandler> Cannot load artifacts")
		}
	}

	if withTests == "true" {
		tests, errLoadTests := pipeline.LoadTestResults(db, pb.ID)
		if errLoadTests != nil {
			return sdk.WrapError(errLoadTests, "getBuildStateHandler> Cannot load tests")
		}
		if len(tests.TestSuites) > 0 {
			pb.Tests = &tests
		}
	}
	pb.Translate(r.Header.Get("Accept-Language"))

	return WriteJSON(w, r, pb, http.StatusOK)
}

func addQueueResultHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	id, errc := requestVarInt(r, "id")
	if errc != nil {
		return sdk.WrapError(errc, "addQueueResultHandler> invalid id")
	}

	// Load Build
	pbJob, errJob := pipeline.GetPipelineBuildJob(db, id)
	if errJob != nil {
		return sdk.WrapError(sdk.ErrNotFound, "addQueueResultHandler> Cannot load queue (%d) from db: %s", id, errJob)
	}

	// Unmarshal into results
	var res sdk.Result
	if err := UnmarshalBody(r, &res); err != nil {
		return sdk.WrapError(err, "addQueueResultHandler> cannot unmarshal request")
	}

	tx, errb := db.Begin()
	if errb != nil {
		return sdk.WrapError(errb, "addQueueResultHandler> Cannot begin tx")
	}
	defer tx.Rollback()

	//Update worker status
	if err := worker.UpdateWorkerStatus(tx, c.Worker.ID, sdk.StatusWaiting); err != nil {
		log.Warning("addQueueResultHandler> Cannot update worker status (%s): %s", c.Worker.ID, err)
		// We want to update pipelineBuildJob status anyway
	}

	// Update action status
	log.Debug("addQueueResultHandler> Updating %d to %s in queue", id, res.Status)
	if err := pipeline.UpdatePipelineBuildJobStatus(tx, pbJob, res.Status); err != nil {
		return sdk.WrapError(err, "addQueueResultHandler> Cannot update %d status", id)
	}

	infos := []sdk.SpawnInfo{{
		RemoteTime: res.RemoteTime,
		Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoWorkerEnd.ID, Args: []interface{}{c.Worker.Name, res.Duration}},
	}}

	if _, err := pipeline.AddSpawnInfosPipelineBuildJob(tx, pbJob.ID, infos); err != nil {
		log.Critical("addQueueResultHandler> Cannot save spawn info job %d: %s", pbJob.ID, err)
		return err
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "addQueueResultHandler> Cannot commit tx")
	}

	return nil
}

func takePipelineBuildJobHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	id, errc := requestVarInt(r, "id")
	if errc != nil {
		return sdk.WrapError(errc, "takePipelineBuildJobHandler> invalid id")
	}

	takeForm := &worker.TakeForm{}
	if err := UnmarshalBody(r, takeForm); err != nil {
		return sdk.WrapError(err, "takePipelineBuildJobHandler> cannot unmarshal request")
	}

	// Load worker
	caller, err := worker.LoadWorker(db, c.Worker.ID)
	if err != nil {
		return sdk.WrapError(err, "takePipelineBuildJobHandler> cannot load calling worker")
	}
	if caller.Status != sdk.StatusChecking {
		return sdk.WrapError(sdk.ErrWrongRequest, "takePipelineBuildJobHandler> worker %s is not available to for build (status = %s)", caller.ID, caller.Status)
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		return sdk.WrapError(errBegin, "takePipelineBuildJobHandler> Cannot start transaction")
	}
	defer tx.Rollback()

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
		Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoJobTaken.ID, Args: []interface{}{c.Worker.Name}},
	}}

	if takeForm.BookedJobID != 0 && takeForm.BookedJobID == id {
		infos = append(infos, sdk.SpawnInfo{
			RemoteTime: takeForm.Time,
			Message:    sdk.SpawnMsg{ID: sdk.MsgSpawnInfoWorkerForJob.ID, Args: []interface{}{c.Worker.Name}},
		})
	}

	pbJob, errTake := pipeline.TakePipelineBuildJob(tx, id, workerModel, caller.Name, infos)
	if errTake != nil {
		return sdk.WrapError(errTake, "takePipelineBuildJobHandler> Cannot take job %d", id)
	}

	if err := worker.SetToBuilding(tx, c.Worker.ID, pbJob.ID); err != nil {
		return sdk.WrapError(err, "takePipelineBuildJobHandler> Cannot update worker status")
	}

	secrets, errSecret := loadActionBuildSecrets(db, pbJob.ID)
	if errSecret != nil {
		return sdk.WrapError(errSecret, "takePipelineBuildJobHandler> Cannot load action build secrets")
	}

	pb, errPb := pipeline.LoadPipelineBuildByID(db, pbJob.PipelineBuildID)
	if errPb != nil {
		return sdk.WrapError(errPb, "takePipelineBuildJobHandler> Cannot get pipeline build")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "takePipelineBuildJobHandler> Cannot commit transaction")
	}

	pbji := worker.PipelineBuildJobInfo{}
	pbji.PipelineBuildJob = *pbJob
	pbji.Secrets = secrets
	pbji.PipelineID = pb.Pipeline.ID
	pbji.BuildNumber = pb.BuildNumber
	return WriteJSON(w, r, pbji, http.StatusOK)
}

func bookPipelineBuildJobHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	id, errc := requestVarInt(r, "id")
	if errc != nil {
		return sdk.WrapError(errc, "bookPipelineBuildJobHandler> invalid id")
	}

	if _, err := pipeline.BookPipelineBuildJob(id, c.Hatchery); err != nil {
		return sdk.WrapError(err, "bookPipelineBuildJobHandler> job already booked")
	}
	return WriteJSON(w, r, nil, http.StatusOK)
}

func addSpawnInfosPipelineBuildJobHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	pbJobID, errc := requestVarInt(r, "id")
	if errc != nil {
		return sdk.WrapError(errc, "addSpawnInfosPipelineBuildJobHandler> invalid id")
	}
	var s []sdk.SpawnInfo
	if err := UnmarshalBody(r, &s); err != nil {
		return sdk.WrapError(err, "addSpawnInfosPipelineBuildJobHandler> cannot unmarshal request")
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		return sdk.WrapError(errBegin, "addSpawnInfosPipelineBuildJobHandler> Cannot start transaction")
	}
	defer tx.Rollback()

	if _, err := pipeline.AddSpawnInfosPipelineBuildJob(tx, pbJobID, s); err != nil {
		return sdk.WrapError(err, "addSpawnInfosPipelineBuildJobHandler> Cannot save job %d", pbJobID)
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "addSpawnInfosPipelineBuildJobHandler> Cannot commit tx")
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
			log.Critical("loadActionBuildSecrets> Loaded an placeholder for %s !", s.Name)
			return nil, fmt.Errorf("Loaded placeholder for %s", s.Name)
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
			log.Critical("loadActionBuildSecrets> Loaded an placeholder for %s !", s.Name)
			return nil, fmt.Errorf("Loaded placeholder for %s", s.Name)
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
			log.Critical("loadActionBuildSecrets> Loaded an placeholder for %s !", s.Name)
			return nil, fmt.Errorf("Loaded placeholder for %s", s.Name)
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
			return sdk.WrapError(errW, "getQueueHandler> cannot load calling worker")
		}
		if caller.Status != sdk.StatusWaiting {
			return sdk.WrapError(sdk.ErrInvalidWorkerStatus, "getQueueHandler> worker %s is not available to build (status = %s)", caller.ID, caller.Status)
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

	lang := r.Header.Get("Accept-Language")
	for p := range queue {
		queue[p].Translate(lang)
	}

	if errQ != nil {
		return sdk.WrapError(errQ, "getQueueHandler> Cannot load queue from db: %s", errQ)
	}

	return WriteJSON(w, r, queue, http.StatusOK)
}

func requirementsErrorHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("requirementsErrorHandler> %s", err)
		return err
	}

	if c.Worker.ID != "" {
		// Load calling worker
		caller, err := worker.LoadWorker(db, c.Worker.ID)
		if err != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "requirementsErrorHandler> cannot load calling worker: %s", err)
		}

		log.Warning("%s (%s) > %s", c.Worker.ID, caller.Name, string(body))
	}
	return nil
}

func addBuildVariableHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	appName := vars["app"]

	buildNumber, errInt := requestVarInt(r, "build")
	if errInt != nil {
		return sdk.WrapError(errInt, "addBuildTestResultsHandler> invalid build number")
	}

	var env *sdk.Environment
	envName := r.FormValue("envName")
	if envName == "" || envName == sdk.DefaultEnv.Name {
		env = &sdk.DefaultEnv
	} else {
		var err error
		env, err = environment.LoadEnvironmentByName(db, projectKey, envName)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnknownEnv, "addBuildVariableHandler> Cannot load environment %s: %s", envName, err)
		}
	}

	if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadExecute) {
		return sdk.WrapError(sdk.ErrForbidden, "addBuildVariableHandler> No enought right on this environment %s", envName)
	}

	// Check that pipeline exists
	p, errLP := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if errLP != nil {
		return sdk.WrapError(errLP, "addBuildVariableHandler> Cannot load pipeline %s", pipelineName)
	}

	// Check that application exists
	a, errLA := application.LoadByName(db, projectKey, appName, c.User)
	if errLA != nil {
		return sdk.WrapError(errLA, "addBuildVariableHandler> Cannot load application %s", appName)
	}

	// load pipeline_build.id
	pbID, errPB := pipeline.LoadPipelineBuildID(db, a.ID, p.ID, env.ID, buildNumber)
	if errPB != nil {
		return sdk.WrapError(errPB, "addBuildVariableHandler> Cannot load pipeline build %d", buildNumber)
	}

	// Unmarshal into results
	var v sdk.Variable
	if err := UnmarshalBody(r, &v); err != nil {
		return sdk.WrapError(err, "addBuildVariableHandler> cannot unmarshal request")
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		return sdk.WrapError(errBegin, "addBuildVariableHandler> Cannot start transaction")
	}
	defer tx.Rollback()

	if err := pipeline.InsertBuildVariable(tx, pbID, v); err != nil {
		return sdk.WrapError(err, "addBuildVariableHandler> Cannot add build variable")
	}

	if err := tx.Commit(); err != nil {
		return sdk.WrapError(err, "addBuildVariableHandler> Cannot commit transaction")
	}

	return nil
}

func addBuildTestResultsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	appName := vars["app"]

	buildNumber, errInt := requestVarInt(r, "build")
	if errInt != nil {
		return sdk.WrapError(errInt, "addBuildTestResultsHandler> invalid build number")
	}

	var env *sdk.Environment
	envName := r.FormValue("envName")
	if envName == "" || envName == sdk.DefaultEnv.Name {
		env = &sdk.DefaultEnv
	} else {
		var errle error
		env, errle = environment.LoadEnvironmentByName(db, projectKey, envName)
		if errle != nil {
			return sdk.WrapError(errle, "addBuildTestResultsHandler> Cannot load environment %s", envName)
		}
	}

	if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadExecute) {
		return sdk.WrapError(sdk.ErrForbidden, "addBuildTestResultsHandler> No enought right on this environment %s: ", envName)
	}

	// Check that pipeline exists
	p, errlp := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if errlp != nil {
		return sdk.WrapError(errlp, "addBuildTestResultsHandler> Cannot load pipeline %s", pipelineName)
	}

	// Check that application exists
	a, errln := application.LoadByName(db, projectKey, appName, c.User)
	if errln != nil {
		return sdk.WrapError(errln, "addBuildTestResultsHandler> Cannot load application %s", appName)
	}

	// load pipeline_build.id
	pb, errl := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(db, a.ID, p.ID, env.ID, buildNumber)
	if errl != nil {
		return sdk.WrapError(errl, "addBuiltTestResultsHandler> Cannot loadpipelinebuild for %s/%s[%s] %d", a.Name, p.Name, envName, buildNumber)
	}

	// Unmarshal into results
	var new venom.Tests
	if err := UnmarshalBody(r, &new); err != nil {
		return sdk.WrapError(err, "addBuildVariableHandler> cannot unmarshal request")
	}

	// Load existing and merge
	tests, err := pipeline.LoadTestResults(db, pb.ID)
	if err != nil {
		return sdk.WrapError(err, "addBuildTestResultsHandler> Cannot load test results")
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
		return sdk.WrapError(err, "addBuildTestsResultsHandler> Cannot insert tests results")
	}

	stats.TestEvent(db, p.ProjectID, a.ID, tests)
	return nil
}

func getBuildTestResultsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
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
			return sdk.WrapError(sdk.ErrUnknownEnv, "getBuildTestResultsHandler> Cannot load environment %s: %s", envName, err)
		}
	}

	if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionRead) {
		return sdk.WrapError(sdk.ErrForbidden, "getBuildTestResultsHandler> No enought right on this environment %s: ", envName)
	}

	// Check that pipeline exists
	p, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		return sdk.WrapError(err, "getBuildTestResultsHandler> Cannot load pipeline %s", pipelineName)
	}

	// Check that application exists
	a, err := application.LoadByName(db, projectKey, appName, c.User)
	if err != nil {
		return sdk.WrapError(err, "getBuildTestResultsHandler> Cannot load application %s", appName)
	}

	// if buildNumber is 'last' fetch last build number
	var buildNumber int64
	if buildNumberS == "last" {
		var errlb error
		bn, errlb := pipeline.GetLastBuildNumberInTx(db, p.ID, a.ID, env.ID)
		if errlb != nil {
			return sdk.WrapError(sdk.ErrNoPipelineBuild, "getBuildTestResultsHandler> Cannot load last build number for %s: %s", pipelineName, errlb)
		}
		buildNumber = bn
	} else {
		var errpi error
		buildNumber, errpi = strconv.ParseInt(buildNumberS, 10, 64)
		if errpi != nil {
			return sdk.WrapError(errpi, "getBuildTestResultsHandler> Cannot parse build number %s", buildNumberS)
		}
	}

	// load pipeline_build.id
	pb, errlpb := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(db, a.ID, p.ID, env.ID, buildNumber)
	if errlpb != nil {
		return sdk.WrapError(errlpb, "getBuildTestResultsHandler> Cannot load pipeline build")
	}

	tests, errltr := pipeline.LoadTestResults(db, pb.ID)
	if errltr != nil {
		return sdk.WrapError(errltr, "getBuildTestResultsHandler> Cannot load test results")
	}

	return WriteJSON(w, r, tests, http.StatusOK)
}
