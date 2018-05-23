package api

import (
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/platform"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func Test_postApplicationImportHandler_NewAppFromYAMLWithoutSecret(t *testing.T) {
	api, db, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := api.Router.GetRoute("POST", api.postApplicationImportHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, nil)

	body := `version: v1.0
name: myNewApp
variables:
  var1:
    value: value 1
  var2:
    type: text
    value: value 2
  var3:
    type: boolean
    value: true
  var4:
    type: number
    value: 42`
	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())

	app, err := application.LoadByName(db, api.Cache, proj.Key, "myNewApp", nil, application.LoadOptions.WithVariables, application.LoadOptions.WithGroups)
	test.NoError(t, err)

	assert.NotNil(t, app)
	assert.Equal(t, "myNewApp", app.Name)

	//Check default permission which should be set to the project ones
	for _, perm := range proj.ProjectGroups {
		var found bool
		for _, aperm := range app.ApplicationGroups {
			if aperm.Group.Name == perm.Group.Name && aperm.Permission == perm.Permission {
				found = true
			}
		}
		assert.True(t, found, "Group %s - %d not found", perm.Group.Name, perm.Permission)
	}

	//Check variables
	for _, v := range app.Variable {
		switch v.Name {
		case "var1":
			assert.True(t, v.Type == sdk.StringVariable, "var1.type should be type string")
			assert.True(t, v.Value == "value 1", "var1.value is wrong")
		case "var2":
			assert.True(t, v.Type == sdk.TextVariable, "var2.type should be type text")
			assert.True(t, v.Value == "value 2", "var2.value is wrong")
		case "var3":
			assert.True(t, v.Type == sdk.BooleanVariable, "var3.type should be type bool")
			assert.True(t, v.Value == "true", "var3.value is wrong")
		case "var4":
			assert.True(t, v.Type == sdk.NumberParameter, "var4.type should be type number")
			assert.True(t, v.Value == "42", "var4.value is wrong")
		default:
			t.Errorf("Unexpected variable %+v", v)
		}
	}

}

func Test_postApplicationImportHandler_NewAppFromYAMLWithKeysAndSecrets(t *testing.T) {
	api, db, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	//We will create an app, with a pgp key, export it then import as a new application(with a different name)
	//This is also a good test for export secrets

	app := &sdk.Application{
		Name: "myNewApp",
	}
	test.NoError(t, application.Insert(db, api.Cache, proj, app, u))

	k := &sdk.ApplicationKey{
		Key: sdk.Key{
			Name: "app-mykey",
			Type: "pgp",
		},
		ApplicationID: app.ID,
	}

	kpgp, err := keys.GeneratePGPKeyPair(k.Name)
	test.NoError(t, err)
	k.Public = kpgp.Public
	k.Private = kpgp.Private
	k.KeyID = kpgp.KeyID
	if err := application.InsertKey(api.mustDB(), k); err != nil {
		t.Fatal(err)
	}

	test.NoError(t, application.InsertVariable(api.mustDB(), api.Cache, app, sdk.Variable{
		Name:  "myPassword",
		Type:  sdk.SecretVariable,
		Value: "MySecretValue",
	}, u))

	//Export all the things
	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
	}
	uri := api.Router.GetRoute("GET", api.getApplicationExportHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	body := rec.Body.String()
	t.Logf(">>%s", body)

	//Change the name of the application
	body = strings.Replace(body, app.Name, "myNewApp-1", 1)

	//Import the new application
	vars = map[string]string{
		"permProjectKey": proj.Key,
	}
	uri = api.Router.GetRoute("POST", api.postApplicationImportHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, nil)
	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	//Do the request
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())

	app, err = application.LoadByName(db, api.Cache, proj.Key, "myNewApp", nil, application.LoadOptions.WithKeys, application.LoadOptions.WithVariablesWithClearPassword)
	test.NoError(t, err)

	//Reload the application to check the keys
	app1, err := application.LoadByName(db, api.Cache, proj.Key, "myNewApp-1", nil, application.LoadOptions.WithKeys, application.LoadOptions.WithVariablesWithClearPassword)
	test.NoError(t, err)

	assert.NotNil(t, app1)
	assert.Equal(t, "myNewApp-1", app1.Name)

	//Check keys
	for _, k := range app.Keys {
		var keyFound bool
		for _, kk := range app1.Keys {
			assert.Equal(t, k.Name, kk.Name)
			assert.Equal(t, k.Public, kk.Public)
			assert.Equal(t, k.Private, kk.Private)
			assert.Equal(t, k.Type, kk.Type)
			keyFound = true
			break
		}
		assert.True(t, keyFound)
	}

	//Check variables
	for _, v := range app1.Variable {
		switch v.Name {
		case "myPassword":
			assert.True(t, v.Type == sdk.SecretVariable, "myPassword.type should be type password")
			assert.True(t, v.Value == "MySecretValue", "myPassword.value is wrong")
		default:
			t.Errorf("Unexpected variable %+v", v)
		}
	}
}

