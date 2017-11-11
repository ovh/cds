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
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) uploadArtifactHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		project := vars["key"]
		pipelineName := vars["permPipelineKey"]
		appName := vars["permApplicationName"]
		tag := vars["tag"]
		buildNumberString := vars["buildNumber"]
		fileName := r.Header.Get(sdk.ArtifactFileName)

		//parse the multipart form in the request
		if err := r.ParseMultipartForm(100000); err != nil {
			return sdk.WrapError(err, "uploadArtifactHandler>  Error parsing multipart form")
		}

		//get a ref to the parsed multipart form
		m := r.MultipartForm
		envName := m.Value["env"][0]

		var sizeStr, permStr, md5sum string
		if len(m.Value["size"]) > 0 {
			sizeStr = m.Value["size"][0]
		}
		if len(m.Value["perm"]) > 0 {
			permStr = m.Value["perm"][0]
		}
		if len(m.Value["md5sum"]) > 0 {
			md5sum = m.Value["md5sum"][0]
		}

		if fileName == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "uploadArtifactHandler> %s header is not set", sdk.ArtifactFileName)
		}

		p, errP := pipeline.LoadPipeline(api.mustDB(), project, pipelineName, false)
		if errP != nil {
			return sdk.WrapError(errP, "uploadArtifactHandler> cannot load pipeline %s-%s", project, pipelineName)
		}

		a, errA := application.LoadByName(api.mustDB(), api.Cache, project, appName, getUser(ctx))
		if errA != nil {
			return sdk.WrapError(errA, "uploadArtifactHandler> cannot load application %s-%s", project, appName)
		}

		var env *sdk.Environment
		if envName == "" || envName == sdk.DefaultEnv.Name {
			env = &sdk.DefaultEnv
		} else {
			var errE error
			env, errE = environment.LoadEnvironmentByName(api.mustDB(), project, envName)
			if errE != nil {
				return sdk.WrapError(errE, "uploadArtifactHandler> Cannot load environment %s", envName)
			}
		}

		if !permission.AccessToEnvironment(env.ID, getUser(ctx), permission.PermissionReadExecute) {
			return sdk.WrapError(sdk.ErrForbidden, "uploadArtifactHandler> No enought right on this environment %s")
		}

		buildNumber, errI := strconv.Atoi(buildNumberString)
		if errI != nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "uploadArtifactHandler> BuildNumber must be an integer: %s", errI)
		}

		hash, errG := generateHash()
		if errG != nil {
			return sdk.WrapError(errG, "uploadArtifactHandler> Could not generate hash")
		}

		var size int64
		var perm uint64

		if sizeStr != "" {
			size, _ = strconv.ParseInt(sizeStr, 10, 64)
		}

		if permStr != "" {
			perm, _ = strconv.ParseUint(permStr, 10, 32)
		}

		art := sdk.Artifact{
			Name:         fileName,
			Project:      project,
			Pipeline:     pipelineName,
			Application:  a.Name,
			Tag:          tag,
			Environment:  envName,
			BuildNumber:  buildNumber,
			DownloadHash: hash,
			Size:         size,
			Perm:         uint32(perm),
			MD5sum:       md5sum,
		}

		files := m.File[fileName]
		for i := range files {
			file, err := files[i].Open()
			if err != nil {
				return sdk.WrapError(err, "uploadArtifactHandler> cannot open file")
			}

			if err := artifact.SaveFile(api.mustDB(), p, a, art, file, env); err != nil {
				file.Close()
				return sdk.WrapError(err, "uploadArtifactHandler> cannot save file")
			}
			file.Close()
		}
		return nil
	}
}

func (api *API) downloadArtifactHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		artifactID, errAtoi := requestVarInt(r, "id")
		if errAtoi != nil {
			return sdk.WrapError(errAtoi, "DownloadArtifactHandler> Cannot get artifact ID")
		}

		// Load artifact
		art, err := artifact.LoadArtifact(api.mustDB(), int64(artifactID))
		if err != nil {
			return sdk.WrapError(err, "downloadArtifactHandler> Cannot load artifact")
		}

		f, err := objectstore.FetchArtifact(art)
		if err != nil {
			return sdk.WrapError(err, "downloadArtifactHandler> Cannot fetch artifact")
		}

		if _, err := io.Copy(w, f); err != nil {
			return sdk.WrapError(err, "downloadArtifactHandler> Cannot stream artifact")
		}

		if err := f.Close(); err != nil {
			return sdk.WrapError(err, "downloadArtifactHandler> Cannot close artifact")
		}

		w.Header().Add("Content-Type", "application/octet-stream")
		w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", art.Name))
		return nil
	}
}

