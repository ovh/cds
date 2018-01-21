package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yesnault/gadgeto/iffy"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_attachPipelinesToApplicationHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       pkey + "_PIP",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}

	if err := pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, pip, u); err != nil {
		t.Fatal(err)
	}

	//Insert Pipeline
	pip2 := &sdk.Pipeline{
		Name:       pkey + "_PIP2",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}

	if err := pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, pip2, u); err != nil {
		t.Fatal(err)
	}

	//Insert Application
	app := &sdk.Application{
		Name: "TEST_APP",
	}

	if err := application.Insert(api.mustDB(), api.Cache, proj, app, u); err != nil {
		t.Fatal(err)
	}

	request := []string{pkey + "_PIP", pkey + "_PIP2"}

	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
	}

	uri := router.GetRoute("POST", api.attachPipelinesToApplicationHandler, vars)
	test.NotEmpty(t, uri)

	jsonBody, _ := json.Marshal(request)
	body := bytes.NewBuffer(jsonBody)
	req, err := http.NewRequest("POST", uri, body)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	appDB, err := application.LoadByName(api.mustDB(), api.Cache, proj.Key, app.Name, u, application.LoadOptions.WithPipelines)
	test.NoError(t, err)

	assert.Equal(t, len(appDB.Pipelines), 2)
}
