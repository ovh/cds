package repositories

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/engine/api/test"
	"github.com/stretchr/testify/assert"
)

func Test_postOperationHandler(t *testing.T) {
	//Bootstrap the service
	s, err := newTestService(t)
	test.NoError(t, err)

	op := new(sdk.Operation)
	op.URL = "https://github.com/ovh/cds.git"
	op.RepositoryStrategy = sdk.RepositoryStrategy{
		ConnectionType: "https",
		DefaultBranch:  "master",
	}
	op.Setup.Checkout = sdk.OperationCheckout{
		Branch: "master",
	}

	//Prepare request
	vars := map[string]string{}
	uri := s.Router.GetRoute("POST", s.postOperationHandler, vars)
	test.NotEmpty(t, uri)
	req := newRequest(t, s, "POST", uri, op)

	//Do the request
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 202, rec.Code)
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), op))

	//Prepare request
	vars = map[string]string{
		"uuid": op.UUID,
	}
	uri = s.Router.GetRoute("GET", s.getOperationsHandler, vars)
	test.NotEmpty(t, uri)
	req = newRequest(t, s, "GET", uri, nil)

	//Do the request
	rec = httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 200, rec.Code)
	t.Logf(rec.Body.String())
}

func Test_postOperationMultiPartHandler(t *testing.T) {
	//Bootstrap the service
	s, err := newTestService(t)
	test.NoError(t, err)

	op := new(sdk.Operation)
	op.Setup.Push = sdk.OperationPush{
		FromBranch: "temp",
		Message:    "initial as code",
	}

	//Prepare request
	vars := map[string]string{}
	uri := s.Router.GetRoute("POST", s.postOperationHandler, vars)
	test.NotEmpty(t, uri)

	buf := new(bytes.Buffer)
	getWorkflowTarFile(t, buf)
	req := newMultiPartTarRequest(t, s, "POST", uri, op, buf)

	//Do the request
	rec := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 202, rec.Code)
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), op))

	//Prepare request
	vars = map[string]string{
		"uuid": op.UUID,
	}
	uri = s.Router.GetRoute("GET", s.getOperationsHandler, vars)
	test.NotEmpty(t, uri)
	req = newRequest(t, s, "GET", uri, nil)

	//Do the request
	rec = httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(rec, req)

	//Asserts
	assert.Equal(t, 200, rec.Code)
	t.Logf(rec.Body.String())
}

func getWorkflowTarFile(t *testing.T, buf *bytes.Buffer) {
	tw := tar.NewWriter(buf)
	defer func() {
		if err := tw.Close(); err != nil {
			t.Errorf("unable to close tar file")
			t.Fail()
		}
	}()
	var files = []struct {
		Name, Body string
	}{
		{"workflow.yml", `name: myworkflow
  version: v1.0
  workflow:
    root:`},
	}
	for _, file := range files {
		hdr := &tar.Header{
			Name: file.Name,
			Mode: 0600,
			Size: int64(len(file.Body)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Errorf("unable to write header")
			t.Fail()
		}
		if _, err := tw.Write([]byte(file.Body)); err != nil {
			t.Errorf("unable to write body")
			t.Fail()
		}
	}
	if err := tw.Close(); err != nil {
		t.Errorf("unable to close tar writer")
		t.Fail()
	}
}
