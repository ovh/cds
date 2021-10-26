package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPI_post_getProjectIntegrationWorkerHookHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	u, pass := assets.InsertAdminUser(t, db)

	integrationModel, err := integration.LoadModelByName(context.TODO(), db, sdk.KafkaIntegration.Name)
	if err != nil {
		assert.NoError(t, integration.CreateBuiltinModels(context.TODO(), api.mustDB()))
		models, _ := integration.LoadModels(db)
		assert.True(t, len(models) > 0)
	}

	integrationModel, err = integration.LoadModelByName(context.TODO(), db, sdk.KafkaIntegration.Name)
	test.NoError(t, err)

	pp := sdk.ProjectIntegration{
		Name:               "kafkaTest",
		Config:             sdk.KafkaIntegration.DefaultConfig.Clone(),
		IntegrationModelID: integrationModel.ID,
		ProjectID:          proj.ID,
	}

	require.NoError(t, integration.InsertIntegration(db, &pp))

	vars := map[string]string{
		permProjectKey:    proj.Key,
		"integrationName": pp.Name,
	}

	wh := sdk.WorkerHookProjectIntegrationModel{
		Configuration: sdk.WorkerHookSetupTeardownConfig{
			ByCapabilities: map[string]sdk.WorkerHookSetupTeardownScripts{
				"echo": {
					Priority: 1,
					Label:    "test",
					Setup:    "echo Hello, world",
				},
			},
		},
	}

	uri := router.GetRoute("POST", api.postProjectIntegrationWorkerHookHandler, vars)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, wh)
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	btes := w.Body.Bytes()
	var wh2 sdk.WorkerHookProjectIntegrationModel
	require.NoError(t, json.Unmarshal(btes, &wh2))

	uri = router.GetRoute("GET", api.getProjectIntegrationWorkerHookHandler, vars)
	req = assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	btes = w.Body.Bytes()
	var wh3 sdk.WorkerHookProjectIntegrationModel
	require.NoError(t, json.Unmarshal(btes, &wh3))

	wh = wh3
	wh.Disable = true

	uri = router.GetRoute("POST", api.postProjectIntegrationWorkerHookHandler, vars)
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, wh)
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	btes = w.Body.Bytes()
	var wh4 sdk.WorkerHookProjectIntegrationModel
	require.NoError(t, json.Unmarshal(btes, &wh4))

	t.Logf(">>wh4= %+v", wh4)

	uri = router.GetRoute("GET", api.getProjectIntegrationWorkerHookHandler, vars)
	req = assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	btes = w.Body.Bytes()
	var wh5 sdk.WorkerHookProjectIntegrationModel
	require.NoError(t, json.Unmarshal(btes, &wh5))
	require.True(t, wh5.Disable)
}