func Test_postApplicationImportHandler_NewAppFromYAMLWithKeysAndSecretsAndReImport(t *testing.T) {
	api, db, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	app := &sdk.Application{
		Name: "myNewApp",
	}
	test.NoError(t, application.Insert(db, api.Cache, proj, app, u))

	k := &sdk.ApplicationKey{
		Key: sdk.Key{
			Name: "app-mykey",
			Type: "pgp",
		},
		ApplicationID: app.ID,
	}

	kpgp, err := keys.GeneratePGPKeyPair(k.Name)
	test.NoError(t, err)
	k.Public = kpgp.Public
	k.Private = kpgp.Private
	k.KeyID = kpgp.KeyID
	if err := application.InsertKey(api.mustDB(), k); err != nil {
		t.Fatal(err)
	}

	test.NoError(t, application.InsertVariable(api.mustDB(), api.Cache, app, sdk.Variable{
		Name:  "myPassword",
		Type:  sdk.SecretVariable,
		Value: "MySecretValue",
	}, u))

	//Export all the things
	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
	}
	uri := api.Router.GetRoute("GET", api.getApplicationExportHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	body := rec.Body.String()
	t.Logf(">>%s", body)

	//Change the name of the application
	body = strings.Replace(body, app.Name, "myNewApp-1", 1)

	//Import the new application
	vars = map[string]string{
		"permProjectKey": proj.Key,
	}
	uri = api.Router.GetRoute("POST", api.postApplicationImportHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, nil)
	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	//Do the request
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())

	app, err = application.LoadByName(db, api.Cache, proj.Key, "myNewApp", nil, application.LoadOptions.WithKeys, application.LoadOptions.WithVariablesWithClearPassword)
	test.NoError(t, err)

	//Reload the application to check the keys
	app1, err := application.LoadByName(db, api.Cache, proj.Key, "myNewApp-1", nil, application.LoadOptions.WithKeys, application.LoadOptions.WithVariablesWithClearPassword)
	test.NoError(t, err)

	assert.NotNil(t, app1)
	assert.Equal(t, "myNewApp-1", app1.Name)

	//Check keys
	for _, k := range app.Keys {
		var keyFound bool
		for _, kk := range app1.Keys {
			assert.Equal(t, k.Name, kk.Name)
			assert.Equal(t, k.Public, kk.Public)
			assert.Equal(t, k.Private, kk.Private)
			assert.Equal(t, k.Type, kk.Type)
			keyFound = true
			break
		}
		assert.True(t, keyFound)
	}

	//Check variables
	for _, v := range app1.Variable {
		switch v.Name {
		case "myPassword":
			assert.True(t, v.Type == sdk.SecretVariable, "myPassword.type should be type password")
			assert.True(t, v.Value == "MySecretValue", "myPassword.value is wrong")
		default:
			t.Errorf("Unexpected variable %+v", v)
		}
	}

	//Reimport
	//Import the new application
	vars = map[string]string{
		"permProjectKey": proj.Key,
	}
	uri = api.Router.GetRoute("POST", api.postApplicationImportHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri+"?force=true", nil)
	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	//Do the request
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())

	app, err = application.LoadByName(db, api.Cache, proj.Key, "myNewApp", nil, application.LoadOptions.WithKeys, application.LoadOptions.WithVariablesWithClearPassword)
	test.NoError(t, err)

	//Reload the application to check the keys
	app1, err = application.LoadByName(db, api.Cache, proj.Key, "myNewApp-1", nil, application.LoadOptions.WithKeys, application.LoadOptions.WithVariablesWithClearPassword)
	test.NoError(t, err)

	assert.NotNil(t, app1)
	assert.Equal(t, "myNewApp-1", app1.Name)

	//Check keys
	for _, k := range app.Keys {
		var keyFound bool
		for _, kk := range app1.Keys {
			assert.Equal(t, k.Name, kk.Name)
			assert.Equal(t, k.Public, kk.Public)
			assert.Equal(t, k.Private, kk.Private)
			assert.Equal(t, k.Type, kk.Type)
			keyFound = true
			break
		}
		assert.True(t, keyFound)
	}

	//Check variables
	for _, v := range app1.Variable {
		switch v.Name {
		case "myPassword":
			assert.True(t, v.Type == sdk.SecretVariable, "myPassword.type should be type password")
			assert.True(t, v.Value == "MySecretValue", "myPassword.value is wrong")
		default:
			t.Errorf("Unexpected variable %+v", v)
		}
	}
}

