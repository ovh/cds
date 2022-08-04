package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func Test_getAdminDatabaseSignatureResume(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, jwt := assets.InsertAdminUser(t, db)

	uri := api.Router.GetRoute("GET", api.getAdminDatabaseSignatureResume, nil)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	t.Logf("%s", w.Body.String())
}

func Test_getAdminDatabaseSignatureTuplesByPrimaryKey(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, jwt := assets.InsertAdminUser(t, db)

	uri := api.Router.GetRoute("GET", api.getAdminDatabaseSignatureResume, nil)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var resume = sdk.CanonicalFormUsageResume{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resume))

	for entity, data := range resume {

		for i := range data {

			vars := map[string]string{
				"entity": entity,
				"signer": data[i].Signer,
			}

			uri := api.Router.GetRoute("GET", api.getAdminDatabaseSignatureTuplesBySigner, vars)
			req := assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil)

			// Do the request
			w := httptest.NewRecorder()
			api.Router.Mux.ServeHTTP(w, req)
			assert.Equal(t, 200, w.Code)

			var pks []string
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &pks))

			assert.Len(t, pks, int(data[i].Number))
		}
	}
}

func Test_postAdminDatabaseSignatureRollEntityByPrimaryKey(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, jwt := assets.InsertAdminUser(t, db)

	uri := api.Router.GetRoute("GET", api.getAdminDatabaseSignatureResume, nil)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var resume = sdk.CanonicalFormUsageResume{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resume))

	for entity, data := range resume {

		for i := range data {

			vars := map[string]string{
				"entity": entity,
				"signer": data[i].Signer,
			}

			uri := api.Router.GetRoute("GET", api.getAdminDatabaseSignatureTuplesBySigner, vars)
			req := assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil)

			// Do the request
			w := httptest.NewRecorder()
			api.Router.Mux.ServeHTTP(w, req)
			assert.Equal(t, 200, w.Code)

			var pks []string
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &pks))

			for _, pk := range pks {
				vars := map[string]string{
					"entity": entity,
					"pk":     pk,
				}

				uri := api.Router.GetRoute("POST", api.postAdminDatabaseSignatureRollEntityByPrimaryKey, vars)
				req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, nil)

				// Do the request
				w := httptest.NewRecorder()
				api.Router.Mux.ServeHTTP(w, req)
				assert.Equal(t, 204, w.Code)
			}
		}
	}
}

func Test_getAdminDatabaseEncryptedEntities(t *testing.T) {
	gorpmapping.Register(gorpmapping.New(gorpmapper.TestEncryptedData{}, "test_encrypted_data", true, "id"))

	api, db, _ := newTestAPI(t)

	_, jwt := assets.InsertAdminUser(t, db)

	uri := api.Router.GetRoute("GET", api.getAdminDatabaseEncryptedEntities, nil)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	t.Logf("%s", w.Body.String())
}

func Test_getAdminDatabaseEncryptedTuplesByEntity(t *testing.T) {
	gorpmapping.Register(gorpmapping.New(gorpmapper.TestEncryptedData{}, "test_encrypted_data", true, "id"))

	api, db, _ := newTestAPI(t)

	_, jwt := assets.InsertAdminUser(t, db)

	uri := api.Router.GetRoute("GET", api.getAdminDatabaseEncryptedTuplesByEntity, map[string]string{"entity": "gorpmapper.TestEncryptedData"})
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	t.Logf("%s", w.Body.String())
}

func Test_postAdminDatabaseRollEncryptedEntityByPrimaryKey(t *testing.T) {
	gorpmapping.Register(gorpmapping.New(gorpmapper.TestEncryptedData{}, "test_encrypted_data", true, "id"))

	api, db, _ := newTestAPI(t)

	_, jwt := assets.InsertAdminUser(t, db)

	uri := api.Router.GetRoute("GET", api.getAdminDatabaseEncryptedTuplesByEntity, map[string]string{"entity": "gorpmapper.TestEncryptedData"})
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil)

	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	var res []string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

	for _, s := range res {
		uri := api.Router.GetRoute("POST", api.postAdminDatabaseRollEncryptedEntityByPrimaryKey, map[string]string{"entity": "gorpmapper.TestEncryptedData", "pk": s})
		req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, nil)

		// Do the request
		w := httptest.NewRecorder()
		api.Router.Mux.ServeHTTP(w, req)
		assert.Equal(t, 204, w.Code)
	}
}

