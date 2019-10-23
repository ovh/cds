package cdn

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/ovh/cds/sdk"
)

func (s *Service) storeArtifact(ctx context.Context, body io.ReadCloser, cdnRequest sdk.CDNRequest) (*sdk.WorkflowNodeRunArtifact, error) {
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
	objectPath, err := storageDriver.Store(ctx, art, ioutil.NopCloser(tee))
	if err != nil {
		return nil, sdk.WrapError(err, "Cannot store artifact")
	}
	art.ObjectPath = objectPath

	sdk.GoRoutine(context.Background(), "StoreArtifactMirroring", func(_ context.Context) {
		defer body.Close()
		s.mirroring(art, &buf)
	})

	return art, nil
}

func (s *Service) downloadArtifact(ctx context.Context, req *http.Request, cdnRequest sdk.CDNRequest) (io.ReadCloser, error) {
	storageDriver, err := s.getDriver(cdnRequest.ProjectKey, cdnRequest.IntegrationName)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get driver")
	}

	content, err := storageDriver.Fetch(ctx, cdnRequest.Artifact)
	if err == nil {
		return content, nil
	}
	// Cannot have mirrors on integrations
	if cdnRequest.IntegrationName != sdk.DefaultStorageIntegrationName {
		return nil, sdk.WrapError(err, "cannot download artifact")
	}

	return s.downloadFromMirrors(ctx, cdnRequest.Artifact)
}
