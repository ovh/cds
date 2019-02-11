package api

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func TestAddUpdateAndDeleteProjectIntegration(t *testing.T) {
	api, db, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), nil)
	u, pass := assets.InsertAdminUser(api.mustDB())

	models, _ := integration.LoadModels(db)
	if len(models) == 0 {
		assert.NoError(t, integration.CreateBuiltinModels(db))
		models, _ = integration.LoadModels(db)
		assert.True(t, len(models) > 0)
	}

	integrationModel, err := integration.LoadModelByName(db, sdk.KafkaIntegration.Name, false)
	test.NoError(t, err)

	pp := sdk.ProjectIntegration{
		Name:               "kafkaTest",
		Config:             sdk.KafkaIntegration.DefaultConfig,
		IntegrationModelID: integrationModel.ID,
	}

	// ADD integration
	vars := map[string]string{}
	vars[permProjectKey] = proj.Key
	uri := router.GetRoute("POST", api.postProjectIntegrationHandler, vars)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, pp)
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// UPDATE integration
	pp.Name = "kafkaTest2"
	pp.ProjectID = proj.ID

	vars = map[string]string{}
	vars[permProjectKey] = proj.Key
	vars["integrationName"] = "kafkaTest"
	uri = router.GetRoute("PUT", api.putProjectIntegrationHandler, vars)
	req = assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, pp)

	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// GET integration
	vars = map[string]string{}
	vars[permProjectKey] = proj.Key
	vars["integrationName"] = pp.Name
	uri = router.GetRoute("GET", api.getProjectIntegrationHandler, vars)

	req = assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// DELETE integration
	vars = map[string]string{}
	vars[permProjectKey] = proj.Key
	vars["integrationName"] = pp.Name
	uri = router.GetRoute("DELETE", api.deleteProjectIntegrationHandler, vars)
	req = assets.NewAuthentifiedRequest(t, u, pass, "DELETE", uri, nil)

	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 204, w.Code)
}
