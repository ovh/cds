package api

import (
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_getApplicationExportHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	require.NotNil(t, proj)
	appName := sdk.RandomString(10)
	app := &sdk.Application{
		Name: appName,
	}
	require.NoError(t, application.Insert(db, *proj, app))

	v1 := sdk.ApplicationVariable{
		Name:  "var1",
		Value: "value 1",
		Type:  sdk.StringVariable,
	}

	require.NoError(t, application.InsertVariable(db, app.ID, &v1, u))

	v2 := sdk.ApplicationVariable{
		Name:  "var2",
		Value: "value 2",
		Type:  sdk.SecretVariable,
	}

	require.NoError(t, application.InsertVariable(db, app.ID, &v2, u))

	//Insert ssh and gpg keys
	k := &sdk.ApplicationKey{
		Name:          "mykey",
		Type:          sdk.KeyTypePGP,
		ApplicationID: app.ID,
	}
	kk, err := keys.GeneratePGPKeyPair(k.Name)
	require.NoError(t, err)

	k.Public = kk.Public
	k.Private = kk.Private
	k.KeyID = kk.KeyID
	require.NoError(t, application.InsertKey(db, k))

	k2 := &sdk.ApplicationKey{
		Name:          "mykey-ssh",
		Type:          sdk.KeyTypeSSH,
		ApplicationID: app.ID,
	}
	kssh, err := keys.GenerateSSHKey(k2.Name)
	require.NoError(t, err)

	k2.Public = kssh.Public
	k2.Private = kssh.Private
	k2.KeyID = kssh.KeyID
	require.NoError(t, application.InsertKey(db, k2))

	//Prepare request
	vars := map[string]string{
		"permProjectKey":  proj.Key,
		"applicationName": app.Name,
	}
	uri := api.Router.GetRoute("GET", api.getApplicationExportHandler, vars)
	require.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())
}
