package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
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

func Test_postWorkflowMaxRunHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)
	api.Config.Workflow.MaxRuns = 10
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	go workflow.Initialize(ctx, api.DBConnectionFactory.GetDBMap(gorpmapping.Mapper), api.Cache, "", "", "", 15, api.Config.Workflow.MaxRuns)
	_, jwt := assets.InsertAdminUser(t, db)

	p := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	w := assets.InsertTestWorkflow(t, db, api.Cache, p, sdk.RandomString(10))

	require.Equal(t, w.MaxRuns, api.Config.Workflow.MaxRuns)

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
