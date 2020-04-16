package api

import (
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func Test_getApplicationExportHandler(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

	u, pass := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	test.NotNil(t, proj)
	appName := sdk.RandomString(10)
	app := &sdk.Application{
		Name: appName,
	}
	if err := application.Insert(db, *proj, app); err != nil {
		t.Fatal(err)
	}

	v1 := sdk.Variable{
		Name:  "var1",
		Value: "value 1",
		Type:  sdk.StringVariable,
	}

	test.NoError(t, application.InsertVariable(db, app.ID, &v1, u))

	v2 := sdk.Variable{
		Name:  "var2",
		Value: "value 2",
		Type:  sdk.SecretVariable,
	}

	test.NoError(t, application.InsertVariable(db, app.ID, &v2, u))

	//Insert ssh and gpg keys
	k := &sdk.ApplicationKey{
		Name:          "mykey",
		Type:          sdk.KeyTypePGP,
		ApplicationID: app.ID,
	}
	kk, err := keys.GeneratePGPKeyPair(k.Name)
	test.NoError(t, err)

	k.Public = kk.Public
	k.Private = kk.Private
	k.KeyID = kk.KeyID
	test.NoError(t, application.InsertKey(api.mustDB(), k))

	k2 := &sdk.ApplicationKey{
		Name:          "mykey-ssh",
		Type:          sdk.KeyTypeSSH,
		ApplicationID: app.ID,
	}
	kssh, err := keys.GenerateSSHKey(k2.Name)
	test.NoError(t, err)

	k2.Public = kssh.Public
	k2.Private = kssh.Private
	k2.KeyID = kssh.KeyID
	test.NoError(t, application.InsertKey(api.mustDB(), k2))

	//Prepare request
	vars := map[string]string{
		"permProjectKey":  proj.Key,
		"applicationName": app.Name,
	}
	uri := api.Router.GetRoute("GET", api.getApplicationExportHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())
}
