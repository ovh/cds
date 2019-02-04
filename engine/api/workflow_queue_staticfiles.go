package api

import (
	"context"
	"mime"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getStaticFilesStoreHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//TODO: to delete when swift will be available with auto-extract and temporary url middlewares
		store := sdk.ArtifactsStore{}
		store.TemporaryURLSupported = false
		return service.WriteJSON(w, store, http.StatusOK)
	}
}

func (api *API) postWorkflowJobStaticFilesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		db := api.mustDB()
		// Load  Existing workflow Run Job
		nodeJobRunID, errI := requestVarInt(r, "permID")
		if errI != nil {
			return sdk.WrapError(sdk.ErrInvalidID, "Invalid node job run ID")
		}

		vars := mux.Vars(r)
		name := vars["name"]

		_, params, errM := mime.ParseMediaType(r.Header.Get("Content-Disposition"))
		if errM != nil {
			return sdk.WrapError(errM, "Cannot read Content Disposition header")
		}
		fileName := params["filename"]

		//parse the multipart form in the request
		if err := r.ParseMultipartForm(100000); err != nil {
			return sdk.WrapError(err, "Error parsing multipart form")
		}
		//get a ref to the parsed multipart form
		m := r.MultipartForm

		var entrypoint string
		if len(m.Value["entrypoint"]) > 0 {
			entrypoint = m.Value["entrypoint"][0]
		}

		var staticKey string
		if len(m.Value["static_key"]) > 0 {
			staticKey = m.Value["static_key"][0]
		}

		if fileName == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "Content-Disposition header is not set")
		}

		nodeJobRun, errJ := workflow.LoadNodeJobRun(db, api.Cache, nodeJobRunID)
		if errJ != nil {
			return sdk.WrapError(errJ, "Cannot load node job run")
		}

		nodeRun, errNr := workflow.LoadNodeRunByID(db, nodeJobRun.WorkflowNodeRunID, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
		if errNr != nil {
			return sdk.WrapError(errNr, "Cannot load node run")
		}

		staticFile := sdk.StaticFiles{
			Name:         name,
			EntryPoint:   entrypoint,
			StaticKey:    staticKey,
			WorkflowID:   nodeRun.WorkflowID,
			NodeRunID:    nodeJobRun.WorkflowNodeRunID,
			NodeJobRunID: nodeJobRunID,
		}

		if staticFile.StaticKey != "" {
			if err := api.SharedStorage.Delete(&staticFile); err != nil {
				return sdk.WrapError(err, "Cannot delete existing static files")
			}
		}

		files := m.File[fileName]
		if len(files) == 1 {
			file, err := files[0].Open()
			if err != nil {
				return sdk.WrapError(err, "cannot open file")
			}
			defer file.Close()

			publicURL, err := api.SharedStorage.ServeStaticFiles(&staticFile, staticFile.EntryPoint, file)
			if err != nil {
				return sdk.WrapError(err, "Cannot serve static files in store")
			}
			staticFile.PublicURL = publicURL
		}

		if err := workflow.InsertStaticFiles(db, &staticFile); err != nil {
			_ = api.SharedStorage.Delete(&staticFile)
			return sdk.WrapError(err, "Cannot insert static files in database")
		}
		return service.WriteJSON(w, staticFile, http.StatusOK)
	}
}

func (api *API) postWorkflowJobStaticFilesWithTempURLHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if !api.SharedStorage.TemporaryURLSupported() {
			return sdk.ErrForbidden
		}

		store, ok := api.SharedStorage.(objectstore.DriverWithRedirect)
		if !ok {
			return sdk.NewErrorFrom(sdk.ErrForbidden, "A cast error occured (seems that your objectstore doesn't support redirect)")
		}

		// Load existing workflow Run Job
		id, errI := requestVarInt(r, "permID")
		if errI != nil {
			return sdk.WrapError(errI, "Invalid node job run ID")
		}

		vars := mux.Vars(r)
		name := vars["name"]

		var staticfile sdk.StaticFiles
		if err := service.UnmarshalBody(r, &staticfile); err != nil {
			return sdk.WithStack(err)
		}

		if staticfile.StaticKey != "" {
			if err := api.SharedStorage.Delete(&staticfile); err != nil {
				return sdk.WrapError(err, "Cannot delete existing static files")
			}
		}

		nodeJobRun, errJ := workflow.LoadNodeJobRun(api.mustDB(), api.Cache, id)
		if errJ != nil {
			return sdk.WrapError(errJ, "Cannot load node job run")
		}
		staticfile.NodeRunID = nodeJobRun.WorkflowNodeRunID
		staticfile.Name = name

		nodeRun, errNr := workflow.LoadNodeRunByID(api.mustDB(), nodeJobRun.WorkflowNodeRunID, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
		if errNr != nil {
			return sdk.WrapError(errNr, "Cannot load node run")
		}
		staticfile.WorkflowID = nodeRun.WorkflowID

		var retryURL = 10
		var url, key string
		var errorStoreURL error
		for i := 0; i < retryURL; i++ {
			url, key, errorStoreURL = store.ServeStaticFilesURL(&staticfile, staticfile.EntryPoint)
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

		staticfile.TempURL = url
		staticfile.SecretKey = key

		cacheKey := cache.Key("workflows:staticfiles", staticfile.GetPath(), staticfile.GetName())
		api.Cache.SetWithTTL(cacheKey, staticfile, 60*60) //Put this in cache for 1 hour

		return service.WriteJSON(w, staticfile, http.StatusOK)
	}
}

func (api *API) postWorkflowJobStaticFilesWithTempURLCallbackHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if !api.SharedStorage.TemporaryURLSupported() {
			return sdk.ErrForbidden
		}

		var staticfile sdk.StaticFiles
		if err := service.UnmarshalBody(r, &staticfile); err != nil {
			return err
		}

		cacheKey := cache.Key("workflows:staticfiles", staticfile.GetPath(), staticfile.GetName())
		var cachedStaticFiles sdk.StaticFiles
		if !api.Cache.Get(cacheKey, &cachedStaticFiles) {
			return sdk.WrapError(sdk.ErrNotFound, "Unable to find artifact, key:%s", cacheKey)
		}

		if !staticfile.Equal(cachedStaticFiles) {
			return sdk.WrapError(sdk.ErrForbidden, "Submitted artifact doesn't match, key:%s art:%v cachedStaticFiles:%v", cacheKey, staticfile, cachedStaticFiles)
		}

		store, ok := api.SharedStorage.(objectstore.DriverWithRedirect)
		if !ok {
			return sdk.WrapError(sdk.ErrForbidden, "cast error")
		}

		publicURL, errP := store.GetPublicURL(&cachedStaticFiles)
		if errP != nil {
			return sdk.WrapError(errP, "Cannot get public URL")
		}
		staticfile.PublicURL = publicURL

		if err := workflow.InsertStaticFiles(api.mustDB(), &staticfile); err != nil {
			_ = api.SharedStorage.Delete(&staticfile)
			return sdk.WrapError(err, "Cannot update workflow node run")
		}

		return service.WriteJSON(w, staticfile, http.StatusOK)
	}
}
