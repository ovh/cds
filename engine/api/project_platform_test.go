package api

import (
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
	api, db, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), nil)
	u, pass := assets.InsertAdminUser(api.mustDB())

	models, _ := platform.LoadModels(db)
	if len(models) == 0 {
		assert.NoError(t, platform.CreateBuiltinModels(db))
		models, _ = platform.LoadModels(db)
	}

	platformModel, err := platform.LoadModelByName(db, sdk.KafkaPlatform.Name, false)
	test.NoError(t, err)

	pp := sdk.ProjectPlatform{
		Name:            "kafkaTest",
		Config:          sdk.KafkaPlatform.DefaultConfig,
		PlatformModelID: platformModel.ID,
	}

	// ADD project platform
	vars := map[string]string{}
	vars["permProjectKey"] = proj.Key
	uri := router.GetRoute("POST", api.postProjectPlatformHandler, vars)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, pp)
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// UPDATE project platform
	pp.Name = "kafkaTest2"
	pp.ProjectID = proj.ID

	vars = map[string]string{}
	vars["permProjectKey"] = proj.Key
	vars["platformName"] = "kafkaTest"
	uri = router.GetRoute("PUT", api.putProjectPlatformHandler, vars)
	req = assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, pp)

	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// GET project platform
	vars = map[string]string{}
	vars["permProjectKey"] = proj.Key
	vars["platformName"] = pp.Name
	uri = router.GetRoute("GET", api.getProjectPlatformHandler, vars)

	req = assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// DELETE project platform
	vars = map[string]string{}
	vars["permProjectKey"] = proj.Key
	vars["platformName"] = pp.Name
	uri = router.GetRoute("DELETE", api.deleteProjectPlatformHandler, vars)
	req = assets.NewAuthentifiedRequest(t, u, pass, "DELETE", uri, nil)

	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 204, w.Code)
}
