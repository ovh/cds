package cdn

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/ovh/cds/sdk"
)

func (s *Service) storeArtifact(body io.ReadCloser, cdnRequest sdk.CDNRequest) (*sdk.WorkflowNodeRunArtifact, error) {
	art := cdnRequest.Artifact
	if _, err := art.IsValid(); err != nil {
		return nil, sdk.WrapError(err, "artifact is not valid")
	}
	storageDriver, err := s.getDriver(cdnRequest.ProjectKey, cdnRequest.IntegrationName)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get driver")
	}

	id := storageDriver.GetProjectIntegration().ID
	if id > 0 {
		art.ProjectIntegrationID = &id
	}

	var buf bytes.Buffer
	tee := io.TeeReader(body, &buf)
	objectPath, err := storageDriver.Store(art, ioutil.NopCloser(tee))
	if err != nil {
		return nil, sdk.WrapError(err, "Cannot store artifact")
	}
	art.ObjectPath = objectPath

	go s.mirroring(art, body, &buf)

	return art, nil
}

func (s *Service) downloadArtifact(req *http.Request, cdnRequest sdk.CDNRequest) (io.ReadCloser, error) {
	storageDriver, err := s.getDriver(cdnRequest.ProjectKey, cdnRequest.IntegrationName)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get driver")
	}

	return storageDriver.Fetch(cdnRequest.Artifact)
}
