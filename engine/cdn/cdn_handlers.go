package cdn

import (
	"context"
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ovh/cds/engine/cdn/objectstore"

	"github.com/gorilla/mux"
	cdnauth "github.com/ovh/cds/engine/api/authentication/cdn"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Status returns sdk.MonitoringStatus, implements interface service.Service
func (s *Service) Status() sdk.MonitoringStatus {
	m := s.CommonMonitoring()

	status := sdk.MonitoringStatusOK

	m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "CDN", Value: status, Status: status})

	return m
}

func (s *Service) statusHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var status = http.StatusOK
		return service.WriteJSON(w, s.Status(), status)
	}
}

func (s *Service) getDownloadHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var status = http.StatusOK
		return service.WriteJSON(w, nil, status)
	}
}

func (s *Service) postUploadHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		token := vars["token"]

		cdnToken, err := cdnauth.VerifyToken(s.ParsedAPIPublicKey, token)
		if err != nil {
			return sdk.WrapError(sdk.ErrForbidden, "cannot verify token")
		}

		// decrypt JWT TOKEN
		// Get payload to check which kind of data it is

		// inside payload --> Type, and config map[string]string
		// config --> nodeRunID, nodeJobRunID, step for logs, tag, name, projectKey

		switch cdnToken.CDNRequest.Type {
		case sdk.CDNArtifactType:
			artifact, err := s.storeArtifact(r, cdnToken.CDNRequest)
			if err != nil {
				return sdk.WrapError(err, "cannot store artifact")
			}
			return service.WriteJSON(w, *artifact, http.StatusOK)
		}

		var status = http.StatusOK

		// response could be WorkflowNodeRunArtifact, Logs, Static Files, Cache, Icones

		return service.WriteJSON(w, nil, status)
	}
}

func (s *Service) storeArtifact(req *http.Request, cdnRequest sdk.CDNRequest) (*sdk.WorkflowNodeRunArtifact, error) {
	_, params, errM := mime.ParseMediaType(req.Header.Get("Content-Disposition"))
	if errM != nil {
		return nil, sdk.WrapError(errM, "Cannot read Content Disposition header")
	}

	//parse the multipart form in the request
	if err := req.ParseMultipartForm(100000); err != nil {
		return nil, sdk.WrapError(err, "Error parsing multipart form")
	}
	//get a ref to the parsed multipart form
	form := req.MultipartForm
	fileName := params["filename"]

	var sizeStr, permStr, md5sum, sha512sum, nodeJobRunIDStr string
	if len(form.Value["size"]) > 0 {
		sizeStr = form.Value["size"][0]
	}
	if len(form.Value["perm"]) > 0 {
		permStr = form.Value["perm"][0]
	}
	if len(form.Value["md5sum"]) > 0 {
		md5sum = form.Value["md5sum"][0]
	}
	if len(form.Value["sha512sum"]) > 0 {
		sha512sum = form.Value["sha512sum"][0]
	}
	if len(form.Value["nodeJobRunID"]) > 0 {
		nodeJobRunIDStr = form.Value["nodeJobRunID"][0]
	}
	nodeJobRunID, errI := strconv.ParseInt(nodeJobRunIDStr, 10, 64)
	if errI != nil {
		return nil, sdk.WrapError(sdk.ErrInvalidID, "Invalid node job run ID")
	}

	if fileName == "" {
		return nil, sdk.WrapError(sdk.ErrWrongRequest, "%s header is not set", "Content-Disposition")
	}

	hash, errG := sdk.GenerateHash()
	if errG != nil {
		return nil, sdk.WrapError(errG, "Could not generate hash")
	}

	var size int64
	var perm uint64

	if sizeStr != "" {
		size, _ = strconv.ParseInt(sizeStr, 10, 64)
	}

	if permStr != "" {
		perm, _ = strconv.ParseUint(permStr, 10, 32)
	}

	var nodeRunID int64
	if nodeRunIDStr, ok := cdnRequest.Config["nodeRunID"]; ok {
		var err error
		nodeRunID, err = strconv.ParseInt(nodeRunIDStr, 10, 64)
		if err != nil {
			return nil, sdk.WrapError(err, "cannot parse nodeRunID : %v", nodeRunIDStr)
		}
	} else {
		return nil, fmt.Errorf("missing nodeRunID in the config")
	}

	var workflowRunID int64
	if workflowRunIDStr, ok := cdnRequest.Config["workflowRunID"]; ok {
		var err error
		workflowRunID, err = strconv.ParseInt(workflowRunIDStr, 10, 64)
		if err != nil {
			return nil, sdk.WrapError(err, "cannot parse workflowRunID : %v", workflowRunIDStr)
		}
	} else {
		return nil, fmt.Errorf("missing workflowRunID in the config")
	}

	art := sdk.WorkflowNodeRunArtifact{
		Name:                 fileName,
		Tag:                  cdnRequest.Config["tag"],
		DownloadHash:         hash,
		Size:                 size,
		Perm:                 uint32(perm),
		MD5sum:               md5sum,
		SHA512sum:            sha512sum,
		WorkflowNodeRunID:    nodeRunID,
		WorkflowNodeJobRunID: nodeJobRunID,
		WorkflowID:           workflowRunID,
		Created:              time.Now(),
	}

	// tag, errT := base64.RawURLEncoding.DecodeString(ref)
	// if errT != nil {
	// 	return nil, sdk.WrapError(errT, "Cannot decode ref")
	// }

	projectIntegration, err := s.Client.ProjectIntegrationGet(cdnRequest.ProjectKey, cdnRequest.Config["integrationName"], true)
	if err != nil {
		return nil, err
	}

	var storageDriver objectstore.Driver
	if strings.HasPrefix(projectIntegration.Name, sdk.DefaultStorageIntegrationName) {
		storageDriver = s.DefaultDriver
	} else {
		var errD error
		storageDriver, errD = objectstore.InitDriver(projectIntegration)
		if errD != nil {
			return nil, sdk.WrapError(errD, "cannot init storage driver")
		}
	}

	id := storageDriver.GetProjectIntegration().ID
	if id > 0 {
		art.ProjectIntegrationID = &id
	}

	files := form.File[fileName]
	if len(files) == 1 {
		file, err := files[0].Open()
		if err != nil {
			_ = file.Close()
			return nil, sdk.WrapError(err, "cannot open file")
		}

		objectPath, err := storageDriver.Store(&art, file)
		if err != nil {
			_ = file.Close()
			return nil, sdk.WrapError(err, "Cannot store artifact")
		}
		// TODO: dev the behavior with mirrors and retry with a goroutine
		log.Debug("objectpath=%s\n", objectPath)
		art.ObjectPath = objectPath
		_ = file.Close()
	}

	return &art, nil
}