func Test_postAdminDatabaseRollEncryptedEntityByPrimaryKeyForApplication(t *testing.T) {
	api, db, _ := newTestAPI(t)
	_, jwt := assets.InsertAdminUser(t, db)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(6), sdk.RandomString(6))
	app := &sdk.Application{
		Name:               "my-amm",
		RepositoryFullname: "ovh/cds",
		RepositoryStrategy: sdk.RepositoryStrategy{
			ConnectionType: "https",
			User:           "foo",
			Password:       "bar",
		},
	}
	require.NoError(t, application.Insert(db, *proj, app))

	var err error
	app, err = application.LoadByIDWithClearVCSStrategyPassword(context.Background(), db, app.ID)
	require.NoError(t, err)

	uri := api.Router.GetRoute("POST", api.postAdminDatabaseRollEncryptedEntityByPrimaryKey, map[string]string{"entity": "application.dbApplication", "pk": fmt.Sprintf("%d", app.ID)})
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, nil)
	// Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 204, w.Code)

	app2, err := application.LoadByIDWithClearVCSStrategyPassword(context.Background(), db, app.ID)
	require.NoError(t, err)
	require.Equal(t, app.RepositoryStrategy, app2.RepositoryStrategy)
}

func Test_postAdminDatabaseRollEncryptedEntityByPrimaryKeyForProjectIntegration(t *testing.T) {
	api, db, router := newTestAPI(t)
	u, jwt := assets.InsertAdminUser(t, db)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(6), sdk.RandomString(6))

	integrationModel, err := integration.LoadModelByName(context.TODO(), db, sdk.KafkaIntegration.Name)
	if err != nil {
		assert.NoError(t, integration.CreateBuiltinModels(context.TODO(), api.mustDB()))
		models, _ := integration.LoadModels(db)
		assert.True(t, len(models) > 0)
	}

	integrationModel, err = integration.LoadModelByName(context.TODO(), db, sdk.AWSIntegration.Name)
	test.NoError(t, err)

	pp := sdk.ProjectIntegration{
		Name:               "test",
		Config:             sdk.AWSIntegration.DefaultConfig.Clone(),
		IntegrationModelID: integrationModel.ID,
	}

	for k, v := range pp.Config {
		v.Value = sdk.RandomString(5)
		pp.Config[k] = v
	}

	t.Logf("%+v", pp.Config)

	// ADD integration
	vars := map[string]string{}
	vars[permProjectKey] = proj.Key
	uri := router.GetRoute("POST", api.postProjectIntegrationHandler, vars)
	req := assets.NewAuthentifiedRequest(t, u, jwt, "POST", uri, pp)
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	require.NoError(t, err)

	integ, err := integration.LoadIntegrationsByProjectIDWithClearPassword(context.TODO(), db, proj.ID)
	t.Logf("%+v", integ[0].Config)
	require.NoError(t, err)

	uri = api.Router.GetRoute("POST", api.postAdminDatabaseRollEncryptedEntityByPrimaryKey, map[string]string{"entity": "integration.dbProjectIntegration", "pk": fmt.Sprintf("%d", integ[0].ID)})
	req = assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, nil)
	// Do the request
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 204, w.Code)

	integ2, err := integration.LoadIntegrationsByProjectIDWithClearPassword(context.TODO(), db, proj.ID)
	require.NoError(t, err)

	t.Logf("%+v", integ2[0].Config)

	require.Len(t, integ2[0].Config, len(pp.Config))
	for k, v := range pp.Config {
		assert.Equal(t, integ2[0].Config[k], v)
	}
}

func Test_postWorkflowMaxRunHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)
	workflow.SetMaxRuns(15)

	_, jwt := assets.InsertAdminUser(t, db)

	p := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	w := assets.InsertTestWorkflow(t, db, api.Cache, p, sdk.RandomString(10))

	require.Equal(t, int64(15), w.MaxRuns)

	uri := api.Router.GetRoute("POST", api.postWorkflowMaxRunHandler, map[string]string{"key": p.Key, "permWorkflowName": w.Name})
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, sdk.UpdateMaxRunRequest{MaxRuns: 5})

	// Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 204, rec.Code)

	wfDb, err := workflow.Load(context.TODO(), db, api.Cache, *p, w.Name, workflow.LoadOptions{})
	require.NoError(t, err)
	require.Equal(t, int64(5), wfDb.MaxRuns)

	wfDb.MaxRuns = 20
	require.NoError(t, workflow.Update(context.TODO(), db, api.Cache, *p, wfDb, workflow.UpdateOptions{}))

	// Max runs must not be updated
	wfDb2, err := workflow.Load(context.TODO(), db, api.Cache, *p, w.Name, workflow.LoadOptions{})
	require.NoError(t, err)
	require.Equal(t, int64(5), wfDb2.MaxRuns)
}