func Test_postApplicationImportHandler_NewAppFromYAMLWithKeysAndSecretsAndReImportWithRegen(t *testing.T) {
	api, db, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	app := &sdk.Application{
		Name: "myNewApp",
	}
	test.NoError(t, application.Insert(db, api.Cache, proj, app, u))

	k := &sdk.ApplicationKey{
		Key: sdk.Key{
			Name: "app-mykey",
			Type: "pgp",
		},
		ApplicationID: app.ID,
	}

	kpgp, err := keys.GeneratePGPKeyPair(k.Name)
	test.NoError(t, err)
	k.Public = kpgp.Public
	k.Private = kpgp.Private
	k.KeyID = kpgp.KeyID
	if err := application.InsertKey(api.mustDB(), k); err != nil {
		t.Fatal(err)
	}

	test.NoError(t, application.InsertVariable(api.mustDB(), api.Cache, app, sdk.Variable{
		Name:  "myPassword",
		Type:  sdk.SecretVariable,
		Value: "MySecretValue",
	}, u))

	//Export all the things
	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
	}
	uri := api.Router.GetRoute("GET", api.getApplicationExportHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	body := rec.Body.String()
	t.Logf(">>%s", body)

	eapp := &exportentities.Application{}
	test.NoError(t, yaml.Unmarshal([]byte(body), eapp))

	False := false
	ek := eapp.Keys[k.Name]
	ek.Regen = &False
	ek.Value = ""
	eapp.Keys[k.Name] = ek

	btes, err := yaml.Marshal(eapp)
	body = string(btes)

	t.Log(body)

	//Import the new application
	vars = map[string]string{
		"permProjectKey": proj.Key,
	}
	uri = api.Router.GetRoute("POST", api.postApplicationImportHandler, vars)
	test.NotEmpty(t, uri)
	uri += "?force=true"
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, nil)
	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	//Do the request
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())

	app, err = application.LoadByName(db, api.Cache, proj.Key, "myNewApp", nil, application.LoadOptions.WithKeys, application.LoadOptions.WithVariablesWithClearPassword)
	test.NoError(t, err)
	//Check keys
	for _, k := range app.Keys {
		assert.NotEmpty(t, k.Private)
		assert.NotEmpty(t, k.Public)
	}
}

