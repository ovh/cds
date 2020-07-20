package api

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/test/assets"
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
