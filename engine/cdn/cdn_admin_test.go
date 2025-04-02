package cdn

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/database"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

func Test_getAdminDatabaseEntityList(t *testing.T) {
	s, _ := newTestService(t)
	s.Mapper.Register(s.Mapper.NewTableMapping(gorpmapper.TestEncryptedData{}, "test_encrypted_data", true, "id"))

	var d1 = gorpmapper.TestEncryptedData{
		Data:                 "data-1",
		SensitiveData:        "sensitive-data-1",
		AnotherSensitiveData: "another-sensitive-data-1",
		SensitiveJsonData:    gorpmapper.SensitiveJsonData{Data: "json-sentitive-data-1"},
	}
	require.NoError(t, s.Mapper.InsertAndSign(context.TODO(), &test.FakeTransaction{
		DbMap: s.mustDB(),
	}, &d1))

	uri := s.Router.GetRoute("GET", s.getAdminDatabaseEntityList, nil)
	w := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(w, newRequest(t, "GET", uri, nil))
	require.Equal(t, 200, w.Code)

	var res []sdk.DatabaseEntity
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

	var found bool
	for _, entity := range res {
		if entity.Name == "gorpmapper.TestEncryptedData" {
			found = true
			require.True(t, entity.Encrypted, "gorpmapper.TestEncryptedData entity should be encrypted")
			require.True(t, entity.Signed, "gorpmapper.TestEncryptedData entity should be signed")
			require.True(t, len(entity.CanonicalForms) >= 1)
			break
		}
	}
	require.True(t, found, "gorpmapper.TestEncryptedData entity should be listed")
}

func Test_getAdminDatabaseEntity(t *testing.T) {
	s, _ := newTestService(t)

	s.Mapper.Register(s.Mapper.NewTableMapping(gorpmapper.TestEncryptedData{}, "test_encrypted_data", true, "id"))

	var d1 = gorpmapper.TestEncryptedData{
		Data:                 "data-1",
		SensitiveData:        "sensitive-data-1",
		AnotherSensitiveData: "another-sensitive-data-1",
		SensitiveJsonData:    gorpmapper.SensitiveJsonData{Data: "json-sentitive-data-1"},
	}
	require.NoError(t, s.Mapper.InsertAndSign(context.TODO(), &test.FakeTransaction{
		DbMap: s.mustDB(),
	}, &d1))

	var d2 = gorpmapper.TestEncryptedData{
		Data:                 "canonical-variant-data-2",
		SensitiveData:        "sensitive-data-2",
		AnotherSensitiveData: "another-sensitive-data-2",
		SensitiveJsonData:    gorpmapper.SensitiveJsonData{Data: "json-sentitive-data-2"},
	}
	require.NoError(t, s.Mapper.InsertAndSign(context.TODO(), &test.FakeTransaction{
		DbMap: s.mustDB(),
	}, &d2))

	uri := s.Router.GetRoute("GET", s.getAdminDatabaseEntity, map[string]string{"entity": "gorpmapper.TestEncryptedData"})
	w := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(w, newRequest(t, "GET", uri, nil))
	require.Equal(t, 200, w.Code)
	var pks []string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &pks))

	var d1Found, d2Found bool
	for _, pk := range pks {
		if pk == strconv.FormatInt(d1.ID, 10) {
			d1Found = true
		}
		if pk == strconv.FormatInt(d2.ID, 10) {
			d2Found = true
		}
	}
	require.True(t, d1Found, "gorpmapper.TestEncryptedData d1 entity pk should be listed")
	require.True(t, d2Found, "gorpmapper.TestEncryptedData d2 entity pk should be listed")

	lastestCanonicalForm, _ := d2.Canonical().Latest()
	sha := gorpmapper.GetSigner(lastestCanonicalForm)
	uri = s.Router.GetRoute("GET", s.getAdminDatabaseEntity, map[string]string{"entity": "gorpmapper.TestEncryptedData"})
	w = httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(w, newRequest(t, "GET", uri, nil, cdsclient.Signer(sha)))
	require.Equal(t, 200, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &pks))

	d1Found, d2Found = false, false
	for _, pk := range pks {
		if pk == strconv.FormatInt(d1.ID, 10) {
			d1Found = true
		}
		if pk == strconv.FormatInt(d2.ID, 10) {
			d2Found = true
		}
	}
	require.False(t, d1Found, "gorpmapper.TestEncryptedData d1 entity pk should not be listed")
	require.True(t, d2Found, "gorpmapper.TestEncryptedData d2 entity pk should be listed")
}

