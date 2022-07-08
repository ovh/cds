package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_crudGPGKey(t *testing.T) {
	api, db, _ := newTestAPI(t)

	user1, pass := assets.InsertLambdaUser(t, db)

	k, err := keys.GenerateKey(sdk.RandomString(10), sdk.KeyTypePGP)
	require.NoError(t, err)

	//------------ Create key

	vars := map[string]string{
		"user": user1.Username,
	}
	uri := api.Router.GetRouteV2("POST", api.postUserGPGGKeyHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, user1, pass, "POST", uri, nil)

	bts, _ := json.Marshal(sdk.UserGPGKey{PublicKey: k.Public})
	// Here, we insert the vcs server as a CDS user (not administrator)
	req.Body = io.NopCloser(bytes.NewReader(bts))

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	var mykey sdk.UserGPGKey
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &mykey))
	require.True(t, strings.HasSuffix(mykey.KeyID, k.KeyID))

	//----------- List keys

	uriGetAll := api.Router.GetRouteV2("GET", api.getUserGPGKeysHandler, vars)
	test.NotEmpty(t, uriGetAll)
	reqAll := assets.NewAuthentifiedRequest(t, user1, pass, "GET", uriGetAll, nil)

	// Here, we insert the vcs server as a CDS user (not administrator)
	reqAll.Body = io.NopCloser(bytes.NewReader(bts))

	wAll := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wAll, reqAll)
	require.Equal(t, 200, wAll.Code)

	var mykeys []sdk.UserGPGKey
	body := wAll.Body.Bytes()
	require.NoError(t, json.Unmarshal(body, &mykeys))
	require.Equal(t, 1, len(mykeys))
	require.True(t, strings.HasSuffix(mykeys[0].KeyID, k.KeyID))

	//----------- Delete a specific key
	vars["gpgKeyID"] = mykey.KeyID
	uriDel := api.Router.GetRouteV2("DELETE", api.deleteUserGPGKey, vars)
	test.NotEmpty(t, uriDel)
	reqDel := assets.NewAuthentifiedRequest(t, user1, pass, "DELETE", uriDel, nil)

	wDel := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wDel, reqDel)
	require.Equal(t, 204, wDel.Code)

	//----------- List a specific
	uriGetOne := api.Router.GetRouteV2("GET", api.getUserGPGKeyHandler, vars)
	test.NotEmpty(t, uriGetOne)
	reqGetOne := assets.NewAuthentifiedRequest(t, user1, pass, "GET", uriGetOne, nil)

	// Here, we insert the vcs server as a CDS user (not administrator)
	reqGetOne.Body = io.NopCloser(bytes.NewReader(bts))

	wGetOne := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wGetOne, reqGetOne)
	require.Equal(t, 404, wGetOne.Code)

}
