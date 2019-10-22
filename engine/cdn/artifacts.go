package cdn

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
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

	var buf bytes.Buffer
	// var btes []byte
	// if _, err := io.CopyBuffer(&buf, body, btes); err != nil {
	// 	return nil, sdk.WrapError(err, "cannot copy body")
	// }
	tee := io.TeeReader(body, &buf)
	// content, err := ioutil.ReadAll(&buf)
	// fmt.Printf("content %+v --- err %+v\n", content, err)
	objectPath, err := storageDriver.Store(art, ioutil.NopCloser(&buf))
	if err != nil {
		return nil, sdk.WrapError(err, "Cannot store artifact")
	}
	art.ObjectPath = objectPath
	fmt.Println("objectPath --- ", objectPath)

	// TODO: test mirroring
	go s.mirroring(art, body, tee)

	return art, nil
}

func (s *Service) downloadArtifact(req *http.Request, cdnRequest sdk.CDNRequest) (io.ReadCloser, error) {
	storageDriver, err := s.getDriver(cdnRequest.ProjectKey, cdnRequest.IntegrationName)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get driver")
	}

	return storageDriver.Fetch(cdnRequest.Artifact)
}
