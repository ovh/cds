package cdn

import (
	"context"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/authentication"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func Test_postUploadArtifactHandler(t *testing.T) {
	//Bootstrap the service
	s, err := newTestService(t)
	test.NoError(t, err)

	art := sdk.WorkflowNodeRunArtifact{
		Name:                 sdk.RandomString(10),
		Tag:                  "test",
		WorkflowNodeRunID:    1,
		WorkflowID:           1,
		WorkflowNodeJobRunID: 1,
		Ref:                  "test",
	}

	cdnReq := sdk.CDNRequest{
		Type:            sdk.CDNArtifactType,
		IntegrationName: sdk.DefaultStorageIntegrationName,
		ProjectKey:      "test",
		Artifact:        &art,
	}

	cdnReqToken, err := authentication.SignJWS(cdnReq, 0)
	test.NoError(t, err, "cannot sign jws")

	//Prepare request
	vars := map[string]string{
		"token": cdnReqToken,
	}
	uri := s.Router.GetRoute("POST", s.postUploadHandler, vars)
	test.NotEmpty(t, uri)
	req := newRequest(t, s, "POST", uri, []byte("hereisatest"))

	//Do the request
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	content, err := s.DefaultDriver.Fetch(context.Background(), &art)
	test.NoError(t, err, "cannot fetch artifact")
	contentStr, err := ioutil.ReadAll(content)
	test.NoError(t, err, "cannot read artifact content")

	test.Equal(t, "hereisatest", string(contentStr))
	//Asserts
	assert.Equal(t, 200, rec.Code)
}
