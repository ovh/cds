package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/platform"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func TestAddUpdateAndDeleteProjectPlatform(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), nil)
	u, pass := assets.InsertAdminUser(api.mustDB(context.Background()))

	models, _ := platform.LoadModels(db)
	if len(models) == 0 {
		assert.NoError(t, platform.CreateModels(db))
		models, _ = platform.LoadModels(db)
	}

	pp := sdk.ProjectPlatform{
		Name:            "kafkaTest",
		Config:          sdk.KafkaPlatform.DefaultConfig,
		PlatformModelID: models[0].ID,
	}

	// ADD project platform
	jsonBody, _ := json.Marshal(pp)
	body := bytes.NewBuffer(jsonBody)

	vars := map[string]string{}
	vars["permProjectKey"] = proj.Key
	uri := router.GetRoute("POST", api.postProjectPlatformHandler, vars)
	req, err := http.NewRequest("POST", uri, body)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// UPDATE project platform
	pp.Name = "kafkaTest2"
	pp.ProjectID = proj.ID
	jsonBody, _ = json.Marshal(pp)
	body = bytes.NewBuffer(jsonBody)

	vars = map[string]string{}
	vars["permProjectKey"] = proj.Key
	vars["platformName"] = "kafkaTest"
	uri = router.GetRoute("PUT", api.putProjectPlatformHandler, vars)
	req, err = http.NewRequest("PUT", uri, body)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// GET project platform
	vars = map[string]string{}
	vars["permProjectKey"] = proj.Key
	vars["platformName"] = pp.Name
	uri = router.GetRoute("GET", api.getProjectPlatformHandler, vars)
	req, err = http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// DELETE project platform
	vars = map[string]string{}
	vars["permProjectKey"] = proj.Key
	vars["platformName"] = pp.Name
	uri = router.GetRoute("DELETE", api.deleteProjectPlatformHandler, vars)
	req, err = http.NewRequest("DELETE", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 204, w.Code)
}
