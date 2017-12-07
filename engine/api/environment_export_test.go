package api

import (
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_getEnvironmentExportHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)
	envName := sdk.RandomString(10)
	env := &sdk.Environment{
		ProjectID: proj.ID,
		Name:      envName,
	}
	test.NoError(t, environment.InsertEnvironment(db, env))

	v1 := &sdk.Variable{
		Name:  "var1",
		Value: "value 1",
		Type:  sdk.StringVariable,
	}
	test.NoError(t, environment.InsertVariable(db, env.ID, v1, u))

	v2 := &sdk.Variable{
		Name:  "var2",
		Value: "value 2",
		Type:  sdk.SecretVariable,
	}
	test.NoError(t, environment.InsertVariable(db, env.ID, v2, u))

	//Insert ssh and gpg keys
	k := &sdk.EnvironmentKey{
		Key: sdk.Key{
			Name: "mykey",
			Type: sdk.KeyTypePgp,
		},
		EnvironmentID: env.ID,
	}
	kid, pubR, privR, err := keys.GeneratePGPKeyPair(k.Name)
	test.NoError(t, err)
	pub, _ := ioutil.ReadAll(pubR)
	priv, _ := ioutil.ReadAll(privR)

	k.Public = string(pub)
	k.Private = string(priv)
	k.KeyID = kid
	test.NoError(t, environment.InsertKey(api.mustDB(), k))

	k2 := &sdk.EnvironmentKey{
		Key: sdk.Key{
			Name: "mykey-ssh",
			Type: sdk.KeyTypeSsh,
		},
		EnvironmentID: env.ID,
	}
	pubR, privR, err = keys.GenerateSSHKeyPair(k2.Name)
	test.NoError(t, err)
	pub, _ = ioutil.ReadAll(pubR)
	priv, _ = ioutil.ReadAll(privR)

	k2.Public = string(pub)
	k2.Private = string(priv)
	k2.KeyID = kid
	test.NoError(t, environment.InsertKey(api.mustDB(), k2))

	//Prepare request
	vars := map[string]string{
		"key": proj.Key,
		"permEnvironmentName": env.Name,
	}
	uri := api.Router.GetRoute("GET", api.getEnvironmentExportHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())
}
