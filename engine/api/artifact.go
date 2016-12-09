package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/artifact"
	"github.com/ovh/cds/engine/api/context"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

func uploadArtifactHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	project := vars["key"]
	pipelineName := vars["permPipelineKey"]
	appName := vars["permApplicationName"]
	tag := vars["tag"]
	buildNumberString := vars["buildNumber"]
	fileName := r.Header.Get(sdk.ArtifactFileName)

	//parse the multipart form in the request
	err := r.ParseMultipartForm(100000)
	if err != nil {
		log.Warning("uploadArtifactHandler: Error parsing multipart form: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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
		log.Warning("uploadArtifactHandler> %s header is not set", sdk.ArtifactFileName)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	p, err := pipeline.LoadPipeline(db, project, pipelineName, false)
	if err != nil {
		log.Warning("uploadArtifactHandler> cannot load pipeline %s-%s: %s\n", project, pipelineName, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	a, err := application.LoadApplicationByName(db, project, appName)
	if err != nil {
		log.Warning("uploadArtifactHandler> cannot load application %s-%s: %s\n", project, appName, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var env *sdk.Environment
	if envName == "" || envName == sdk.DefaultEnv.Name {
		env = &sdk.DefaultEnv
	} else {
		env, err = environment.LoadEnvironmentByName(db, project, envName)
		if err != nil {
			log.Warning("uploadArtifactHandler> Cannot load environment %s: %s\n", envName, err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}

	if env.ID != sdk.DefaultEnv.ID && !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionReadExecute) {
		log.Warning("uploadArtifactHandler> No enought right on this environment %s: \n", envName)
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	buildNumber, err := strconv.Atoi(buildNumberString)
	if err != nil {
		log.Warning("uploadArtifactHandler> BuildNumber must be an integer: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	hash, err := generateHash()
	if err != nil {
		log.Warning("uploadArtifactHandler> Could not generate hash: %s\n", err)
		WriteError(w, r, err)
		return
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
			log.Warning("uploadArtifactHandler> cannot open file: %s\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		err = artifact.SaveFile(db, p, a, art, file, env)
		if err != nil {
			log.Warning("uploadArtifactHandler> cannot save file: %s\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			file.Close()
			return
		}
		file.Close()
	}
}

func downloadArtifactHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	project := vars["key"]
	pipelineName := vars["permPipelineKey"]
	appName := vars["permApplicationName"]
	artifactIDS := vars["id"]

	// Load pipeline
	_, err := pipeline.LoadPipeline(db, project, pipelineName, false)
	if err != nil {
		log.Warning("DownloadArtifactHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Load application
	_, err = application.LoadApplicationByName(db, project, appName)
	if err != nil {
		log.Warning("DownloadArtifactHandler> Cannot load application %s: %s\n", appName, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	artifactID, err := strconv.Atoi(artifactIDS)
	if err != nil {
		log.Warning("DownloadArtifactHandler> Cannot convert '%s' into int: %s\n", artifactIDS, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Load artifact
	art, err := artifact.LoadArtifact(db, int64(artifactID))
	if err != nil {
		log.Warning("downloadArtifactHandler> Cannot load artifact %d: %s\n", artifactID, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Info("downloadArtifactHandler: Serving %+v\n", art)
	err = artifact.StreamFile(w, *art)
	if err != nil {
		log.Warning("downloadArtifactHandler: Cannot stream artifact %s-%s-%s-%s-%s file: %s\n", art.Project, art.Application, art.Environment, art.Pipeline, art.Tag, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "application/octet-stream")
	w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", art.Name))
}

func listArtifactsBuildHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	project := vars["key"]
	pipelineName := vars["permPipelineKey"]
	appName := vars["permApplicationName"]
	buildNumberString := vars["buildNumber"]

	envName := r.FormValue("envName")

	// Load pipeline
	p, err := pipeline.LoadPipeline(db, project, pipelineName, false)
	if err != nil {
		log.Warning("listArtifactsBuildHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Load application
	a, err := application.LoadApplicationByName(db, project, appName)
	if err != nil {
		log.Warning("listArtifactsBuildHandler> Cannot load application %s: %s\n", appName, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var env *sdk.Environment
	if envName == "" || envName == sdk.DefaultEnv.Name {
		env = &sdk.DefaultEnv
	} else {
		env, err = environment.LoadEnvironmentByName(db, project, envName)
		if err != nil {
			log.Warning("listArtifactsBuildHandler> Cannot load environment %s: %s\n", envName, err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}

	if env.ID != sdk.DefaultEnv.ID && !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionRead) {
		log.Warning("listArtifactsBuildHandler> No enought right on this environment %s: \n", envName)
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	buildNumber, err := strconv.ParseInt(buildNumberString, 10, 64)
	if err != nil {
		log.Warning("listArtifactsBuildHandler> BuildNumber must be an integer: %s\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	art, err := artifact.LoadArtifactsByBuildNumber(db, p.ID, a.ID, buildNumber, env.ID)
	if err != nil {
		log.Warning("listArtifactsBuildHandler> Cannot load artifacts: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	WriteJSON(w, r, art, http.StatusOK)
}

func listArtifactsHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	project := vars["key"]
	pipelineName := vars["permPipelineKey"]
	appName := vars["permApplicationName"]
	tag := vars["tag"]

	envName := r.FormValue("envName")

	// Load pipeline
	p, err := pipeline.LoadPipeline(db, project, pipelineName, false)
	if err != nil {
		log.Warning("listArtifactsHandler> Cannot load pipeline %s: %s\n", pipelineName, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Load application
	a, err := application.LoadApplicationByName(db, project, appName)
	if err != nil {
		log.Warning("listArtifactsHandler> Cannot load application %s: %s\n", appName, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var env *sdk.Environment
	if envName == "" || envName == sdk.DefaultEnv.Name || p.Type == sdk.BuildPipeline {
		env = &sdk.DefaultEnv
	} else {
		env, err = environment.LoadEnvironmentByName(db, project, envName)
		if err != nil {
			log.Warning("listArtifactsHandler> Cannot load environment %s: %s\n", envName, err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}

	if env.ID != sdk.DefaultEnv.ID && !permission.AccessToEnvironment(env.ID, c.User, permission.PermissionRead) {
		log.Warning("listArtifactsHandler> No enought right on this environment %s: \n", envName)
		WriteError(w, r, sdk.ErrForbidden)
		return
	}

	art, err := artifact.LoadArtifacts(db, p.ID, a.ID, env.ID, tag)
	if err != nil {
		log.Warning("listArtifactsHandler> Cannot load artifacts: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(art) == 0 {
		log.Warning("listArtifactHandler> %s-%s-%s-%s/%s: not found\n", project, appName, env.Name, pipelineName, tag)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	WriteJSON(w, r, art, http.StatusOK)
}

func downloadArtifactDirectHandler(w http.ResponseWriter, r *http.Request, db *sql.DB, c *context.Context) {
	vars := mux.Vars(r)
	hash := vars["hash"]

	art, err := artifact.LoadArtifactByHash(db, hash)
	if err != nil {
		log.Warning("downloadArtifactDirectHandler> Could not load artifact with hash %s: %s\n", hash, err)
		WriteError(w, r, err)
		return
	}

	w.Header().Add("Content-Type", "application/octet-stream")
	w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", art.Name))

	log.Info("downloadArtifactDirectHandler: Serving %+v\n", art)
	err = artifact.StreamFile(w, *art)
	if err != nil {
		log.Warning("downloadArtifactDirectHandler: Cannot stream artifact %s-%s-%s-%s-%s file: %s\n", art.Project, art.Application, art.Environment, art.Pipeline, art.Tag, err)
		WriteError(w, r, err)
		return
	}
}

func generateHash() (string, error) {
	size := 128
	bs := make([]byte, size)
	_, err := rand.Read(bs)
	if err != nil {
		log.Critical("generateID: rand.Read failed: %s\n", err)
		return "", err
	}
	str := hex.EncodeToString(bs)
	token := []byte(str)[0:size]

	log.Debug("generateID: new generated id: %s\n", token)
	return string(token), nil
}
