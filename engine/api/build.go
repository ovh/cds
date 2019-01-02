package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/artifact"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// DEPRECATED
func (api *API) getPipelineBuildTriggeredHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		pipelineName := vars["permPipelineKey"]
		appName := vars["permApplicationName"]

		envName := r.FormValue("envName")

		buildNumber, err := requestVarInt(r, "build")
		if err != nil {
			return sdk.WrapError(err, "Invalid build number")
		}

		// Load Pipeline
		p, err := pipeline.LoadPipeline(api.mustDB(), projectKey, pipelineName, false)
		if err != nil {
			return sdk.WrapError(err, "Cannot load pipeline %s", pipelineName)
		}

		// Load Application
		a, err := application.LoadByName(api.mustDB(), api.Cache, projectKey, appName, getUser(ctx))
		if err != nil {
			return sdk.WrapError(err, "Cannot load application %s", appName)
		}

		// Load Env
		env := &sdk.DefaultEnv
		if envName != sdk.DefaultEnv.Name && envName != "" {
			env, err = environment.LoadEnvironmentByName(api.mustDB(), projectKey, envName)
			if err != nil {
				return sdk.WrapError(err, "Cannot load environment %s", envName)
			}
		}

		// Load Children
		pbs, err := pipeline.LoadPipelineBuildChildren(api.mustDB(), p.ID, a.ID, buildNumber, env.ID)
		if err != nil {
			return sdk.WrapError(sdk.ErrNoPipelineBuild, "Cannot load pipeline build children: %s", err)
		}
		return service.WriteJSON(w, pbs, http.StatusOK)
	}
}

func (api *API) deleteBuildHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		pipelineName := vars["permPipelineKey"]
		appName := vars["permApplicationName"]

		envName := r.FormValue("envName")

		buildNumber, err := requestVarInt(r, "build")
		if err != nil {
			return sdk.WrapError(err, "Invalid build number")
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
		if envName == "" || envName == sdk.DefaultEnv.Name {
			env = &sdk.DefaultEnv
		} else {
			env, err = environment.LoadEnvironmentByName(api.mustDB(), projectKey, envName)
			if err != nil {
				return sdk.WrapError(err, "Cannot load environment %s", envName)
			}
		}

		if !permission.AccessToEnvironment(projectKey, env.Name, getUser(ctx), permission.PermissionRead) {
			return sdk.WrapError(sdk.ErrForbidden, "No enough right on this environment %s", envName)
		}

		pbID, errPB := pipeline.LoadPipelineBuildID(api.mustDB(), a.ID, p.ID, env.ID, buildNumber)
		if errPB != nil {
			return sdk.WrapError(errPB, "Cannot load pipeline build")
		}

		tx, err := api.mustDB().Begin()
		if err != nil {
			return sdk.WrapError(err, "Cannot start transaction")
		}
		defer tx.Rollback()

		if err := pipeline.DeletePipelineBuildByID(tx, pbID); err != nil {
			return sdk.WrapError(sdk.ErrUnknownError, "Cannot delete pipeline build: %s", err)
		}

		if err := tx.Commit(); err != nil {
			return sdk.WrapError(err, "Cannot commit transaction")
		}

		return nil
	}
}

func (api *API) getBuildStateHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		pipelineName := vars["permPipelineKey"]
		buildNumberS := vars["build"]
		appName := vars["permApplicationName"]

		envName := r.FormValue("envName")
		withArtifacts := r.FormValue("withArtifacts")
		withTests := r.FormValue("withTests")

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
		if envName == "" || envName == sdk.DefaultEnv.Name {
			env = &sdk.DefaultEnv
		} else {
			env, err = environment.LoadEnvironmentByName(api.mustDB(), projectKey, envName)
			if err != nil {
				return sdk.WrapError(err, "Cannot load environment %s", envName)
			}
		}

		if !permission.AccessToEnvironment(projectKey, env.Name, getUser(ctx), permission.PermissionRead) {
			return sdk.WrapError(sdk.ErrForbidden, "No enough right on this environment %s: ", envName)
		}

		// if buildNumber is 'last' fetch last build number
		var buildNumber int64
		if buildNumberS == "last" {
			lastBuildNumber, errg := pipeline.GetLastBuildNumberInTx(api.mustDB(), p.ID, a.ID, env.ID)
			if errg != nil {
				return sdk.WrapError(sdk.ErrNotFound, "Cannot load last pipeline build number for %s-%s-%s: %s", a.Name, pipelineName, env.Name, errg)
			}
			buildNumber = lastBuildNumber
		} else {
			buildNumber, err = strconv.ParseInt(buildNumberS, 10, 64)
			if err != nil {
				return sdk.WrapError(sdk.ErrWrongRequest, "Cannot parse build number %s: %s", buildNumberS, err)
			}
		}

		// load pipeline_build.id
		pb, err := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(api.mustDB(), a.ID, p.ID, env.ID, buildNumber)
		if err != nil {
			return sdk.WrapError(err, "%s! Cannot load last pipeline build for %s-%s-%s[%s] (buildNUmber:%d)", getUser(ctx).Username, projectKey, appName, pipelineName, env.Name, buildNumber)
		}

		if withArtifacts == "true" {
			var errLoadArtifact error
			pb.Artifacts, errLoadArtifact = artifact.LoadArtifactsByBuildNumber(api.mustDB(), p.ID, a.ID, buildNumber, env.ID)
			if errLoadArtifact != nil {
				return sdk.WrapError(errLoadArtifact, "Cannot load artifacts")
			}
		}

		if withTests == "true" {
			tests, errLoadTests := pipeline.LoadTestResults(api.mustDB(), pb.ID)
			if errLoadTests != nil {
				return sdk.WrapError(errLoadTests, "Cannot load tests")
			}
			if len(tests.TestSuites) > 0 {
				pb.Tests = &tests
			}
		}
		pb.Translate(r.Header.Get("Accept-Language"))

		return service.WriteJSON(w, pb, http.StatusOK)
	}
}

func (api *API) getBuildTestResultsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
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
			env, err = environment.LoadEnvironmentByName(api.mustDB(), projectKey, envName)
			if err != nil {
				return sdk.WrapError(sdk.ErrUnknownEnv, "Cannot load environment %s: %s", envName, err)
			}
		}

		if !permission.AccessToEnvironment(projectKey, env.Name, getUser(ctx), permission.PermissionRead) {
			return sdk.WrapError(sdk.ErrForbidden, "No enough right on this environment %s: ", envName)
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
			var errlb error
			bn, errlb := pipeline.GetLastBuildNumberInTx(api.mustDB(), p.ID, a.ID, env.ID)
			if errlb != nil {
				return sdk.WrapError(sdk.ErrNoPipelineBuild, "Cannot load last build number for %s: %s", pipelineName, errlb)
			}
			buildNumber = bn
		} else {
			var errpi error
			buildNumber, errpi = strconv.ParseInt(buildNumberS, 10, 64)
			if errpi != nil {
				return sdk.WrapError(errpi, "Cannot parse build number %s", buildNumberS)
			}
		}

		// load pipeline_build.id
		pb, errlpb := pipeline.LoadPipelineBuildByApplicationPipelineEnvBuildNumber(api.mustDB(), a.ID, p.ID, env.ID, buildNumber)
		if errlpb != nil {
			return sdk.WrapError(errlpb, "Cannot load pipeline build")
		}

		tests, errltr := pipeline.LoadTestResults(api.mustDB(), pb.ID)
		if errltr != nil {
			return sdk.WrapError(errltr, "Cannot load test results")
		}

		return service.WriteJSON(w, tests, http.StatusOK)
	}
}