func Test_postAdminDatabaseEntityInfoAndRoll(t *testing.T) {
	s, _ := newTestService(t)
	s.Mapper.Register(s.Mapper.NewTableMapping(gorpmapper.TestEncryptedData{}, "test_encrypted_data", true, "id"))

	var d1 = gorpmapper.TestEncryptedData{
		Data:                 "data-1",
		SensitiveData:        "sensitive-data-1",
		AnotherSensitiveData: "another-sensitive-data-1",
		SensitiveJsonData:    gorpmapper.SensitiveJsonData{Data: "json-sentitive-data-1"},
	}
	require.NoError(t, s.Mapper.InsertAndSign(context.TODO(), &test.FakeTransaction{
		DbMap: s.mustDB(),
	}, &d1))
	pk := strconv.FormatInt(d1.ID, 10)

	uri := s.Router.GetRoute(http.MethodPost, s.postAdminDatabaseEntityInfo, map[string]string{"entity": "gorpmapper.TestEncryptedData"})
	req := newRequest(t, http.MethodPost, uri, []string{pk})
	w := httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	var res []sdk.DatabaseEntityInfo
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
	require.Len(t, res, 1)
	require.Equal(t, pk, res[0].PK)
	require.True(t, res[0].Encrypted)
	require.Equal(t, int64(1234567890), res[0].EncryptionTS)
	require.True(t, res[0].Signed)
	require.Equal(t, int64(1234567890), res[0].SignatureTS)

	s.Mapper = gorpmapper.New()
	s.Mapper.Register(s.Mapper.NewTableMapping(gorpmapper.TestEncryptedData{}, "test_encrypted_data", true, "id"))
	now := time.Now().Unix()
	sigKeys := database.RollingKeyConfig{
		Cipher: "hmac",
		Keys: []database.KeyConfig{test.DefaultSignatureKey, {
			Timestamp: now,
			Key:       test.DefaultSignatureKey.Key[1:] + "0",
		}},
	}
	encryptKeys := database.RollingKeyConfig{
		Cipher: "xchacha20-poly1305",
		Keys: []database.KeyConfig{test.DefaultEncryptionKey, {
			Timestamp: now,
			Key:       test.DefaultEncryptionKey.Key[1:] + "0",
		}},
	}
	signatureKeyConfig := sigKeys.GetKeys(gorpmapper.KeySignIdentifier)
	encryptionKeyConfig := encryptKeys.GetKeys(gorpmapper.KeyEncryptionIdentifier)
	require.NoError(t, s.Mapper.ConfigureKeys(signatureKeyConfig, encryptionKeyConfig), "cannot setup database keys")

	uri = s.Router.GetRoute(http.MethodPost, s.postAdminDatabaseEntityRoll, map[string]string{"entity": "gorpmapper.TestEncryptedData"})
	req = newRequest(t, http.MethodPost, uri, []string{pk})
	w = httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
	require.Len(t, res, 1)
	require.Equal(t, pk, res[0].PK)
	require.True(t, res[0].Encrypted)
	require.Equal(t, now, res[0].EncryptionTS)
	require.True(t, res[0].Signed)
	require.Equal(t, now, res[0].SignatureTS)

	uri = s.Router.GetRoute(http.MethodPost, s.postAdminDatabaseEntityInfo, map[string]string{"entity": "gorpmapper.TestEncryptedData"})
	req = newRequest(t, http.MethodPost, uri, []string{pk})
	w = httptest.NewRecorder()
	s.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
	require.Len(t, res, 1)
	require.Equal(t, pk, res[0].PK)
	require.True(t, res[0].Encrypted)
	require.Equal(t, now, res[0].EncryptionTS)
	require.True(t, res[0].Signed)
	require.Equal(t, now, res[0].SignatureTS)
}
