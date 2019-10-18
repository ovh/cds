package cdn

import (
	"io"
	"net/http"

	"github.com/ovh/cds/sdk"
)

func (s *Service) storeArtifact(body io.ReadCloser, cdnRequest sdk.CDNRequest) (*sdk.WorkflowNodeRunArtifact, error) {
	// TODO: add isValid on Artifact
	art := cdnRequest.Artifact
	storageDriver, err := s.getDriver(cdnRequest.ProjectKey, cdnRequest.IntegrationName)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get driver")
	}

	id := storageDriver.GetProjectIntegration().ID
	if id > 0 {
		art.ProjectIntegrationID = &id
	}

	objectPath, err := storageDriver.Store(art, body)
	if err != nil {
		_ = body.Close()
		return nil, sdk.WrapError(err, "Cannot store artifact")
	}
	defer body.Close()

	art.ObjectPath = objectPath

	return art, nil
}

func (s *Service) downloadArtifact(req *http.Request, cdnRequest sdk.CDNRequest) (io.ReadCloser, error) {
	storageDriver, err := s.getDriver(cdnRequest.ProjectKey, cdnRequest.IntegrationName)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get driver")
	}

	return storageDriver.Fetch(cdnRequest.Artifact)
}
