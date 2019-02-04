package api

import (
	"context"
	"encoding/base64"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) postWorkflowJobArtifactHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		ref := vars["ref"]
		integrationName := vars["integrationName"]
		// TODO yesnault projectKey := vars["projectKey"]

		_, params, errM := mime.ParseMediaType(r.Header.Get("Content-Disposition"))
		if errM != nil {
			return sdk.WrapError(errM, "postWorkflowJobArtifactHandler> Cannot read Content Disposition header")
		}

		fileName := params["filename"]

		//parse the multipart form in the request
		if err := r.ParseMultipartForm(100000); err != nil {
			return sdk.WrapError(err, "postWorkflowJobArtifactHandler: Error parsing multipart form")

		}
		//get a ref to the parsed multipart form
		m := r.MultipartForm

		var sizeStr, permStr, md5sum, sha512sum, nodeJobRunIDStr string
		if len(m.Value["size"]) > 0 {
			sizeStr = m.Value["size"][0]
		}
		if len(m.Value["perm"]) > 0 {
			permStr = m.Value["perm"][0]
		}
		if len(m.Value["md5sum"]) > 0 {
			md5sum = m.Value["md5sum"][0]
		}
		if len(m.Value["sha512sum"]) > 0 {
			sha512sum = m.Value["sha512sum"][0]
		}
		if len(m.Value["nodeJobRunID"]) > 0 {
			nodeJobRunIDStr = m.Value["nodeJobRunID"][0]
		}
		nodeJobRunID, errI := strconv.ParseInt(nodeJobRunIDStr, 10, 64)
		if errI != nil {
			return sdk.WrapError(sdk.ErrInvalidID, "postWorkflowJobArtifactHandler> Invalid node job run ID")
		}

		if fileName == "" {
			log.Warning("uploadArtifactHandler> %s header is not set", "Content-Disposition")
			return sdk.WrapError(sdk.ErrWrongRequest, "postWorkflowJobArtifactHandler> %s header is not set", "Content-Disposition")
		}

		nodeJobRun, errJ := workflow.LoadNodeJobRun(api.mustDB(), api.Cache, nodeJobRunID)
		if errJ != nil {
			return sdk.WrapError(errJ, "Cannot load node job run")
		}

		nodeRun, errR := workflow.LoadNodeRunByID(api.mustDB(), nodeJobRun.WorkflowNodeRunID, workflow.LoadRunOptions{WithArtifacts: true, DisableDetailledNodeRun: true})
		if errR != nil {
			return sdk.WrapError(errR, "Cannot load node run")
		}

		hash, errG := generateHash()
		if errG != nil {
			return sdk.WrapError(errG, "postWorkflowJobArtifactHandler> Could not generate hash")
		}

		var size int64
		var perm uint64

		if sizeStr != "" {
			size, _ = strconv.ParseInt(sizeStr, 10, 64)
		}

		if permStr != "" {
			perm, _ = strconv.ParseUint(permStr, 10, 32)
		}

		tag, errT := base64.RawURLEncoding.DecodeString(ref)
		if errT != nil {
			return sdk.WrapError(errT, "postWorkflowJobArtifactHandler> Cannot decode ref")
		}

		// TODO yesnault check IntegrationName on projectKey

		art := sdk.WorkflowNodeRunArtifact{
			Name:              fileName,
			Tag:               string(tag),
			Ref:               ref,
			DownloadHash:      hash,
			Size:              size,
			Perm:              uint32(perm),
			MD5sum:            md5sum,
			SHA512sum:         sha512sum,
			WorkflowNodeRunID: nodeRun.ID,
			WorkflowID:        nodeRun.WorkflowRunID,
			Created:           time.Now(),
			IntegrationName:   integrationName,
		}

		files := m.File[fileName]
		if len(files) == 1 {
			file, err := files[0].Open()
			if err != nil {
				file.Close()
				return sdk.WrapError(err, "cannot open file")
			}

			objectPath, err := api.SharedStorage.Store(&art, file)
			if err != nil {
				file.Close()
				return sdk.WrapError(err, "Cannot store artifact")
			}
			log.Debug("objectpath=%s\n", objectPath)
			art.ObjectPath = objectPath
			file.Close()
		}

		nodeRun.Artifacts = append(nodeRun.Artifacts, art)
		if err := workflow.InsertArtifact(api.mustDB(), &art); err != nil {
			_ = api.SharedStorage.Delete(&art)
			return sdk.WrapError(err, "Cannot update workflow node run")
		}
		return nil
	}
}

