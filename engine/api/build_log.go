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
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"

)

func getBuildLogsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {

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
		WriteError(w, r, err)
		return
	}
	offsetS := r.FormValue("offset")
	var offset int64
	if offsetS != "" {
		offset, err = strconv.ParseInt(offsetS, 10, 64)
		if err != nil {
			log.Warning("getBuildLogsHandler> Cannot parse offset %s: %s\n", offsetS, err)
			WriteError(w, r, err)
			return
		}
	}

	var env *sdk.Environment
	envName := r.FormValue("envName")
	if envName == "" || envName == sdk.DefaultEnv.Name {
		env = &sdk.DefaultEnv
	} else {
		env, err = environment.LoadEnvironmentByName(db, projectKey, envName)
		if err != nil {
			log.Warning("getBuildLogsHandler> Cannot load environment %s: %s\n", envName, err)
			WriteError(w, r, sdk.ErrUnknownEnv)
			return
		}

	}

	if env.ID != sdk.DefaultEnv.ID && !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionRead) {
		log.Warning("getBuildLogsHandler> No enought right on this environment %s: \n", envName)
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	// Check that pipeline exists
	p, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		log.Warning("getBuildLogsHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
		WriteError(w, r, sdk.ErrPipelineNotFound)
		return
	}

	// Check that application exists
	a, err := application.LoadApplicationByName(db, projectKey, appName)
	if err != nil {
		log.Warning("getBuildLogsHandler> Cannot load application %s: %s\n", appName, err)
		WriteError(w, r, sdk.ErrApplicationNotFound)
		return
	}

	// if buildNumber is 'last' fetch last build number
	var buildNumber int64
	if buildNumberS == "last" {
		bn, err := pipeline.GetLastBuildNumberInTx(db, p.ID, a.ID, env.ID)
		if err != nil {
			log.Warning("getBuildLogsHandler> Cannot load last build number for %s: %s\n", pipelineName, err)
			WriteError(w, r, err)
			return
		}
		buildNumber = bn
	} else {
		buildNumber, err = strconv.ParseInt(buildNumberS, 10, 64)
		if err != nil {
			log.Warning("getBuildLogsHandler> Cannot parse build number %s: %s\n", buildNumberS, err)
			WriteError(w, r, err)
			return
		}
	}

	// load pipeline_build.id
	var pipelinelogs []sdk.Log
	pb, err := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(db, a.ID, p.ID, env.ID, buildNumber)
	if err != nil {
		log.Warning("getBuildLogsHandler> Cannot load pipeline build id: %s\n", err)
		WriteError(w, r, err)
		return
	}

	pipelinelogs, err = pipeline.LoadPipelineBuildLogs(db, pb.ID, offset)
	if err != nil {
		log.Warning("getBuildLogshandler> Cannot load pipeline build logs: %s\n", err)
		WriteError(w, r, err)
		return
	}

	// add pipeline result
	// Important for cli to known that build is finished
	if pb.Status.String() == sdk.StatusFail.String() || pb.Status.String() == sdk.StatusSuccess.String() {
		l := sdk.NewLog(0, "SYSTEM", fmt.Sprintf("Build finished with status: %s\n", pb.Status))
		pipelinelogs = append(pipelinelogs, *l)
	}

	WriteJSON(w, r, pipelinelogs, http.StatusOK)
}

func getActionBuildLogsHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {

	// Get pipeline and action name in URL
	vars := mux.Vars(r)
	projectKey := vars["key"]
	pipelineName := vars["permPipelineKey"]
	buildNumberS := vars["build"]
	actionIDString := vars["actionID"]
	appName := vars["permApplicationName"]

	actionID, err := strconv.ParseInt(actionIDString, 10, 64)
	if err != nil {
		log.Warning("getActionBuildLogsHandler> actionID should be an integer : %s\n", err)
		WriteError(w, r, err)
		return
	}

	// Get offset
	err = r.ParseForm()
	if err != nil {
		log.Warning("getActionBuildLogsHandler> cannot parse form: %s\n", err)
		WriteError(w, r, err)
		return
	}
	offsetS := r.FormValue("offset")
	var offset int64
	if offsetS != "" {
		offset, err = strconv.ParseInt(offsetS, 10, 64)
		if err != nil {
			log.Warning("getActionBuildLogsHandler> Cannot parse offset %s: %s\n", offsetS, err)
			WriteError(w, r, err)
			return
		}
	}

	// Check that pipeline exists
	p, err := pipeline.LoadPipeline(db, projectKey, pipelineName, false)
	if err != nil {
		log.Warning("getActionBuildLogsHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
		WriteError(w, r, sdk.ErrPipelineNotFound)
		return
	}

	a, err := application.LoadApplicationByName(db, projectKey, appName)
	if err != nil {
		log.Warning("getActionBuildLogsHandler> Cannot load application %s: %s\n", appName, err)
		WriteError(w, r, sdk.ErrApplicationNotFound)
		return
	}

	var env *sdk.Environment
	envName := r.FormValue("envName")
	if envName == "" || envName == sdk.DefaultEnv.Name {
		env = &sdk.DefaultEnv
	} else {
		env, err = environment.LoadEnvironmentByName(db, projectKey, envName)
	}

	if env.ID != sdk.DefaultEnv.ID && !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionRead) {
		log.Warning("getActionBuildLogsHandler> No enought right on this environment %s: \n", envName)
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	// if buildNumber is 'last' fetch last build number
	var buildNumber int64
	if buildNumberS == "last" {
		bn, err := pipeline.GetLastBuildNumberInTx(db, p.ID, a.ID, env.ID)
		if err != nil {
			log.Warning("getActionBuildLogsHandler> Cannot load last build number for %s: %s\n", pipelineName, err)
			WriteError(w, r, err)
			return
		}
		buildNumber = bn
	} else {
		buildNumber, err = strconv.ParseInt(buildNumberS, 10, 64)
		if err != nil {
			log.Warning("getActionBuildLogsHandler> Cannot parse build number %s: %s\n", buildNumberS, err)
			WriteError(w, r, err)
			return
		}
	}

	// load pipeline_build.id
	var pipelinelogs sdk.BuildState
	pbID, err := pipeline.LoadPipelineBuildID(db, a.ID, p.ID, env.ID, buildNumber)
	if err != nil {

		log.Warning("getActionBuildLogsHandler> Cannot load pipeline build id: %s\n", err)
		WriteError(w, r, err)
		return

	}
	pipelinelogs, err = pipeline.LoadPipelineActionBuildLogs(db, pbID, actionID, offset)
	if err != nil {
		log.Warning("getActionBuildLogsHandler> Cannot load pipeline build logs: %s\n", err)
		WriteError(w, r, err)
		return
	}

	WriteJSON(w, r, pipelinelogs, http.StatusOK)
}

func addBuildLogHandler(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {

	// Get action name in URL
	vars := mux.Vars(r)
	idString := vars["id"]

	// Load Queue
	id, errInt := strconv.ParseInt(idString, 10, 64)
	if errInt != nil {
		WriteError(w, r, sdk.ErrInvalidID)
		return
	}
	_, err := pipeline.GetPipelineBuildJob(db, id)
	if err != nil {
		log.Warning("addBuildLogHandler> Cannot load build %s from db: %s\n", id, err)
		WriteError(w, r, err)
		return
	}

	// Get body
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warning("addBuildLogHandler> Cannot read body: %s\n", err)
		WriteError(w, r, err)
		return
	}

	// Unmarshal into results
	var logs []sdk.Log

	err = json.Unmarshal([]byte(data), &logs)
	if err != nil {
		log.Warning("addBuildLogHandler> Cannot unmarshal Result: %s\n", err)
		WriteError(w, r, err)
		return
	}

	for i := range logs {
		err = pipeline.InsertLog(db, logs[i].ActionBuildID, logs[i].Step, logs[i].Value)
		if err != nil {
			log.Warning("addBuildLogHandler> Cannot insert log line:  %s\n", err)
			WriteError(w, r, err)
			return
		}
	}
}

func setEngineLogLevel(w http.ResponseWriter, r *http.Request, db *gorp.DbMap, c *context.Context) {

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
		w.WriteHeader(http.StatusBadRequest)
		return
	}

}
