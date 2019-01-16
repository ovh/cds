package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/artifact"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) downloadArtifactHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		artifactID, errAtoi := requestVarInt(r, "id")
		if errAtoi != nil {
			return sdk.WrapError(errAtoi, "Cannot get artifact ID")
		}

		// Load artifact
		art, err := artifact.LoadArtifact(api.mustDB(), int64(artifactID))
		if err != nil {
			return sdk.WrapError(err, "Cannot load artifact")
		}

		f, err := objectstore.Fetch(art)
		if err != nil {
			return sdk.WrapError(err, "Cannot fetch artifact")
		}

		if _, err := io.Copy(w, f); err != nil {
			_ = f.Close()
			return sdk.WrapError(err, "Cannot stream artifact")
		}

		if err := f.Close(); err != nil {
			return sdk.WrapError(err, "Cannot close artifact")
		}

		w.Header().Add("Content-Type", "application/octet-stream")
		w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", art.Name))
		return nil
	}
}

func (api *API) listArtifactsBuildHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		pipelineName := vars["permPipelineKey"]
		appName := vars["permApplicationName"]
		buildNumberString := vars["buildNumber"]
		envName := r.FormValue("envName")

		// Load pipeline
		p, errP := pipeline.LoadPipeline(api.mustDB(), key, pipelineName, false)
		if errP != nil {
			return sdk.WrapError(errP, "Cannot load pipeline %s", pipelineName)
		}

		// Load application
		a, errA := application.LoadByName(api.mustDB(), api.Cache, key, appName, deprecatedGetUser(ctx))
		if errA != nil {
			return sdk.WrapError(errA, "Cannot load application %s", appName)
		}

		var env *sdk.Environment
		if envName == "" || envName == sdk.DefaultEnv.Name {
			env = &sdk.DefaultEnv
		} else {
			var errE error
			env, errE = environment.LoadEnvironmentByName(api.mustDB(), key, envName)
			if errE != nil {
				return sdk.WrapError(errE, "Cannot load environment %s", envName)
			}
		}

		if !permission.AccessToEnvironment(key, env.Name, deprecatedGetUser(ctx), permission.PermissionRead) {
			return sdk.WrapError(sdk.ErrForbidden, "No enough right on this environment %s", envName)
		}

		buildNumber, errI := strconv.ParseInt(buildNumberString, 10, 64)
		if errI != nil {
			return sdk.WrapError(errI, "BuildNumber must be an integer")
		}

		art, errArt := artifact.LoadArtifactsByBuildNumber(api.mustDB(), p.ID, a.ID, buildNumber, env.ID)
		if errArt != nil {
			return sdk.WrapError(errArt, "Cannot load artifacts")
		}

		return service.WriteJSON(w, art, http.StatusOK)
	}
}

func (api *API) listArtifactsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		key := vars["key"]
		pipelineName := vars["permPipelineKey"]
		appName := vars["permApplicationName"]
		tag := vars["tag"]
		envName := r.FormValue("envName")

		// Load pipeline
		p, errP := pipeline.LoadPipeline(api.mustDB(), key, pipelineName, false)
		if errP != nil {
			return sdk.WrapError(errP, "Cannot load pipeline %s", pipelineName)
		}

		// Load application
		a, errA := application.LoadByName(api.mustDB(), api.Cache, key, appName, deprecatedGetUser(ctx))
		if errA != nil {
			return sdk.WrapError(errA, "Cannot load application %s", appName)
		}

		var env *sdk.Environment
		if envName == "" || envName == sdk.DefaultEnv.Name || p.Type == sdk.BuildPipeline {
			env = &sdk.DefaultEnv
		} else {
			var errE error
			env, errE = environment.LoadEnvironmentByName(api.mustDB(), key, envName)
			if errE != nil {
				return sdk.WrapError(errE, "Cannot load environment %s", envName)
			}
		}

		if !permission.AccessToEnvironment(key, env.Name, deprecatedGetUser(ctx), permission.PermissionRead) {
			return sdk.WrapError(sdk.ErrForbidden, "No enough right on this environment %s", envName)
		}

		art, errArt := artifact.LoadArtifacts(api.mustDB(), p.ID, a.ID, env.ID, tag)
		if errArt != nil {
			return sdk.WrapError(errArt, "Cannot load artifacts")
		}

		if len(art) == 0 {
			return sdk.WrapError(sdk.ErrNotFound, "%s-%s-%s-%s/%s: not found", key, appName, env.Name, pipelineName, tag)
		}

		return service.WriteJSON(w, art, http.StatusOK)
	}
}

func (api *API) getArtifactsStoreHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, objectstore.Instance(), http.StatusOK)
	}
}

func (api *API) downloadArtifactDirectHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		hash := vars["hash"]

		art, err := artifact.LoadArtifactByHash(api.mustDB(), hash)
		if err != nil {
			return sdk.WrapError(err, "Could not load artifact with hash %s", hash)
		}

		w.Header().Add("Content-Type", "application/octet-stream")
		w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", art.Name))

		log.Debug("downloadArtifactDirectHandler: Serving %s/%s", art.GetPath(), art.GetName())
		f, err := objectstore.Fetch(art)
		if err != nil {
			return sdk.WrapError(err, "Cannot fetch artifact")
		}

		if _, err := io.Copy(w, f); err != nil {
			_ = f.Close()
			return sdk.WrapError(err, "Cannot stream artifact")
		}

		if err := f.Close(); err != nil {
			return sdk.WrapError(err, "Cannot close artifact")
		}

		return nil
	}
}

func (api *API) postArtifactWithTempURLHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if !objectstore.Instance().TemporaryURLSupported {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		store, ok := objectstore.Storage().(objectstore.DriverWithRedirect)
		if !ok {
			return sdk.WrapError(sdk.ErrForbidden, "Cast error")
		}

		vars := mux.Vars(r)
		proj := vars["key"]
		pip := vars["permPipelineKey"]
		app := vars["permApplicationName"]
		tag := vars["tag"]
		buildNumberString := vars["buildNumber"]
		envName := r.FormValue("envName")

		var env *sdk.Environment
		if envName == "" || envName == sdk.DefaultEnv.Name {
			env = &sdk.DefaultEnv
		} else {
			var errE error
			env, errE = environment.LoadEnvironmentByName(api.mustDB(), proj, envName)
			if errE != nil {
				return sdk.WrapError(errE, "Cannot load environment %s", envName)
			}
		}

		if !permission.AccessToEnvironment(proj, env.Name, deprecatedGetUser(ctx), permission.PermissionReadExecute) {
			return sdk.WrapError(sdk.ErrForbidden, "No enough right on this environment %s", env.Name)
		}

		buildNumber, errI := strconv.Atoi(buildNumberString)
		if errI != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "BuildNumber must be an integer: %v", errI)
		}

		hash, errG := generateHash()
		if errG != nil {
			return sdk.WrapError(errG, "Could not generate hash")
		}

		art := new(sdk.Artifact)
		if err := service.UnmarshalBody(r, art); err != nil {
			return sdk.WrapError(err, "Unable to unmarshal artifact")
		}

		art.DownloadHash = hash
		art.Project = proj
		art.Application = app
		art.Pipeline = pip
		art.Environment = env.Name
		art.BuildNumber = buildNumber
		art.Tag = tag

		url, key, err := store.StoreURL(art)
		if err != nil {
			return sdk.WrapError(err, "Could not generate hash")
		}

		art.TempURL = url
		art.TempURLSecretKey = key

		cacheKey := cache.Key("artifacts", art.GetPath(), art.GetName())
		api.Cache.SetWithTTL(cacheKey, art, 60*60) //Put this in cache for 1 hour

		return service.WriteJSON(w, art, http.StatusOK)
	}
}

func (api *API) postArtifactWithTempURLCallbackHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if !objectstore.Instance().TemporaryURLSupported {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		vars := mux.Vars(r)
		projKey := vars["key"]
		pipName := vars["permPipelineKey"]
		appName := vars["permApplicationName"]
		envName := r.FormValue("envName")

		art := new(sdk.Artifact)
		if err := service.UnmarshalBody(r, art); err != nil {
			return sdk.WrapError(err, "Unable to read artifact")
		}

		cacheKey := cache.Key("artifacts", art.GetPath(), art.GetName())
		cachedArt := new(sdk.Artifact)
		if !api.Cache.Get(cacheKey, cachedArt) {
			return sdk.WrapError(sdk.ErrNotFound, "Unable to find artifact")
		}

		if art.DownloadHash != cachedArt.DownloadHash {
			return sdk.WrapError(sdk.ErrForbidden, "Submitted artifact doesn't match")
		}

		var env *sdk.Environment
		if envName == "" || envName == sdk.DefaultEnv.Name {
			env = &sdk.DefaultEnv
		} else {
			var errE error
			env, errE = environment.LoadEnvironmentByName(api.mustDB(), projKey, envName)
			if errE != nil {
				return sdk.WrapError(errE, "Cannot load environment %s", envName)
			}
		}

		if !permission.AccessToEnvironment(projKey, env.Name, deprecatedGetUser(ctx), permission.PermissionReadExecute) {
			return sdk.WrapError(sdk.ErrForbidden, "No enough right on this environment %s", env.Name)
		}

		pip, errpip := pipeline.LoadPipeline(api.mustDB(), projKey, pipName, false)
		if errpip != nil {
			return sdk.WrapError(errpip, "Unable to load pipeline %s/%s", projKey, pipName)
		}

		app, errapp := application.LoadByName(api.mustDB(), api.Cache, projKey, appName, deprecatedGetUser(ctx))
		if errapp != nil {
			return sdk.WrapError(errapp, "Unable to load application %s/%s", projKey, appName)
		}

		if err := artifact.InsertArtifact(api.mustDB(), pip.ID, app.ID, env.ID, *art); err != nil {
			return sdk.WrapError(err, "Unable to save artifact")
		}

		return nil
	}
}

func generateHash() (string, error) {
	size := 128
	bs := make([]byte, size)
	if _, err := rand.Read(bs); err != nil {
		return "", sdk.WrapError(err, "rand.Read failed")
	}
	str := hex.EncodeToString(bs)
	token := []byte(str)[0:size]

	log.Debug("api> generateHash> new generated id: %s", token)
	return string(token), nil
}
