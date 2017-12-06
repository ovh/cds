package api

import (
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func Test_getPipelineExportHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)
	pipName := sdk.RandomString(10)
	pip := &sdk.Pipeline{
		ProjectID: proj.ID,
		Name:      pipName,
		Type:      sdk.BuildPipeline,
	}

	if err := pipeline.InsertPipeline(db, api.Cache, proj, pip, u); err != nil {
		t.Fatal(err)
	}

	//Prepare request
	vars := map[string]string{
		"key":             proj.Key,
		"permPipelineKey": pip.Name,
	}
	uri := api.Router.GetRoute("GET", api.getPipelineExportHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())

}
