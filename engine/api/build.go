package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/build"
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

func getPipelineBuildTriggeredHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
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

func deleteBuildHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
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

	result, err := pipeline.SelectBuildInHistory(db, p.ID, a.ID, buildNumber, env.ID)
	if err != nil && err != sql.ErrNoRows {
		log.Warning("deleteBuildHandler> Cannot check build history %s: %s\n", buildNumberS, err)
		WriteError(w, r, err)
		return
	}
	if err == nil {
		// Delete from history
		err = pipeline.DeletePipelineBuildArtifact(tx, result.ID)
		if err != nil {
			log.Warning("deleteBuildHandler> %s ! Cannot delete pipeline build artifact [%d] : %s\n", c.User.Username, result.ID, err)
			WriteError(w, r, err)
			return
		}

		// Delete history
		queryDeletePipelineHistory := `DELETE FROM pipeline_history WHERE pipeline_build_id = $1`
		_, err = tx.Exec(queryDeletePipelineHistory, result.ID)
		if err != nil {
			log.Warning("deleteBuildHandler> %s ! Cannot delete pipeline history [%s]: %s\n", c.User.Username, result.ID, err)
			WriteError(w, r, err)
			return
		}
	} else {
		// Delete from pipeline_build
		result, err := pipeline.LoadPipelineBuild(db, p.ID, a.ID, buildNumber, env.ID)
		if err != nil {
			log.Warning("deleteBuildHandler> %s ! Cannot load pipeline build to delete %s-%s-%s[%s] (buildNUmber:%d): %s\n", c.User.Username, projectKey, appName, pipelineName, env.Name, buildNumber, err)
			WriteError(w, r, err)
			return
		}
		err = pipeline.DeletePipelineBuild(tx, result.ID)
		if err != nil {
			log.Warning("deleteBuildHandler> Cannot delete pipeline build [%d]: %s\n", result.ID, err)
			WriteError(w, r, err)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Warning("deleteBuildHandler> Cannot commit transaction: %s\n", err)
		WriteError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func getBuildStateHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	buildNumberS := vars["build"]
	appName := vars["permApplicationName"]

	envName := r.FormValue("envName")

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
	var result sdk.PipelineBuild
	inHistory := false
	if buildNumberS == "last" {
		lastBuildNumber, history, err := pipeline.GetProbableLastBuildNumber(db, p.ID, a.ID, env.ID)
		if err != nil {
			log.Warning("getBuildStateHandler> Cannot load last pipeline build for %s-%s-%s: %s\n", a.Name, pipelineName, env.Name, err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		buildNumber = lastBuildNumber
		if history {
			inHistory = true
			result, err = pipeline.SelectBuildInHistory(db, p.ID, a.ID, buildNumber, env.ID)
			if err != nil {
				log.Warning("getBuildStateHandler> Cannot load last pipeline build from history for %s: %s\n", pipelineName, err)
				w.WriteHeader(http.StatusNotFound)
				return
			}
			for _, s := range result.Stages {
				for _, a := range s.ActionBuilds {
					a.Logs = ""
				}
			}
		}
	} else {
		buildNumber, err = strconv.ParseInt(buildNumberS, 10, 64)
		if err != nil {
			log.Warning("getBuildStateHandler> Cannot parse build number %s: %s\n", buildNumberS, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		result, err = pipeline.SelectBuildInHistory(db, p.ID, a.ID, buildNumber, env.ID)
		if err != nil && err != sql.ErrNoRows {
			log.Warning("getBuildStateHandler> Cannot check build history %s: %s\n", buildNumberS, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err == nil {
			inHistory = true
		}
		for _, s := range result.Stages {
			for _, a := range s.ActionBuilds {
				a.Logs = ""
			}
		}

	}

	if !inHistory {
		// load pipeline stage
		stages, err := pipeline.LoadStages(db, p.ID)
		if err != nil {
			log.Warning("getBuildStateHandler> Cannot load pipeline stages: %s\n", err)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// load pipeline_build.id
		pb, err := pipeline.LoadPipelineBuild(db, p.ID, a.ID, buildNumber, env.ID)
		if err != nil {
			log.Warning("getBuildStateHandler> %s! Cannot load last pipeline build for %s-%s-%s[%s] (buildNUmber:%d): %s\n", c.User.Username, projectKey, appName, pipelineName, env.Name, buildNumber, err)
			WriteError(w, r, err)
			return
		}

		// load actions status for build
		actionsBuilds, err := build.LoadBuildByPipelineBuildID(db, pb.ID)
		if err != nil {
			log.Warning("getBuildStateHandler> Cannot load pipeline build action: %s\n", err)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		for index := range actionsBuilds {
			// attach log to actionsBuild
			actionData := &actionsBuilds[index]

			// attach actionBuild to stage
			for indexStage := range stages {
				stage := &stages[indexStage]
				if stage.ID == actionData.PipelineStageID {
					stage.ActionBuilds = append(stage.ActionBuilds, *actionData)
				}
			}
		}

		result.Stages = stages
		result.Status = pb.Status
		result.Version = pb.Version
		result.Done = pb.Done
		result.Start = pb.Start
		result.Trigger = pb.Trigger
	}
	result.Environment = *env
	result.Application = *a
	result.Pipeline = *p
	result.BuildNumber = buildNumber

	WriteJSON(w, r, result, http.StatusOK)
}

func addQueueResultHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	// Get action name in URL
	vars := mux.Vars(r)
	id := vars["id"]

	/*
		workerID, err := worker.FindBuildingWorker(db, id)
		if err != nil {
			log.Warning("addQueueResultHandler> Could not load worker building %s: %s\n", id, err)
			WriteError(w, r, sdk.ErrForbidden)
			return
		}

		if c.Worker.ID != workerID {
			log.Warning("addQueueResultHandler> Worker %s is not supposed to be building %s\n", c.Worker.ID, id)
			WriteError(w, r, sdk.ErrForbidden)
			return
		}
	*/

	// Load Build
	b, err := build.LoadActionBuild(db, id)
	if err != nil {
		log.Warning("addQueueResultHandler> Cannot load queue from db: %s\n", err)
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
	err = build.UpdateActionBuildStatus(tx, &b, res.Status)
	if err != nil {
		log.Warning("addQueueResultHandler> Cannot update %s status: %s\n", id, err)
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

func takeActionBuildHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

	// Get action name in URL
	vars := mux.Vars(r)
	id := vars["id"]

	// Load worker
	caller, err := worker.LoadWorker(db, c.Worker.ID)
	if err != nil {
		log.Warning("takeActionBuildHandler> cannot load calling worker: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if caller.Status != sdk.StatusWaiting {
		log.Info("takeActionBuildHandler> worker %s is not available to for build (status = %s)\n", caller.ID, caller.Status)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// update database
	ab, err := build.TakeActionBuild(db, id, caller)
	if err != nil {
		if err != build.ErrAlreadyTaken {
			log.Warning("takeActionBuildHandler> Cannot give ActionBuild %s: %s\n", id, err)
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Update worker status to "building"
	err = worker.SetToBuilding(db, c.Worker.ID, ab.ID)
	if err != nil {
		log.Warning("takeActionBuildHandler> Cannot update worker status: %s\n", err)
		// We want the worker to run the task anyway now
	}

	log.Debug("Updated %s (PipelineAction %d) to %s\n", id, ab.PipelineActionID, sdk.StatusBuilding)

	// load action and return it to worker
	a, err := action.LoadActionByPipelineActionID(db, ab.PipelineActionID)
	if err != nil {
		log.Warning("takeActionBuildHandler> Cannot load action from  PipelineActionID %d: %s\n", ab.PipelineActionID, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	secrets, err := loadActionBuildSecrets(db, ab.ID)
	if err != nil {
		log.Warning("takeActionBuildHandler> Cannot load action build secrets: %s\n", err)
		WriteError(w, r, err)
		return
	}

	abi := worker.ActionBuildInfo{}
	abi.ActionBuild = ab
	abi.Action = *a
	abi.Secrets = secrets
	WriteJSON(w, r, abi, http.StatusOK)
}

func loadActionBuildSecrets(db *sql.DB, abID int64) ([]sdk.Variable, error) {

	query := `SELECT pipeline.project_id, pipeline_build.application_id, pipeline_build.environment_id
	FROM pipeline_build JOIN action_build ON action_build.pipeline_build_id = pipeline_build.id
	JOIN pipeline ON pipeline.id = pipeline_build.pipeline_id
	WHERE action_build.id = $1`

	var projectID, appID, envID int64
	var secrets []sdk.Variable
	err := db.QueryRow(query, abID).Scan(&projectID, &appID, &envID)
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

func getQueueHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	if c.Worker.ID != "" {
		// Load calling worker
		caller, errW := worker.LoadWorker(db, c.Worker.ID)
		if errW != nil {
			log.Warning("getQueueHandler> cannot load calling worker: %s\n", errW)
			WriteError(w, r, errW)
			return
		}
		if caller.Status != sdk.StatusWaiting {
			log.Debug("getQueueHandler> worker %s is not available to build (status = %s)\n", caller.ID, caller.Status)
			WriteError(w, r, sdk.ErrInvalidID)
			return
		}
	}

	var queue []sdk.ActionBuild
	var errQ error
	switch c.Agent {
	case sdk.HatcheryAgent, sdk.WorkerAgent:
		queue, errQ = build.LoadGroupWaitingQueue(db, c.Worker.GroupID)
	default:
		queue, errQ = build.LoadUserWaitingQueue(db, c.User)
	}

	if errQ != nil {
		log.Warning("getQueueHandler> Cannot load queue from db: %s\n", errQ)
		WriteError(w, r, errQ)
		return
	}

	if log.IsDebug() {
		for _, a := range queue {
			log.Debug("getQueueHandler> ActionBuild : %d %s [%s]", a.ID, a.ActionName, a.Status)
		}
	}

	WriteJSON(w, r, queue, http.StatusOK)
}

func requirementsErrorHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
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

func addBuildVariableHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
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
	pb, errPB := pipeline.LoadPipelineBuild(db, p.ID, a.ID, buildNumber, env.ID)
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

	if err := pipeline.InsertBuildVariable(db, pb.ID, v); err != nil {
		log.Warning("addBuildVariableHandler> Cannot add build variable: %s\n", err)
		WriteError(w, r, err)
		return
	}

	log.Notice("addBuildVariableHandler> Build variable %s added\n", v.Name)
}

func addBuildTestResultsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
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

	// if buildNumber is 'last' fetch last build number
	var buildNumber int64
	if buildNumberS == "last" {
		bn, _, err := pipeline.GetProbableLastBuildNumber(db, p.ID, a.ID, env.ID)
		if err != nil {
			log.Warning("addBuildTestResultsHandler> Cannot load last build number for %s: %s\n", pipelineName, err)
			WriteError(w, r, sdk.ErrNoPipelineBuild)
			return
		}
		buildNumber = bn
	} else {
		buildNumber, err = strconv.ParseInt(buildNumberS, 10, 64)
		if err != nil {
			log.Warning("addBuildTestResultsHandler> Cannot parse build number %s: %s\n", buildNumberS, err)
			WriteError(w, r, err)
			return
		}
	}

	// load pipeline_build.id
	pb, err := pipeline.LoadPipelineBuild(db, p.ID, a.ID, buildNumber, env.ID)
	if err != nil {
		log.Warning("addBuiltTestResultsHandler> Cannot loadpipelinebuild for %s/%s[%s] %d: %s\n", a.Name, p.Name, envName, buildNumber, err)
		if err != sdk.ErrNoPipelineBuild {
			log.Warning("addBuildTestResultsHandler> Cannot load pipeline build: %s\n", err)
			WriteError(w, r, err)
			return
		}

		pb, err = pipeline.LoadPipelineHistoryBuild(db, p.ID, a.ID, buildNumber, env.ID)
		if err != nil {
			log.Warning("addBuildTestResultsHandler> Cannot load pipeline build from history: %s\n", err)
			WriteError(w, r, sdk.ErrNoPipelineBuild)
			return
		}
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
	tests, err := build.LoadTestResults(db, pb.ID)
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

	err = build.UpdateTestResults(db, pb.ID, tests)
	if err != nil {
		log.Warning("addBuildTestsResultsHandler> Cannot insert tests results: %s\n", err)
		WriteError(w, r, err)
	}

	stats.TestEvent(db, p.ProjectID, a.ID, tests)
}

func getBuildTestResultsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {

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
		bn, _, err := pipeline.GetProbableLastBuildNumber(db, p.ID, a.ID, env.ID)
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
	pb, err := pipeline.LoadPipelineBuild(db, p.ID, a.ID, buildNumber, env.ID)
	if err != nil {
		if err != sdk.ErrNoPipelineBuild {
			log.Warning("getBuildTestResultsHandler> Cannot load pipeline build: %s\n", err)
			WriteError(w, r, err)
			return
		}

		pb, err = pipeline.LoadPipelineHistoryBuild(db, p.ID, a.ID, buildNumber, env.ID)
		if err != nil {
			log.Warning("getBuildTestResultsHandler> Cannot load pipeline build from history: %s\n", err)
			WriteError(w, r, sdk.ErrNoPipelineBuild)
			return
		}
	}

	tests, err := build.LoadTestResults(db, pb.ID)
	if err != nil {
		log.Warning("getBuildTestResultsHandler> Cannot load test results: %s\n", err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, tests, http.StatusOK)
}