func (api *API) postWorkflowJobArtifacWithTempURLHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if !api.SharedStorage.TemporaryURLSupported() {
			return sdk.WrapError(sdk.ErrForbidden, "postWorkflowJobArtifacWithTempURLHandler")
		}

		store, ok := api.SharedStorage.(objectstore.DriverWithRedirect)
		if !ok {
			return sdk.WrapError(sdk.ErrForbidden, "postWorkflowJobArtifacWithTempURLHandler > cast error")
		}

		vars := mux.Vars(r)
		ref := vars["ref"]

		// TODO yesnault check integrationName on projectKey

		// integrationName := vars["integrationName"]
		// projectKey := vars["projectKey"]

		hash, errG := generateHash()
		if errG != nil {
			return sdk.WrapError(errG, "postWorkflowJobArtifacWithTempURLHandler> Could not generate hash")
		}

		art := sdk.WorkflowNodeRunArtifact{}
		if err := service.UnmarshalBody(r, &art); err != nil {
			return sdk.WithStack(err)
		}

		nodeJobRun, errJ := workflow.LoadNodeJobRun(api.mustDB(), api.Cache, art.WorkflowNodeJobRunID)
		if errJ != nil {
			return sdk.WrapError(errJ, "postWorkflowJobArtifacWithTempURLHandler> Cannot load node job run")
		}

		nodeRun, errR := workflow.LoadNodeRunByID(api.mustDB(), nodeJobRun.WorkflowNodeRunID, workflow.LoadRunOptions{WithArtifacts: true, DisableDetailledNodeRun: true})
		if errR != nil {
			return sdk.WrapError(errR, "postWorkflowJobArtifacWithTempURLHandler> Cannot load node run")
		}

		tag, errT := base64.RawURLEncoding.DecodeString(ref)
		if errT != nil {
			return sdk.WrapError(errT, "postWorkflowJobArtifacWithTempURLHandler> Cannot decode ref")
		}

		art.WorkflowID = nodeRun.WorkflowRunID
		art.WorkflowNodeRunID = nodeRun.ID
		art.DownloadHash = hash
		art.Tag = string(tag)
		art.Ref = ref

		var retryURL = 10
		var url, key string
		var errorStoreURL error

		for i := 0; i < retryURL; i++ {
			url, key, errorStoreURL = store.StoreURL(&art)
			if errorStoreURL != nil {
				log.Warning("Error on store.StoreURL: %v - Try %d/%d", errorStoreURL, i, retryURL)
			} else {
				// no error
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		if url == "" || key == "" {
			return sdk.WrapError(errorStoreURL, "Could not generate hash after %d attempts", retryURL)
		}

		art.TempURL = url
		art.TempURLSecretKey = key

		cacheKey := cache.Key("workflows:artifacts", art.GetPath(), art.GetName())
		api.Cache.SetWithTTL(cacheKey, art, 60*60) //Put this in cache for 1 hour

		return service.WriteJSON(w, art, http.StatusOK)
	}
}

func (api *API) postWorkflowJobArtifactWithTempURLCallbackHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if !api.SharedStorage.TemporaryURLSupported() {
			return sdk.WrapError(sdk.ErrForbidden, "postWorkflowJobArtifactWithTempURLCallbackHandler")
		}

		art := sdk.WorkflowNodeRunArtifact{}
		if err := service.UnmarshalBody(r, &art); err != nil {
			return err
		}

		cacheKey := cache.Key("workflows:artifacts", art.GetPath(), art.GetName())
		cachedArt := sdk.WorkflowNodeRunArtifact{}
		if !api.Cache.Get(cacheKey, &cachedArt) {
			return sdk.WrapError(sdk.ErrNotFound, "postWorkflowJobArtifactWithTempURLCallbackHandler> Unable to find artifact, key:%s", cacheKey)
		}

		if !art.Equal(cachedArt) {
			return sdk.WrapError(sdk.ErrForbidden, "postWorkflowJobArtifactWithTempURLCallbackHandler> Submitted artifact doesn't match, key:%s art:%v cachedArt:%v", cacheKey, art, cachedArt)
		}

		nodeRun, errR := workflow.LoadNodeRunByID(api.mustDB(), art.WorkflowNodeRunID, workflow.LoadRunOptions{WithArtifacts: true, DisableDetailledNodeRun: true})
		if errR != nil {
			return sdk.WrapError(errR, "Cannot load node run")
		}

		nodeRun.Artifacts = append(nodeRun.Artifacts, art)
		if err := workflow.InsertArtifact(api.mustDB(), &art); err != nil {
			_ = api.SharedStorage.Delete(&art)
			return sdk.WrapError(err, "Cannot update workflow node run")
		}

		return nil
	}
}