func Test_postApplicationImportHandler_NewAppFromYAMLWithEmptyKey(t *testing.T) {
	api, db, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := api.Router.GetRoute("POST", api.postApplicationImportHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, nil)

	body := `version: v1.0
name: myNewApp
keys:
  app-myPGPkey:
    type: pgp
    regen: true
  app-mySSHKey:
    type: ssh`
	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())

	app, err := application.LoadByName(db, api.Cache, proj.Key, "myNewApp", nil, application.LoadOptions.WithKeys)
	test.NoError(t, err)

	assert.NotNil(t, app)
	assert.Equal(t, "myNewApp", app.Name)

	var myPGPkey, mySSHKey bool
	for _, k := range app.Keys {
		switch k.Name {
		case "app-myPGPkey":
			myPGPkey = true
			assert.NotEmpty(t, k.KeyID)
			assert.NotEmpty(t, k.Private)
			assert.NotEmpty(t, k.Public)
			assert.NotEmpty(t, k.Type)
		case "app-mySSHKey":
			mySSHKey = true
			assert.NotEmpty(t, k.Private)
			assert.NotEmpty(t, k.Public)
			assert.NotEmpty(t, k.Type)
		default:
			t.Errorf("Unexpected variable %+v", k)
		}
	}
	assert.True(t, myPGPkey, "myPGPkey not found")
	assert.True(t, mySSHKey, "mySSHKey not found")

}

func Test_postApplicationImportHandler_ExistingAppFromYAMLWithoutForce(t *testing.T) {
	api, db, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	app := sdk.Application{
		Name: "myNewApp",
	}
	test.NoError(t, application.Insert(db, api.Cache, proj, &app, u))

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := api.Router.GetRoute("POST", api.postApplicationImportHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, nil)

	body := `version: v1.0
name: myNewApp`
	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 409, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())
}

func Test_postApplicationImportHandler_ExistingAppFromYAMLInheritPermissions(t *testing.T) {
	api, db, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	app := sdk.Application{
		Name: "myNewApp",
	}
	test.NoError(t, application.Insert(db, api.Cache, proj, &app, u))

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := api.Router.GetRoute("POST", api.postApplicationImportHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri+"?force=true", nil)

	body := `version: v1.0
name: myNewApp`
	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())
}

func Test_postApplicationImportHandler_ExistingAppWithDeploymentStrategy(t *testing.T) {
	api, db, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	pf := sdk.PlatformModel{
		Name:       "test-deploy-2",
		Deployment: true,
	}
	test.NoError(t, platform.InsertModel(db, &pf))
	defer platform.DeleteModel(db, pf.ID)

	pp := sdk.ProjectPlatform{
		Model:           pf,
		Name:            pf.Name,
		PlatformModelID: pf.ID,
		ProjectID:       proj.ID,
		Config: sdk.PlatformConfig{
			"token": sdk.PlatformConfigValue{
				Type:  sdk.PlatformConfigTypePassword,
				Value: "my-secret-token",
			},
			"url": sdk.PlatformConfigValue{
				Type:  sdk.PlatformConfigTypeString,
				Value: "my-url",
			},
		},
	}
	test.NoError(t, platform.InsertPlatform(db, &pp))

	app := sdk.Application{
		Name: "myNewApp",
	}
	test.NoError(t, application.Insert(db, api.Cache, proj, &app, u))

	test.NoError(t, application.SetDeploymentStrategy(db, proj.ID, app.ID, pf.ID, pp.Name, sdk.PlatformConfig{
		"token": sdk.PlatformConfigValue{
			Type:  sdk.PlatformConfigTypePassword,
			Value: "my-secret-token-2",
		},
		"url": sdk.PlatformConfigValue{
			Type:  sdk.PlatformConfigTypeString,
			Value: "my-url-2",
		},
	}))

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := api.Router.GetRoute("POST", api.postApplicationImportHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri+"?force=true", nil)

	body := `version: v1.0
name: myNewApp
deployments:
  test-deploy-2:
    url: 
      value: my-url-3`
	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())

	//Now get it !

	//Prepare request
	vars = map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
	}
	uri = api.Router.GetRoute("GET", api.getApplicationExportHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())
}
