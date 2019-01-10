package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) getStepBuildLogsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		projectKey := vars["key"]
		pipelineName := vars["permPipelineKey"]
		buildNumberS := vars["build"]
		appName := vars["permApplicationName"]
		pipelineActionIDString := vars["actionID"]
		stepOrderString := vars["stepOrder"]

		stepOrder, errInt := strconv.ParseInt(stepOrderString, 10, 64)
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
			env, errEnv = environment.LoadEnvironmentByName(api.mustDB(), projectKey, envName)
			if errEnv != nil {
				return sdk.WrapError(errEnv, "Cannot load environment %s", envName)
			}
		}

		if !permission.AccessToEnvironment(projectKey, env.Name, getUser(ctx), permission.PermissionRead) {
			return sdk.WrapError(sdk.ErrForbidden, "No enough right on this environment %s", envName)
		}

		// Check that pipeline exists
		p, err := pipeline.LoadPipeline(api.mustDB(), projectKey, pipelineName, false)
		if err != nil {
			return sdk.WrapError(err, "Cannot load pipeline %s", pipelineName)
		}

		// Check that application exists
		a, err := application.LoadByName(api.mustDB(), api.Cache, projectKey, appName, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "Cannot load application %s", appName)
		}

		// if buildNumber is 'last' fetch last build number
		var buildNumber int64
		if buildNumberS == "last" {
			var errLastBuildN error
			bn, errLastBuildN := pipeline.GetLastBuildNumberInTx(api.mustDB(), p.ID, a.ID, env.ID)
			if errLastBuildN != nil {
				return sdk.WrapError(errLastBuildN, "Cannot load last build number for %s", pipelineName)
			}
			buildNumber = bn
		} else {
			buildNumber, err = strconv.ParseInt(buildNumberS, 10, 64)
			if err != nil {
				return sdk.WrapError(err, "Cannot parse build number %s", buildNumberS)
			}
		}

		// load pipeline_build.id
		pb, errPB := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(api.mustDB(), a.ID, p.ID, env.ID, buildNumber)
		if errPB != nil {
			return sdk.WrapError(errPB, "Cannot load pipeline build id")
		}

		result, errLog := pipeline.LoadPipelineStepBuildLogs(api.mustDB(), pb, pipelineActionID, stepOrder)
		if errLog != nil {
			return sdk.WrapError(errLog, "Cannot load pipeline build logs")
		}

		return service.WriteJSON(w, result, http.StatusOK)
	}
}

func (api *API) getBuildLogsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		projectKey := vars["key"]
		pipelineName := vars["permPipelineKey"]
		buildNumberS := vars["build"]
		appName := vars["permApplicationName"]

		// Get offset
		err := r.ParseForm()
		if err != nil {
			return sdk.WrapError(err, "cannot parse form")
		}

		var env *sdk.Environment
		envName := r.FormValue("envName")
		if envName == "" || envName == sdk.DefaultEnv.Name {
			env = &sdk.DefaultEnv
		} else {
			env, err = environment.LoadEnvironmentByName(api.mustDB(), projectKey, envName)
			if err != nil {
				return sdk.WrapError(sdk.ErrUnknownEnv, "Cannot load environment %s", envName)

			}

		}

		if !permission.AccessToEnvironment(projectKey, env.Name, getUser(ctx), permission.PermissionRead) {
			return sdk.WrapError(sdk.ErrForbidden, "No enough right on this environment %s", envName)

		}

		// Check that pipeline exists
		p, err := pipeline.LoadPipeline(api.mustDB(), projectKey, pipelineName, false)
		if err != nil {
			return sdk.WrapError(err, "Cannot load pipeline %s", pipelineName)
		}

		// Check that application exists
		a, err := application.LoadByName(api.mustDB(), api.Cache, projectKey, appName, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "Cannot load application %s", appName)

		}

		// if buildNumber is 'last' fetch last build number
		var buildNumber int64
		if buildNumberS == "last" {
			var errLastBuildN error
			bn, errLastBuildN := pipeline.GetLastBuildNumberInTx(api.mustDB(), p.ID, a.ID, env.ID)
			if errLastBuildN != nil {
				return sdk.WrapError(errLastBuildN, "Cannot load last build number for %s", pipelineName)
			}
			buildNumber = bn
		} else {
			buildNumber, err = strconv.ParseInt(buildNumberS, 10, 64)
			if err != nil {
				return sdk.WrapError(err, "Cannot parse build number %s", buildNumberS)

			}
		}

		// load pipeline_build.id
		var pipelinelogs []sdk.Log
		pb, err := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(api.mustDB(), a.ID, p.ID, env.ID, buildNumber)
		if err != nil {
			return sdk.WrapError(err, "Cannot load pipeline build id")

		}

		pipelinelogs, err = pipeline.LoadPipelineBuildLogs(api.mustDB(), pb)
		if err != nil {
			return sdk.WrapError(err, "Cannot load pipeline build logs")

		}

		// add pipeline result
		// Important for cli to known that build is finished
		if pb.Status.String() == sdk.StatusFail.String() || pb.Status.String() == sdk.StatusSuccess.String() {
			l := sdk.NewLog(0, fmt.Sprintf("Build finished with status: %s\n", pb.Status), pb.ID, 0)
			pipelinelogs = append(pipelinelogs, *l)
		}
		return service.WriteJSON(w, pipelinelogs, http.StatusOK)
	}
}

func (api *API) getPipelineBuildJobLogsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

		// Get pipeline and action name in URL
		vars := mux.Vars(r)
		projectKey := vars["key"]
		pipelineName := vars["permPipelineKey"]
		buildNumberS := vars["build"]
		pipelineActionIDString := vars["actionID"]
		appName := vars["permApplicationName"]

		pipelineActionID, err := strconv.ParseInt(pipelineActionIDString, 10, 64)
		if err != nil {
			return sdk.WrapError(sdk.ErrInvalidID, "actionID should be an integer")
		}

		// Check that pipeline exists
		p, err := pipeline.LoadPipeline(api.mustDB(), projectKey, pipelineName, false)
		if err != nil {
			return sdk.WrapError(err, "Cannot load pipeline %s", pipelineName)
		}

		a, err := application.LoadByName(api.mustDB(), api.Cache, projectKey, appName, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "Cannot load application %s", appName)
		}

		var env *sdk.Environment
		envName := r.FormValue("envName")
		if envName == "" || envName == sdk.DefaultEnv.Name {
			env = &sdk.DefaultEnv
		} else {
			var errload error
			env, errload = environment.LoadEnvironmentByName(api.mustDB(), projectKey, envName)
			if errload != nil {
				return sdk.WrapError(errload, "Cannot load environment %s on application %s", envName, appName)
			}
		}

		if !permission.AccessToEnvironment(projectKey, env.Name, getUser(ctx), permission.PermissionRead) {
			return sdk.WrapError(sdk.ErrForbidden, "No enough right on this environment %s", envName)
		}

		// if buildNumber is 'last' fetch last build number
		var buildNumber int64
		if buildNumberS == "last" {
			bn, errLastBuild := pipeline.GetLastBuildNumberInTx(api.mustDB(), p.ID, a.ID, env.ID)
			if errLastBuild != nil {
				return sdk.WrapError(errLastBuild, "Cannot load last build number for %s", pipelineName)
			}
			buildNumber = bn
		} else {
			buildNumber, err = strconv.ParseInt(buildNumberS, 10, 64)
			if err != nil {
				return sdk.WrapError(err, "Cannot parse build number %s", buildNumberS)
			}
		}

		// load pipeline_build.id
		var pipelinelogs sdk.BuildState
		pb, err := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(api.mustDB(), a.ID, p.ID, env.ID, buildNumber)
		if err != nil {
			return sdk.WrapError(err, "Cannot load pipeline build id")
		}
		pipelinelogs, err = pipeline.LoadPipelineBuildJobLogs(api.mustDB(), pb, pipelineActionID)
		if err != nil {
			return sdk.WrapError(err, "Cannot load pipeline build logs")
		}

		return service.WriteJSON(w, pipelinelogs, http.StatusOK)
	}
}
