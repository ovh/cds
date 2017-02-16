package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func getStepBuildLogsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	buildNumberS := vars["build"]
	appName := vars["permApplicationName"]
	pipelineActionIDString := vars["actionID"]
	stepOrderString := vars["stepOrder"]

	stepOrder, errInt := strconv.Atoi(stepOrderString)
	if errInt != nil {
		return sdk.ErrWrongRequest
	}

	pipelineActionID, errPA := strconv.ParseInt(pipelineActionIDString, 10, 64)
	if errPA != nil {
		return sdk.ErrInvalidID
	}

	var env *sdk.Environment
	envName := r.FormValue("envName")
	if envName == "" || envName == sdk.DefaultEnv.Name {
		env = &sdk.DefaultEnv
	} else {
		var errEnv error
		env, errEnv = environment.LoadEnvironmentByName(db, projectKey, envName)
		if errEnv != nil {
			log.Warning("getStepBuildLogsHandler> Cannot load environment %s: %s\n", envName, errEnv)
			return errEnv
		}
	}

	if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionRead) {
		log.Warning("getStepBuildLogsHandler> No enought right on this environment %s: \n", envName)
		return sdk.ErrForbidden
	}

	// Check that pipeline exists
	p, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		log.Warning("getStepBuildLogsHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
		return err
	}

	// Check that application exists
	a, err := application.LoadApplicationByName(db, projectKey, appName)
	if err != nil {
		log.Warning("getStepBuildLogsHandler> Cannot load application %s: %s\n", appName, err)
		return err
	}

	// if buildNumber is 'last' fetch last build number
	var buildNumber int64
	if buildNumberS == "last" {
		bn, err := pipeline.GetLastBuildNumberInTx(db, p.ID, a.ID, env.ID)
		if err != nil {
			log.Warning("getStepBuildLogsHandler> Cannot load last build number for %s: %s\n", pipelineName, err)
			return err
		}
		buildNumber = bn
	} else {
		buildNumber, err = strconv.ParseInt(buildNumberS, 10, 64)
		if err != nil {
			log.Warning("getStepBuildLogsHandler> Cannot parse build number %s: %s\n", buildNumberS, err)
			return err
		}
	}

	// load pipeline_build.id
	pb, errPB := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(db, a.ID, p.ID, env.ID, buildNumber)
	if errPB != nil {
		log.Warning("getBuildLogsHandler> Cannot load pipeline build id: %s\n", errPB)
		return errPB
	}

	result, errLog := pipeline.LoadPipelineStepBuildLogs(db, pb, pipelineActionID, stepOrder)
	if errLog != nil {
		log.Warning("getBuildLogshandler> Cannot load pipeline build logs: %s\n", errLog)
		return errLog
	}

	return WriteJSON(w, r, result, http.StatusOK)
}
func getBuildLogsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {
	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	buildNumberS := vars["build"]
	appName := vars["permApplicationName"]

	// Get offset
	err := r.ParseForm()
	if err != nil {
		log.Warning("getBuildLogsHandler> cannot parse form: %s\n", err)
		return err
	}

	var env *sdk.Environment
	envName := r.FormValue("envName")
	if envName == "" || envName == sdk.DefaultEnv.Name {
		env = &sdk.DefaultEnv
	} else {
		env, err = environment.LoadEnvironmentByName(db, projectKey, envName)
		if err != nil {
			log.Warning("getBuildLogsHandler> Cannot load environment %s: %s\n", envName, err)
			return sdk.ErrUnknownEnv

		}

	}

	if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionRead) {
		log.Warning("getBuildLogsHandler> No enought right on this environment %s: \n", envName)
		return sdk.ErrForbidden

	}

	// Check that pipeline exists
	p, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		log.Warning("getBuildLogsHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
		return sdk.ErrPipelineNotFound

	}

	// Check that application exists
	a, err := application.LoadApplicationByName(db, projectKey, appName)
	if err != nil {
		log.Warning("getBuildLogsHandler> Cannot load application %s: %s\n", appName, err)
		return sdk.ErrApplicationNotFound

	}

	// if buildNumber is 'last' fetch last build number
	var buildNumber int64
	if buildNumberS == "last" {
		bn, err := pipeline.GetLastBuildNumberInTx(db, p.ID, a.ID, env.ID)
		if err != nil {
			log.Warning("getBuildLogsHandler> Cannot load last build number for %s: %s\n", pipelineName, err)
			return err

		}
		buildNumber = bn
	} else {
		buildNumber, err = strconv.ParseInt(buildNumberS, 10, 64)
		if err != nil {
			log.Warning("getBuildLogsHandler> Cannot parse build number %s: %s\n", buildNumberS, err)
			return err

		}
	}

	// load pipeline_build.id
	var pipelinelogs []sdk.Log
	pb, err := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(db, a.ID, p.ID, env.ID, buildNumber)
	if err != nil {
		log.Warning("getBuildLogsHandler> Cannot load pipeline build id: %s\n", err)
		return err

	}

	pipelinelogs, err = pipeline.LoadPipelineBuildLogs(db, pb)
	if err != nil {
		log.Warning("getBuildLogshandler> Cannot load pipeline build logs: %s\n", err)
		return err

	}

	// add pipeline result
	// Important for cli to known that build is finished
	if pb.Status.String() == sdk.StatusFail.String() || pb.Status.String() == sdk.StatusSuccess.String() {
		l := sdk.NewLog(0, fmt.Sprintf("Build finished with status: %s\n", pb.Status), pb.ID, 0)
		pipelinelogs = append(pipelinelogs, *l)
	}
	return WriteJSON(w, r, pipelinelogs, http.StatusOK)
}

func getPipelineBuildJobLogsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {

	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	buildNumberS := vars["build"]
	pipelineActionIDString := vars["actionID"]
	appName := vars["permApplicationName"]

	pipelineActionID, err := strconv.ParseInt(pipelineActionIDString, 10, 64)
	if err != nil {
		log.Warning("getPipelineBuildJobLogsHandler> actionID should be an integer : %s\n", err)
		return sdk.ErrInvalidID
	}

	// Check that pipeline exists
	p, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		log.Warning("getPipelineBuildJobLogsHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
		return err
	}

	a, err := application.LoadApplicationByName(db, projectKey, appName)
	if err != nil {
		log.Warning("getPipelineBuildJobLogsHandler> Cannot load application %s: %s\n", appName, err)
		return err
	}

	var env *sdk.Environment
	envName := r.FormValue("envName")
	if envName == "" || envName == sdk.DefaultEnv.Name {
		env = &sdk.DefaultEnv
	} else {
		env, err = environment.LoadEnvironmentByName(db, projectKey, envName)
	}

	if !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionRead) {
		log.Warning("getPipelineBuildJobLogsHandler> No enought right on this environment %s: \n", envName)
		return sdk.ErrForbidden
	}

	// if buildNumber is 'last' fetch last build number
	var buildNumber int64
	if buildNumberS == "last" {
		bn, err := pipeline.GetLastBuildNumberInTx(db, p.ID, a.ID, env.ID)
		if err != nil {
			log.Warning("getPipelineBuildJobLogsHandler> Cannot load last build number for %s: %s\n", pipelineName, err)
			return err
		}
		buildNumber = bn
	} else {
		buildNumber, err = strconv.ParseInt(buildNumberS, 10, 64)
		if err != nil {
			log.Warning("getPipelineBuildJobLogsHandler> Cannot parse build number %s: %s\n", buildNumberS, err)
			return err
		}
	}

	// load pipeline_build.id
	var pipelinelogs sdk.BuildState
	pb, err := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(db, a.ID, p.ID, env.ID, buildNumber)
	if err != nil {
		log.Warning("getPipelineBuildJobLogsHandler> Cannot load pipeline build id: %s\n", err)
		return err
	}
	pipelinelogs, err = pipeline.LoadPipelineBuildJobLogs(db, pb, pipelineActionID)
	if err != nil {
		log.Warning("getPipelineBuildJobLogsHandler> Cannot load pipeline build logs: %s\n", err)
		return err
	}

	return WriteJSON(w, r, pipelinelogs, http.StatusOK)
}

func addBuildLogHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("addBuildLogHandler> Cannot read body: %s\n", err)
		return err

	}

	// Unmarshal into results
	var logs sdk.Log

	if err := json.Unmarshal([]byte(data), &logs); err != nil {
		log.Warning("addBuildLogHandler> Cannot unmarshal Result: %s\n", err)
		return err

	}

	existingLogs, errLog := pipeline.LoadStepLogs(db, logs.PipelineBuildJobID, logs.StepOrder)
	if errLog != nil && errLog != sql.ErrNoRows {
		log.Warning("addBuildLogHandler> Cannot load existing logs: %s\n", err)
		return err
	}

	if existingLogs == nil {
		if err := pipeline.InsertLog(db, &logs); err != nil {
			log.Warning("addBuildLogHandler> Cannot insert log:  %s\n", err)
			return err
		}
	} else {
		existingLogs.Value += logs.Value
		existingLogs.LastModified = logs.LastModified
		existingLogs.Done = logs.Done
		if err := pipeline.UpdateLog(db, existingLogs); err != nil {
			log.Warning("addBuildLogHandler> Cannot update log:  %s\n", err)
			return err
		}
	}
	return nil
}

func setEngineLogLevel(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Ctx) error {

	// Get log level in URL
	vars := mux.Vars(r)
	lvl := vars["level"]

	switch lvl {
	case "debug":
		log.SetLevel(log.DebugLevel)
		break
	case "info":
		log.SetLevel(log.InfoLevel)
		break
	case "notice":
		log.SetLevel(log.NoticeLevel)
		break
	case "warning":
		log.SetLevel(log.WarningLevel)
		break
	case "critical":
		log.SetLevel(log.CriticalLevel)
		break

	default:
		log.Warning("setEngineLogLevel> Unknown log level %s\n", lvl)
		return sdk.ErrWrongRequest

	}
	return nil
}