func (api *API) listArtifactsBuildHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		project := vars["key"]
		pipelineName := vars["permPipelineKey"]
		appName := vars["permApplicationName"]
		buildNumberString := vars["buildNumber"]
		envName := r.FormValue("envName")

		// Load pipeline
		p, errP := pipeline.LoadPipeline(api.mustDB(), project, pipelineName, false)
		if errP != nil {
			return sdk.WrapError(errP, "listArtifactsBuildHandler> Cannot load pipeline %s", pipelineName)
		}

		// Load application
		a, errA := application.LoadByName(api.mustDB(), api.Cache, project, appName, getUser(ctx))
		if errA != nil {
			return sdk.WrapError(errA, "listArtifactsBuildHandler> Cannot load application %s", appName)
		}

		var env *sdk.Environment
		if envName == "" || envName == sdk.DefaultEnv.Name {
			env = &sdk.DefaultEnv
		} else {
			var errE error
			env, errE = environment.LoadEnvironmentByName(api.mustDB(), project, envName)
			if errE != nil {
				return sdk.WrapError(errE, "listArtifactsBuildHandler> Cannot load environment %s", envName)
			}
		}

		if !permission.AccessToEnvironment(env.ID, getUser(ctx), permission.PermissionRead) {
			return sdk.WrapError(sdk.ErrForbidden, "listArtifactsBuildHandler> No enought right on this environment %s", envName)
		}

		buildNumber, errI := strconv.ParseInt(buildNumberString, 10, 64)
		if errI != nil {
			return sdk.WrapError(errI, "listArtifactsBuildHandler> BuildNumber must be an integer")
		}

		art, errArt := artifact.LoadArtifactsByBuildNumber(api.mustDB(), p.ID, a.ID, buildNumber, env.ID)
		if errArt != nil {
			return sdk.WrapError(errArt, "listArtifactsBuildHandler> Cannot load artifacts")
		}

		return WriteJSON(w, r, art, http.StatusOK)
	}
}

func (api *API) listArtifactsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		project := vars["key"]
		pipelineName := vars["permPipelineKey"]
		appName := vars["permApplicationName"]
		tag := vars["tag"]
		envName := r.FormValue("envName")

		// Load pipeline
		p, errP := pipeline.LoadPipeline(api.mustDB(), project, pipelineName, false)
		if errP != nil {
			return sdk.WrapError(errP, "listArtifactsHandler> Cannot load pipeline %s", pipelineName)
		}

		// Load application
		a, errA := application.LoadByName(api.mustDB(), api.Cache, project, appName, getUser(ctx))
		if errA != nil {
			return sdk.WrapError(errA, "listArtifactsHandler> Cannot load application %s", appName)
		}

		var env *sdk.Environment
		if envName == "" || envName == sdk.DefaultEnv.Name || p.Type == sdk.BuildPipeline {
			env = &sdk.DefaultEnv
		} else {
			var errE error
			env, errE = environment.LoadEnvironmentByName(api.mustDB(), project, envName)
			if errE != nil {
				return sdk.WrapError(errE, "listArtifactsHandler> Cannot load environment %s", envName)
			}
		}

		if !permission.AccessToEnvironment(env.ID, getUser(ctx), permission.PermissionRead) {
			return sdk.WrapError(sdk.ErrForbidden, "listArtifactsHandler> No enought right on this environment %s", envName)
		}

		art, errArt := artifact.LoadArtifacts(api.mustDB(), p.ID, a.ID, env.ID, tag)
		if errArt != nil {
			return sdk.WrapError(errArt, "listArtifactsHandler> Cannot load artifacts")
		}

		if len(art) == 0 {
			return sdk.WrapError(sdk.ErrNotFound, "listArtifactHandler> %s-%s-%s-%s/%s: not found", project, appName, env.Name, pipelineName, tag)
		}

		return WriteJSON(w, r, art, http.StatusOK)
	}
}

func (api *API) downloadArtifactDirectHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		hash := vars["hash"]

		art, err := artifact.LoadArtifactByHash(api.mustDB(), hash)
		if err != nil {
			return sdk.WrapError(err, "downloadArtifactDirectHandler> Could not load artifact with hash %s", hash)
		}

		log.Debug("downloadArtifactDirectHandler: Serving %+v", art)
		f, err := objectstore.FetchArtifact(art)
		if err != nil {
			return sdk.WrapError(err, "downloadArtifactDirectHandler> Cannot fetch artifact")
		}

		if _, err := io.Copy(w, f); err != nil {
			return sdk.WrapError(err, "downloadArtifactDirectHandler> Cannot stream artifact")
		}

		if err := f.Close(); err != nil {
			return sdk.WrapError(err, "downloadArtifactDirectHandler> Cannot close artifact")
		}

		w.Header().Add("Content-Type", "application/octet-stream")
		w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", art.Name))

		return nil
	}
}

func generateHash() (string, error) {
	size := 128
	bs := make([]byte, size)
	if _, err := rand.Read(bs); err != nil {
		return "", sdk.WrapError(err, "generateHash> rand.Read failed")
	}
	str := hex.EncodeToString(bs)
	token := []byte(str)[0:size]

	log.Debug("generateHash> new generated id: %s", token)
	return string(token), nil
}
