package api

import (
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) postWorkflowJobStaticFilesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if _, isWorker := api.isWorker(ctx); !isWorker {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		vars := mux.Vars(r)
		name := vars["name"]

		_, params, err := mime.ParseMediaType(r.Header.Get("Content-Disposition"))
		if err != nil {
			return sdk.WrapError(err, "cannot read Content Disposition header")
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

		var nodeJobRunIDStr string
		if len(m.Value["nodeJobRunID"]) > 0 {
			nodeJobRunIDStr = m.Value["nodeJobRunID"][0]
		}
		nodeJobRunID, err := strconv.ParseInt(nodeJobRunIDStr, 10, 64)
		if err != nil {
			return sdk.NewErrorWithStack(err, sdk.NewErrorFrom(sdk.ErrInvalidID, "invalid node job run ID: %v", nodeJobRunIDStr))
		}

		if fileName == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "Content-Disposition header is not set")
		}

		nodeJobRun, err := workflow.LoadNodeJobRun(ctx, api.mustDB(), api.Cache, nodeJobRunID)
		if err != nil {
			return sdk.WrapError(err, "cannot load node job run")
		}

		nodeRun, err := workflow.LoadNodeRunByID(api.mustDB(), nodeJobRun.WorkflowNodeRunID, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
		if err != nil {
			return sdk.WrapError(err, "cannot load node run")
		}

		staticFile := sdk.StaticFiles{
			Name:         name,
			EntryPoint:   entrypoint,
			StaticKey:    staticKey,
			WorkflowID:   nodeRun.WorkflowID,
			NodeRunID:    nodeJobRun.WorkflowNodeRunID,
			NodeJobRunID: nodeJobRunID,
		}

		storageDriver, err := objectstore.GetDriver(ctx, api.mustDB(), api.SharedStorage, vars["permProjectKey"], vars["integrationName"])
		if err != nil {
			return err
		}

		if staticFile.StaticKey != "" {
			if err := storageDriver.Delete(ctx, &staticFile); err != nil {
				return sdk.WrapError(err, "cannot delete existing static files")
			}
		}

		id := storageDriver.GetProjectIntegration().ID
		if id > 0 {
			staticFile.ProjectIntegrationID = &id
		}

		files := m.File[fileName]
		if len(files) == 1 {
			file, err := files[0].Open()
			if err != nil {
				return sdk.WrapError(err, "cannot open file")
			}
			defer file.Close()

			publicURL, err := storageDriver.ServeStaticFiles(&staticFile, staticFile.EntryPoint, file)
			if err != nil {
				return sdk.WrapError(err, "cannot serve static files in store")
			}
			staticFile.PublicURL = publicURL
		}

		if err := workflow.InsertStaticFiles(api.mustDB(), &staticFile); err != nil {
			_ = storageDriver.Delete(ctx, &staticFile)
			return sdk.WrapError(err, "cannot insert static files in database")
		}
		return service.WriteJSON(w, staticFile, http.StatusOK)
	}
}

func (api *API) postWorkflowJobArtifactCallbackHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if _, isWorker := api.isWorker(ctx); !isWorker {
			return sdk.WithStack(sdk.ErrForbidden)
		}

		art := sdk.WorkflowNodeRunArtifact{}
		if err := service.UnmarshalBody(r, &art); err != nil {
			return err
		}

		cacheKey := cache.Key("workflows:artifacts", art.GetPath(), art.GetName())
		cachedArt := sdk.WorkflowNodeRunArtifact{}
		find, err := api.Cache.Get(cacheKey, &cachedArt)
		if err != nil {
			log.Error(ctx, "cannot get from cache %s: %v", cacheKey, err)
		}
		if !find {
			return sdk.WrapError(sdk.ErrNotFound, "Unable to find artifact, key:%s", cacheKey)
		}

		if !art.Equal(cachedArt) {
			return sdk.WrapError(sdk.ErrForbidden, "Submitted artifact doesn't match, key:%s art:%v cachedArt:%v", cacheKey, art, cachedArt)
		}

		nodeRun, err := workflow.LoadNodeRunByID(api.mustDB(), art.WorkflowNodeRunID, workflow.LoadRunOptions{WithArtifacts: true, DisableDetailledNodeRun: true})
		if err != nil {
			return sdk.WrapError(err, "cannot load node run")
		}

		nodeRun.Artifacts = append(nodeRun.Artifacts, art)
		if err := workflow.InsertArtifact(api.mustDB(), &art); err != nil {
			// TODO: call to cdn to delete artifact
			return sdk.WrapError(err, "Cannot update workflow node run")
		}

		return nil
	}
}

func (api *API) postWorkflowJobArtifactHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if _, isWorker := api.isWorker(ctx); !isWorker {
			return sdk.WithStack(sdk.ErrForbidden)
		}
		vars := mux.Vars(r)
		ref := vars["ref"]

		hash, errG := sdk.GenerateHash()
		if errG != nil {
			return sdk.WrapError(errG, "Could not generate hash")
		}

		art := sdk.WorkflowNodeRunArtifact{}
		if err := service.UnmarshalBody(r, &art); err != nil {
			return err
		}

		nodeJobRun, err := workflow.LoadNodeJobRun(ctx, api.mustDB(), api.Cache, art.WorkflowNodeJobRunID)
		if err != nil {
			return sdk.WrapError(err, "cannot load node job run with art.WorkflowNodeJobRunID: %d", art.WorkflowNodeJobRunID)
		}

		nodeRun, err := workflow.LoadNodeRunByID(api.mustDB(), nodeJobRun.WorkflowNodeRunID, workflow.LoadRunOptions{WithArtifacts: true, DisableDetailledNodeRun: true})
		if err != nil {
			return sdk.WrapError(err, "cannot load node run")
		}

		tag, err := base64.RawURLEncoding.DecodeString(ref)
		if err != nil {
			return sdk.WrapError(err, "cannot decode ref")
		}

		art.WorkflowID = nodeRun.WorkflowRunID
		art.WorkflowNodeRunID = nodeRun.ID
		art.DownloadHash = hash
		art.Tag = string(tag)
		art.Ref = ref

		srvs, err := services.LoadAllByType(ctx, api.mustDB(), services.TypeCDN)
		if err != nil {
			return sdk.WrapError(err, "cannot load services of type CDN")
		}
		cdnService := srvs[0]
		cdnReq := sdk.CDNRequest{
			Type:            sdk.CDNArtifactType,
			IntegrationName: vars["integrationName"],
			ProjectKey:      vars["permProjectKey"],
			Artifact:        &art,
		}

		cdnReqToken, err := authentication.SignJWS(cdnReq, 0)
		if err != nil {
			return sdk.WrapError(err, "cannot sign jws for cdn request")
		}
		art.TempURL = fmt.Sprintf("%s/upload/%s", cdnService.HTTPURL, cdnReqToken)

		cacheKey := cache.Key("workflows:artifacts", art.GetPath(), art.GetName())
		//Put this in cache for 1 hour
		if err := api.Cache.SetWithTTL(cacheKey, art, 60*60); err != nil {
			log.Error(ctx, "cannot SetWithTTL: %s: %v", cacheKey, err)
		}

		return service.WriteJSON(w, art, http.StatusOK)
	}
}
