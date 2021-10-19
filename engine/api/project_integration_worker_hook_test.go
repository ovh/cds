package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPI_put_getProjectIntegrationWorkerHooksHandler(t *testing.T) {
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

	whs := []sdk.WorkerHookProjectIntegrationModel{
		{
			Configuration: sdk.WorkerHookSetupTeardownConfig{
				ByCapabilities: map[string]sdk.WorkerHookSetupTeardownScripts{
					"echo": {
						Priority: 1,
						Label:    "test",
						Setup:    "echo Hello, world",
					},
				},
			},
		},
	}

	uri := router.GetRoute("POST", api.postProjectIntegrationWorkerHooksHandler, vars)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, whs)
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	btes := w.Body.Bytes()
	var whs2 []sdk.WorkerHookProjectIntegrationModel
	require.NoError(t, json.Unmarshal(btes, &whs2))
	require.Len(t, whs2, 1)
	t.Logf(">> whs2=%+v", whs2)

	uri = router.GetRoute("GET", api.getProjectIntegrationWorkerHooksHandler, vars)
	req = assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	btes = w.Body.Bytes()
	var whs3 []sdk.WorkerHookProjectIntegrationModel
	require.NoError(t, json.Unmarshal(btes, &whs3))

	t.Logf(">> whs3=%+v", whs3)

	wh := whs3[0]
	wh.Disable = true
	vars = map[string]string{
		permProjectKey:    proj.Key,
		"integrationName": pp.Name,
		"id":              strconv.FormatInt(wh.ID, 10),
	}

	uri = router.GetRoute("PUT", api.putProjectIntegrationWorkerHookHandler, vars)
	req = assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, wh)
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	btes = w.Body.Bytes()
	var wh2 sdk.WorkerHookProjectIntegrationModel
	require.NoError(t, json.Unmarshal(btes, &wh2))

	t.Logf(">>wh2= %+v", wh2)

	uri = router.GetRoute("GET", api.getProjectIntegrationWorkerHookHandler, vars)
	req = assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	btes = w.Body.Bytes()
	var wh3 sdk.WorkerHookProjectIntegrationModel
	require.NoError(t, json.Unmarshal(btes, &wh3))
	require.True(t, wh3.Disable)
	t.Logf(">> wh2=%+v", wh2)

}
