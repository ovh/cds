package cdn

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func Test_getAdminDatabaseSignatureResume(t *testing.T) {
	s, _ := newTestService(t)

	uri := s.Router.GetRoute("GET", s.getAdminDatabaseSignatureResume, nil)
	req := newRequest(t, "GET", uri, nil)

	// Do the request
	w := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	t.Logf("%s", w.Body.String())
}

func Test_getAdminDatabaseSignatureTuplesByPrimaryKey(t *testing.T) {
	s, _ := newTestService(t)

	uri := s.Router.GetRoute("GET", s.getAdminDatabaseSignatureResume, nil)
	req := newRequest(t, "GET", uri, nil)

	// Do the request
	w := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var resume = sdk.CanonicalFormUsageResume{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resume))

	for entity, data := range resume {

		for i := range data {

			vars := map[string]string{
				"entity": entity,
				"signer": data[i].Signer,
			}

			uri := s.Router.GetRoute("GET", s.getAdminDatabaseSignatureTuplesBySigner, vars)
			req := newRequest(t, "GET", uri, nil)

			// Do the request
			w := httptest.NewRecorder()
			s.Router.Mux.ServeHTTP(w, req)
			assert.Equal(t, 200, w.Code)

			var pks []string
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &pks))

			assert.Len(t, pks, int(data[i].Number))
		}
	}
}

func Test_postAdminDatabaseSignatureRollEntityByPrimaryKey(t *testing.T) {
	s, _ := newTestService(t)

	uri := s.Router.GetRoute("GET", s.getAdminDatabaseSignatureResume, nil)
	req := newRequest(t, "GET", uri, nil)

	// Do the request
	w := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var resume = sdk.CanonicalFormUsageResume{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resume))

	for entity, data := range resume {

		for i := range data {

			vars := map[string]string{
				"entity": entity,
				"signer": data[i].Signer,
			}

			uri := s.Router.GetRoute("GET", s.getAdminDatabaseSignatureTuplesBySigner, vars)
			req := newRequest(t, "GET", uri, nil)

			// Do the request
			w := httptest.NewRecorder()
			s.Router.Mux.ServeHTTP(w, req)
			assert.Equal(t, 200, w.Code)

			var pks []string
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &pks))

			for _, pk := range pks {
				vars := map[string]string{
					"entity": entity,
					"pk":     pk,
				}

				uri := s.Router.GetRoute("POST", s.postAdminDatabaseSignatureRollEntityByPrimaryKey, vars)
				req := newRequest(t, "POST", uri, nil)

				// Do the request
				w := httptest.NewRecorder()
				s.Router.Mux.ServeHTTP(w, req)
				assert.Equal(t, 204, w.Code)
			}
		}
	}
}

func Test_getAdminDatabaseEncryptedEntities(t *testing.T) {
	s, _ := newTestService(t)
	s.Mapper.Register(s.Mapper.NewTableMapping(gorpmapper.TestEncryptedData{}, "test_encrypted_data", true, "id"))

	uri := s.Router.GetRoute("GET", s.getAdminDatabaseEncryptedEntities, nil)
	req := newRequest(t, "GET", uri, nil)

	// Do the request
	w := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	t.Logf("%s", w.Body.String())
}

func Test_getAdminDatabaseEncryptedTuplesByEntity(t *testing.T) {
	s, _ := newTestService(t)
	s.Mapper.Register(s.Mapper.NewTableMapping(gorpmapper.TestEncryptedData{}, "test_encrypted_data", true, "id"))

	uri := s.Router.GetRoute("GET", s.getAdminDatabaseEncryptedTuplesByEntity, map[string]string{"entity": "gorpmapper.TestEncryptedData"})
	req := newRequest(t, "GET", uri, nil)

	// Do the request
	w := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	t.Logf("%s", w.Body.String())
}

func Test_postAdminDatabaseRollEncryptedEntityByPrimaryKey(t *testing.T) {

	s, _ := newTestService(t)
	s.Mapper.Register(s.Mapper.NewTableMapping(gorpmapper.TestEncryptedData{}, "test_encrypted_data", true, "id"))

	uri := s.Router.GetRoute("GET", s.getAdminDatabaseEncryptedTuplesByEntity, map[string]string{"entity": "gorpmapper.TestEncryptedData"})
	req := newRequest(t, "GET", uri, nil)

	// Do the request
	w := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	var res []string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

	for _, r := range res {
		uri := s.Router.GetRoute("POST", s.postAdminDatabaseRollEncryptedEntityByPrimaryKey, map[string]string{"entity": "gorpmapper.TestEncryptedData", "pk": r})
		req := newRequest(t, "POST", uri, nil)

		// Do the request
		w := httptest.NewRecorder()
		s.Router.Mux.ServeHTTP(w, req)
		assert.Equal(t, 204, w.Code)
	}
}
