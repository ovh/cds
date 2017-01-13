package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

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

func getPipelineBuildTriggeredHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	buildNumberS := vars["build"]
	appName := vars["permApplicationName"]

	envName := r.FormValue("envName")

	buildNumber, err := strconv.ParseInt(buildNumberS, 10, 64)
	if err != nil {
		log.Warning("getPipelineBuildTriggeredHandler> BuildNumber %s is not an integer: %s\n", buildNumberS, err)
		WriteError(w, r, err)
		return
	}

	// Load Pipeline
	p, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		log.Warning("getPipelineBuildTriggeredHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
		WriteError(w, r, sdk.ErrPipelineNotFound)
		return
	}

	// Load Application
	a, err := application.LoadApplicationByName(db, projectKey, appName)
	if err != nil {
		log.Warning("getPipelineBuildTriggeredHandler> Cannot load application %s: %s\n", appName, err)
		WriteError(w, r, sdk.ErrApplicationNotFound)
		return
	}

	// Load Env
	env := &sdk.DefaultEnv
	if envName != sdk.DefaultEnv.Name && envName != "" {
		env, err = environment.LoadEnvironmentByName(db, projectKey, envName)
		if err != nil {
			log.Warning("getPipelineBuildTriggeredHandler> Cannot load environment %s: %s\n", envName, err)
			WriteError(w, r, sdk.ErrNoEnvironment)
			return
		}
	}

	// Load Children
	pbs, err := pipeline.LoadPipelineBuildChildren(db, p.ID, a.ID, buildNumber, env.ID)
	if err != nil {
		log.Warning("getPipelineBuildTriggeredHandler> Cannot load pipeline build children: %s\n", err)
		WriteError(w, r, sdk.ErrNoPipelineBuild)
		return
	}
	WriteJSON(w, r, pbs, http.StatusOK)

}

func deleteBuildHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {
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
		WriteError(w, r, sdk.ErrPipelineNotFound)
		return
	}

	a, err := application.LoadApplicationByName(db, projectKey, appName)
	if err != nil {
		log.Warning("deleteBuildHandler> Cannot load application %s: %s\n", appName, err)
		WriteError(w, r, sdk.ErrApplicationNotFound)
		return
	}

	var env *sdk.Environment
	if envName == "" || envName == sdk.DefaultEnv.Name {
		env = &sdk.DefaultEnv
	} else {
		env, err = environment.LoadEnvironmentByName(db, projectKey, envName)
		if err != nil {
			log.Warning("deleteBuildHandler> Cannot load environment %s: %s\n", envName, err)
			WriteError(w, r, sdk.ErrUnknownEnv)
			return
		}
	}

	if env.ID != sdk.DefaultEnv.ID && !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionRead) {
		log.Warning("deleteBuildHandler> No enought right on this environment %s: \n", envName)
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	var buildNumber int64
	buildNumber, err = strconv.ParseInt(buildNumberS, 10, 64)
	if err != nil {
		log.Warning("deleteBuildHandler> Cannot parse build number %s: %s\n", buildNumberS, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("deleteBuildHandler> Cannot start transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	if err := pipeline.DeletePipelineBuild(tx, p.ID, a.ID, buildNumber, env.ID); err != nil {
		log.Warning("deleteBuildHandler> Cannot delete pipeline build: %s\n", err)
		WriteError(w, r, err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Warning("deleteBuildHandler> Cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getBuildStateHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {
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
		WriteError(w, r, sdk.ErrPipelineNotFound)
		return
	}

	a, err := application.LoadApplicationByName(db, projectKey, appName)
	if err != nil {
		log.Warning("getBuildStateHandler> Cannot load application %s: %s\n", appName, err)
		WriteError(w, r, sdk.ErrApplicationNotFound)
		return
	}

	var env *sdk.Environment
	if envName == "" || envName == sdk.DefaultEnv.Name {
		env = &sdk.DefaultEnv
	} else {
		env, err = environment.LoadEnvironmentByName(db, projectKey, envName)
		if err != nil {
			log.Warning("getBuildStateHandler> Cannot load environment %s: %s\n", envName, err)
			WriteError(w, r, sdk.ErrUnknownEnv)
			return
		}
	}

	if env.ID != sdk.DefaultEnv.ID && !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionRead) {
		log.Warning("getBuildStateHandler> No enought right on this environment %s: \n", envName)
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	// if buildNumber is 'last' fetch last build number
	var buildNumber int64
	if buildNumberS == "last" {
		lastBuildNumber, err := pipeline.GetLastBuildNumberInTx(db, p.ID, a.ID, env.ID)
		if err != nil {
			log.Warning("getBuildStateHandler> Cannot load last pipeline build number for %s-%s-%s: %s\n", a.Name, pipelineName, env.Name, err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		buildNumber = lastBuildNumber
	} else {
		buildNumber, err = strconv.ParseInt(buildNumberS, 10, 64)
		if err != nil {
			log.Warning("getBuildStateHandler> Cannot parse build number %s: %s\n", buildNumberS, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	// load pipeline_build.id
	pb, err := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(db, a.ID, p.ID, env.ID, buildNumber)
	if err != nil {
		log.Warning("getBuildStateHandler> %s! Cannot load last pipeline build for %s-%s-%s[%s] (buildNUmber:%d): %s\n", c.User.Username, projectKey, appName, pipelineName, env.Name, buildNumber, err)
		WriteError(w, r, err)
		return
	}

	for index := range pb.Stages {
		stage := &pb.Stages[index]
		for indexPbJob := range stage.PipelineBuildJobs {
			_ = &stage.PipelineBuildJobs[indexPbJob]

			// TODO GET LOGS
		}
	}

	if withArtifacts == "true" {
		var errLoadArtifact error
		pb.Artifacts, errLoadArtifact = artifact.LoadArtifactsByBuildNumber(db, p.ID, a.ID, buildNumber, env.ID)
		if errLoadArtifact != nil {
			log.Warning("getBuildStateHandler> Cannot load artifacts: %s", errLoadArtifact)
			WriteError(w, r, errLoadArtifact)
			return
		}
	}

	if withTests == "true" {
		tests, errLoadTests := pipeline.LoadTestResults(db, pb.ID)
		if errLoadTests != nil {
			log.Warning("getBuildStateHandler> Cannot load tests: %s", errLoadTests)
			WriteError(w, r, errLoadTests)
			return
		}
		if len(tests.TestSuites) > 0 {
			pb.Tests = &tests
		}
	}

	WriteJSON(w, r, pb, http.StatusOK)
}

func addQueueResultHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {

	// Get action name in URL
	vars := mux.Vars(r)
	idString := vars["id"]

	id, errInt := strconv.ParseInt(idString, 10, 64)
	if errInt != nil {
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}

	// Load Build
	pbJob, errJob := pipeline.GetPipelineBuildJob(db, id)
	if errJob != nil {
		log.Warning("addQueueResultHandler> Cannot load queue from db: %s\n", errJob)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("addQueueResultHandler> Cannot read body: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Unmarshal into results
	var res sdk.Result
	err = json.Unmarshal([]byte(data), &res)
	if err != nil {
		log.Warning("addQueueResultHandler> Cannot unmarshal Result: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Warning("addQueueResultHandler> Cannot begin tx: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}
	defer tx.Rollback()

	//Update worker status
	err = worker.UpdateWorkerStatus(tx, c.Worker.ID, sdk.StatusWaiting)
	if err != nil {
		log.Warning("addQueueResultHandler> Cannot update worker status (%s): %s\n", c.Worker.ID, err)
		// We want to update ActionBuild status anyway
	}

	// Update action status
	log.Debug("Updating %s to %s in queue\n", id, res.Status)
	err = pipeline.UpdatePipelineBuildJobStatus(tx, pbJob, res.Status)
	if err != nil {
		log.Warning("addQueueResultHandler> Cannot update %d status: %s\n", id, err)
		WriteError(w, r, err)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("addQueueResultHandler> Cannot commit tx: %s\n", err)
		WriteError(w, r, sdk.ErrUnknownError)
		return
	}

}

func takeActionBuildHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {

	// Get action name in URL
	vars := mux.Vars(r)
	idString := vars["id"]

	// Load worker
	caller, err := worker.LoadWorker(db, c.Worker.ID)
	if err != nil {
		log.Warning("takeActionBuildHandler> cannot load calling worker: %s\n", err)
		WriteError(w, r, err)
		return
	}
	if caller.Status != sdk.StatusChecking {
		log.Info("takeActionBuildHandler> worker %s is not available to for build (status = %s)\n", caller.ID, caller.Status)
		WriteError(w, r, sdk.ErrWrongRequest)
		return
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		log.Info("takeActionBuildHandler> Cannot start transaction: %s\n", errBegin)
		WriteError(w, r, errBegin)
		return
	}
	defer tx.Rollback()

	// Take job
	id, errInt := strconv.ParseInt(idString, 10, 64)
	if errInt != nil {
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}
	pbJob, errTake := pipeline.TakeActionBuild(tx, id, caller)
	if errTake != nil {
		if errTake != pipeline.ErrAlreadyTaken {
			log.Warning("takeActionBuildHandler> Cannot give ActionBuild %s: %s\n", "github.com/go-gorp/gorp", err)
		}
		WriteError(w, r, errTake)
		return
	}

	if err := worker.SetToBuilding(tx, c.Worker.ID, pbJob.ID); err != nil {
		log.Warning("takeActionBuildHandler> Cannot update worker status: %s\n", err)
		WriteError(w, r, err)
		return
	}

	log.Debug("Updated %s (PipelineAction %d) to %s\n", id, pbJob.Job.PipelineActionID, sdk.StatusBuilding)

	secrets, errSecret := loadActionBuildSecrets(db, pbJob.ID)
	if errSecret != nil {
		log.Warning("takeActionBuildHandler> Cannot load action build secrets: %s\n", errSecret)
		WriteError(w, r, errSecret)
		return
	}

	pb, errPb := pipeline.LoadPipelineBuildByID(db, pbJob.PipelineBuildID)
	if errPb != nil {
		log.Warning("takeActionBuildHandler> Cannot get pipeline build: %s\n", errPb)
		WriteError(w, r, errPb)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Info("takeActionBuildHandler> Cannot commit transaction: %s\n", err)
		WriteError(w, r, errBegin)
		return
	}

	pbji := worker.PipelineBuildJobInfo{}
	pbji.PipelineBuildJob = *pbJob
	pbji.Secrets = secrets
	pbji.PipelineID = pb.Pipeline.ID
	pbji.BuildNumber = pb.BuildNumber
	WriteJSON(w, r, pbji, http.StatusOK)
}

func loadActionBuildSecrets(db *gorp.DbMap, pbJobID int64) ([]sdk.Variable, error) {

	query := `SELECT pipeline.project_id, pipeline_build.application_id, pipeline_build.environment_id
	FROM pipeline_build
	JOIN pipeline_build_job ON pipeline_build_job.pipeline_build_id = pipeline_build.id
	JOIN pipeline ON pipeline.id = pipeline_build.pipeline_id
	WHERE pipeline_build_job.id = $1`

	var projectID, appID, envID int64
	var secrets []sdk.Variable
	err := db.QueryRow(query, pbJobID).Scan(&projectID, &appID, &envID)
	if err != nil {
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

func getQueueHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {
	if c.Worker.ID != "" {
		// Load calling worker
		caller, errW := worker.LoadWorker(db, c.Worker.ID)
		if errW != nil {
			log.Warning("getQueueHandler> cannot load calling worker: %s\n", errW)
			WriteError(w, r, errW)
			return
		}
		if caller.Status != sdk.StatusChecking {
			log.Warning("getQueueHandler> worker %s is not available to build (status = %s)\n", caller.ID, caller.Status)
			WriteError(w, r, sdk.ErrInvalidID)
			return
		}
	}

	var queue []sdk.PipelineBuildJob
	var errQ error
	switch c.Agent {
	case sdk.HatcheryAgent, sdk.WorkerAgent:
		queue, errQ = pipeline.LoadGroupWaitingQueue(db, c.Worker.GroupID)
	default:
		queue, errQ = pipeline.LoadUserWaitingQueue(db, c.User)
	}

	if errQ != nil {
		log.Warning("getQueueHandler> Cannot load queue from db: %s\n", errQ)
		WriteError(w, r, errQ)
		return
	}

	if log.IsDebug() {
		for _, pbJob := range queue {
			log.Debug("getQueueHandler> PipelineBuildJob : %d %s [%s]", pbJob.ID, pbJob.Job.Action.Name, pbJob.Status)
		}
	}

	WriteJSON(w, r, queue, http.StatusOK)
}

func requirementsErrorHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("requirementsErrorHandler> %s\n", err)
		WriteError(w, r, err)
		return
	}

	if c.Worker.ID != "" {
		// Load calling worker
		caller, err := worker.LoadWorker(db, c.Worker.ID)
		if err != nil {
			log.Warning("requirementsErrorHandler> cannot load calling worker: %s\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		log.Warning("%s (%s) > %s", c.Worker.ID, caller.Name, string(body))
	}
}

func addBuildVariableHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {
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
			WriteError(w, r, sdk.ErrUnknownEnv)
			return
		}

	}

	if env.ID != sdk.DefaultEnv.ID && !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadExecute) {
		log.Warning("addBuildVariableHandler> No enought right on this environment %s: \n", envName)
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	// Check that pipeline exists
	p, errLP := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if errLP != nil {
		log.Warning("addBuildVariableHandler> Cannot load pipeline %s: %s\n", pipelineName, errLP)
		WriteError(w, r, errLP)
		return
	}

	// Check that application exists
	a, errLA := application.LoadApplicationByName(db, projectKey, appName)
	if errLA != nil {
		log.Warning("addBuildVariableHandler> Cannot load application %s: %s\n", appName, errLA)
		WriteError(w, r, errLA)
		return
	}

	// if buildNumber is 'last' fetch last build number
	buildNumber, errP := strconv.ParseInt(buildNumberS, 10, 64)
	if errP != nil {
		log.Warning("addBuildVariableHandler> Cannot parse build number %s: %s\n", buildNumberS, errP)
		WriteError(w, r, errP)
		return
	}

	// load pipeline_build.id
	pbID, errPB := pipeline.LoadPipelineBuildID(db, a.ID, p.ID, env.ID, buildNumber)
	if errPB != nil {
		log.Warning("addBuildVariableHandler> Cannot load pipeline build %d: %s\n", buildNumber, errPB)
		WriteError(w, r, errPB)
		return
	}

	// Get body
	data, errR := ioutil.ReadAll(r.Body)
	if errR != nil {
		log.Warning("addBuildVariableHandler> Cannot read body: %s\n", errR)
		WriteError(w, r, errR)
		return
	}

	// Unmarshal into results
	var v sdk.Variable
	if err := json.Unmarshal([]byte(data), &v); err != nil {
		log.Warning("addBuildVariableHandler> Cannot unmarshal Tests: %s\n", err)
		WriteError(w, r, err)
		return
	}

	tx, errBegin := db.Begin()
	if errBegin != nil {
		log.Warning("addBuildVariableHandler> Cannot start transaction: %s\n", errBegin)
		WriteError(w, r, errBegin)
		return
	}
	defer tx.Rollback()

	if err := pipeline.InsertBuildVariable(tx, pbID, v); err != nil {
		log.Warning("addBuildVariableHandler> Cannot add build variable: %s\n", err)
		WriteError(w, r, err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Warning("addBuildVariableHandler> Cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}
}

func addBuildTestResultsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {
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
			log.Warning("addBuildTestResultsHandler> Cannot load environment %s: %s\n", envName, err)
			WriteError(w, r, sdk.ErrUnknownEnv)
			return
		}

	}

	if env.ID != sdk.DefaultEnv.ID && !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadExecute) {
		log.Warning("addBuildTestResultsHandler> No enought right on this environment %s: \n", envName)
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	// Check that pipeline exists
	p, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		log.Warning("addBuildTestResultsHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Check that application exists
	a, err := application.LoadApplicationByName(db, projectKey, appName)
	if err != nil {
		log.Warning("addBuildTestResultsHandler> Cannot load application %s: %s\n", appName, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	buildNumber, err := strconv.ParseInt(buildNumberS, 10, 64)
	if err != nil {
		log.Warning("addBuildTestResultsHandler> Cannot parse build number %s: %s\n", buildNumberS, err)
		WriteError(w, r, err)
		return
	}

	// load pipeline_build.id
	pb, err := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(db, a.ID, p.ID, env.ID, buildNumber)
	if err != nil {
		log.Warning("addBuiltTestResultsHandler> Cannot loadpipelinebuild for %s/%s[%s] %d: %s\n", a.Name, p.Name, envName, buildNumber, err)
		WriteError(w, r, err)
		return
	}

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("addBuildTestResultsHandler> Cannot read body: %s\n", err)
		WriteError(w, r, err)
		return
	}

	// Unmarshal into results
	var new sdk.Tests
	err = json.Unmarshal([]byte(data), &new)
	if err != nil {
		log.Warning("addBuildtestResultsHandler> Cannot unmarshal Tests: %s\n", err)
		WriteError(w, r, err)
		return
	}

	// Load existing and merge
	tests, err := pipeline.LoadTestResults(db, pb.ID)
	if err != nil {
		log.Warning("addBuildTestResultsHandler> Cannot load test results: %s\n", err)
		WriteError(w, r, err)
		return
	}

	for _, s := range new.TestSuites {
		var found bool
		for i := range tests.TestSuites {
			if tests.TestSuites[i].Name == s.Name {
				found = true
				tests.TestSuites[i] = s
				break
			}
		}
		if !found {
			tests.TestSuites = append(tests.TestSuites, s)
		}
	}

	// update total values
	tests.Total = 0
	tests.TotalOK = 0
	tests.TotalKO = 0
	tests.TotalSkipped = 0
	for _, ts := range tests.TestSuites {
		tests.Total += ts.Total
		tests.TotalKO += ts.Failures + ts.Errors
		tests.TotalOK += ts.Total - ts.Skip - ts.Failures - ts.Errors
		tests.TotalSkipped += ts.Skip
	}

	if err := pipeline.UpdateTestResults(db, pb.ID, tests); err != nil {
		log.Warning("addBuildTestsResultsHandler> Cannot insert tests results: %s\n", err)
		WriteError(w, r, err)
	}

	stats.TestEvent(db, p.ProjectID, a.ID, tests)
}

func getBuildTestResultsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {

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
			WriteError(w, r, sdk.ErrUnknownEnv)
			return
		}

	}

	if env.ID != sdk.DefaultEnv.ID && !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionRead) {
		log.Warning("getBuildTestResultsHandler> No enought right on this environment %s: \n", envName)
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	// Check that pipeline exists
	p, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		log.Warning("getBuildTestResultsHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Check that application exists
	a, err := application.LoadApplicationByName(db, projectKey, appName)
	if err != nil {
		log.Warning("getBuildTestResultsHandler> Cannot load application %s: %s\n", appName, err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// if buildNumber is 'last' fetch last build number
	var buildNumber int64
	if buildNumberS == "last" {
		bn, err := pipeline.GetLastBuildNumberInTx(db, p.ID, a.ID, env.ID)
		if err != nil {
			log.Warning("getBuildTestResultsHandler> Cannot load last build number for %s: %s\n", pipelineName, err)
			WriteError(w, r, sdk.ErrNoPipelineBuild)
			return
		}
		buildNumber = bn
	} else {
		buildNumber, err = strconv.ParseInt(buildNumberS, 10, 64)
		if err != nil {
			log.Warning("getBuildTestResultsHandler> Cannot parse build number %s: %s\n", buildNumberS, err)
			WriteError(w, r, err)
			return
		}
	}

	// load pipeline_build.id
	pb, err := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(db, a.ID, p.ID, env.ID, buildNumber)
	if err != nil {
		log.Warning("getBuildTestResultsHandler> Cannot load pipeline build: %s\n", err)
		WriteError(w, r, err)
		return
	}

	tests, err := pipeline.LoadTestResults(db, pb.ID)
	if err != nil {
		log.Warning("getBuildTestResultsHandler> Cannot load test results: %s\n", err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, tests, http.StatusOK)
}
