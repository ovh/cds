package api

import (
	"bytes"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

func Test_postApplicationImportHandler_NewAppFromYAMLWithoutSecret(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

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

	app, err := application.LoadByName(db, api.Cache, proj.Key, "myNewApp", application.LoadOptions.WithVariables)
	test.NoError(t, err)

	assert.NotNil(t, app)
	assert.Equal(t, "myNewApp", app.Name)

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
	api, db, _, end := newTestAPI(t)
	defer end()

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

	app, err = application.LoadByName(db, api.Cache, proj.Key, "myNewApp", application.LoadOptions.WithKeys, application.LoadOptions.WithVariablesWithClearPassword)
	test.NoError(t, err)

	//Reload the application to check the keys
	app1, err := application.LoadByName(db, api.Cache, proj.Key, "myNewApp-1", application.LoadOptions.WithKeys, application.LoadOptions.WithVariablesWithClearPassword)
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
	api, db, _, end := newTestAPI(t)
	defer end()

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

	app, err = application.LoadByName(db, api.Cache, proj.Key, "myNewApp", application.LoadOptions.WithKeys, application.LoadOptions.WithVariablesWithClearPassword)
	test.NoError(t, err)

	//Reload the application to check the keys
	app1, err := application.LoadByName(db, api.Cache, proj.Key, "myNewApp-1", application.LoadOptions.WithKeys, application.LoadOptions.WithVariablesWithClearPassword)
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

	app, err = application.LoadByName(db, api.Cache, proj.Key, "myNewApp", application.LoadOptions.WithKeys, application.LoadOptions.WithVariablesWithClearPassword)
	test.NoError(t, err)

	//Reload the application to check the keys
	app1, err = application.LoadByName(db, api.Cache, proj.Key, "myNewApp-1", application.LoadOptions.WithKeys, application.LoadOptions.WithVariablesWithClearPassword)
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
	// init project and application for test
	api, db, _, end := newTestAPI(t)
	defer end()

	u, pass := assets.InsertAdminUser(db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	app := &sdk.Application{
		Name: "myNewApp",
	}
	test.NoError(t, application.Insert(db, api.Cache, proj, app, u))

	// create password, pgp and ssh keys
	k1 := &sdk.ApplicationKey{
		Key: sdk.Key{
			Name: "app-key-1",
			Type: "pgp",
		},
		ApplicationID: app.ID,
	}

	kpgp, err := keys.GeneratePGPKeyPair(k1.Name)
	test.NoError(t, err)
	k1.Public = kpgp.Public
	k1.Private = kpgp.Private
	k1.KeyID = kpgp.KeyID
	test.NoError(t, application.InsertKey(api.mustDB(), k1))

	// create password, pgp and ssh keys
	k2 := &sdk.ApplicationKey{
		Key: sdk.Key{
			Name: "app-key-2",
			Type: "ssh",
		},
		ApplicationID: app.ID,
	}

	kssh, err := keys.GenerateSSHKey(k2.Name)
	test.NoError(t, err)
	k2.Public = kssh.Public
	k2.Private = kssh.Private
	k2.KeyID = kssh.KeyID
	test.NoError(t, application.InsertKey(api.mustDB(), k2))

	test.NoError(t, application.InsertVariable(api.mustDB(), api.Cache, app, sdk.Variable{
		Name:  "myPassword",
		Type:  sdk.SecretVariable,
		Value: "MySecretValue",
	}, u))

	// check that keys secrets are well stored
	app, err = application.LoadByName(db, api.Cache, proj.Key, "myNewApp",
		application.LoadOptions.WithClearKeys,
		application.LoadOptions.WithVariablesWithClearPassword,
	)
	test.NoError(t, err)
	test.Equal(t, 1, len(app.Variable))
	test.Equal(t, "MySecretValue", app.Variable[0].Value)
	test.Equal(t, 2, len(app.Keys))
	test.Equal(t, kpgp.Private, app.Keys[0].Private)
	test.Equal(t, kssh.Private, app.Keys[1].Private)

	// export the app then import it with regen false
	uri := api.Router.GetRoute("GET", api.getApplicationExportHandler, map[string]string{
		"permProjectKey":  proj.Key,
		"applicationName": app.Name,
	})
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
	test.Equal(t, 1, len(eapp.Variables))
	test.Equal(t, 2, len(eapp.Keys))

	False := false
	ek1 := eapp.Keys[k1.Name]
	ek1.Regen = &False
	ek1.Value = ""
	eapp.Keys[k1.Name] = ek1

	ek2 := eapp.Keys[k2.Name]
	ek2.Regen = &False
	ek2.Value = ""
	eapp.Keys[k2.Name] = ek2

	btes, err := yaml.Marshal(eapp)
	body = string(btes)

	t.Log(body)

	// import the new application then check secrets values.
	uri = api.Router.GetRoute("POST", api.postApplicationImportHandler, map[string]string{
		"permProjectKey": proj.Key,
	})
	test.NotEmpty(t, uri)
	uri += "?force=true"
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, nil)
	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	t.Logf(">>%s", rec.Body.String())

	app, err = application.LoadByName(db, api.Cache, proj.Key, "myNewApp",
		application.LoadOptions.WithClearKeys,
		application.LoadOptions.WithVariablesWithClearPassword,
	)
	test.NoError(t, err)
	test.Equal(t, 1, len(app.Variable))
	test.Equal(t, "MySecretValue", app.Variable[0].Value)
	test.Equal(t, 2, len(app.Keys))
	test.Equal(t, kpgp.Private, app.Keys[0].Private)
	test.Equal(t, kssh.Private, app.Keys[1].Private)
}

func Test_postApplicationImportHandler_NewAppFromYAMLWithEmptyKey(t *testing.T) {
	api, db, _, end := newTestAPI(t)
	defer end()

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

	app, err := application.LoadByName(db, api.Cache, proj.Key, "myNewApp", application.LoadOptions.WithKeys)
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
	api, db, _, end := newTestAPI(t)
	defer end()

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
	api, db, _, end := newTestAPI(t)
	defer end()

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
	api, db, _, end := newTestAPI(t)
	defer end()

	u, pass := assets.InsertAdminUser(db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	pfname := sdk.RandomString(10)
	pf := sdk.IntegrationModel{
		Name:       pfname,
		Deployment: true,
		DeploymentDefaultConfig: sdk.IntegrationConfig{
			"token": sdk.IntegrationConfigValue{
				Type:  sdk.IntegrationConfigTypePassword,
				Value: "my-secret-token",
			},
			"url": sdk.IntegrationConfigValue{
				Type:  sdk.IntegrationConfigTypeString,
				Value: "my-url",
			},
		},
	}
	test.NoError(t, integration.InsertModel(db, &pf))
	defer func() { _ = integration.DeleteModel(db, pf.ID) }()

	pp := sdk.ProjectIntegration{
		Model:              pf,
		Name:               pf.Name,
		IntegrationModelID: pf.ID,
		ProjectID:          proj.ID,
	}
	test.NoError(t, integration.InsertIntegration(db, &pp))

	app := sdk.Application{
		Name: "myNewApp",
	}
	test.NoError(t, application.Insert(db, api.Cache, proj, &app, u))

	test.NoError(t, application.SetDeploymentStrategy(db, proj.ID, app.ID, pf.ID, pp.Name, sdk.IntegrationConfig{
		"token": sdk.IntegrationConfigValue{
			Type:  sdk.IntegrationConfigTypePassword,
			Value: "my-secret-token-2",
		},
		"url": sdk.IntegrationConfigValue{
			Type:  sdk.IntegrationConfigTypeString,
			Value: "my-url-2",
		},
	}))

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

	body := rec.Body.String()

	//Prepare request
	vars = map[string]string{
		"permProjectKey": proj.Key,
	}
	uri = api.Router.GetRoute("POST", api.postApplicationImportHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri+"?force=true", nil)

	body = strings.Replace(body, "my-url-2", "my-url-3", 1)

	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	//Do the request
	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())

	//Now get it !

	//Prepare request
	vars = map[string]string{
		"permProjectKey":  proj.Key,
		"applicationName": app.Name,
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

	actualApp, err := application.LoadByName(api.mustDB(), api.Cache, proj.Key, app.Name, application.LoadOptions.WithClearDeploymentStrategies)
	test.NoError(t, err)
	assert.Equal(t, "my-secret-token-2", actualApp.DeploymentStrategies[pfname]["token"].Value)
	assert.Equal(t, "my-url-3", actualApp.DeploymentStrategies[pfname]["url"].Value)
}

func Test_postApplicationImportHandler_DontOverrideDeploymentPasswordIfNotGiven(t *testing.T) {
	// init test case, create a project with deployment integration then an application with deployment config
	api, db, _, end := newTestAPI(t)
	defer end()

	u, pass := assets.InsertAdminUser(db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	pfname := sdk.RandomString(10)
	pf := sdk.IntegrationModel{
		Name:       pfname,
		Deployment: true,
		DeploymentDefaultConfig: sdk.IntegrationConfig{
			"token": sdk.IntegrationConfigValue{
				Type:  sdk.IntegrationConfigTypePassword,
				Value: "my-secret-token",
			},
			"url": sdk.IntegrationConfigValue{
				Type:  sdk.IntegrationConfigTypeString,
				Value: "my-url",
			},
		},
	}
	test.NoError(t, integration.InsertModel(db, &pf))
	defer func() { _ = integration.DeleteModel(db, pf.ID) }()

	pp := sdk.ProjectIntegration{
		Model:              pf,
		Name:               pf.Name,
		IntegrationModelID: pf.ID,
		ProjectID:          proj.ID,
	}
	test.NoError(t, integration.InsertIntegration(db, &pp))

	app := sdk.Application{
		Name: "myNewApp",
	}
	test.NoError(t, application.Insert(db, api.Cache, proj, &app, u))

	test.NoError(t, application.SetDeploymentStrategy(db, proj.ID, app.ID, pf.ID, pp.Name, sdk.IntegrationConfig{
		"token": sdk.IntegrationConfigValue{
			Type:  sdk.IntegrationConfigTypePassword,
			Value: "my-secret-token-2",
		},
		"url": sdk.IntegrationConfigValue{
			Type:  sdk.IntegrationConfigTypeString,
			Value: "my-url",
		},
	}))

	// import updated application without deployment token

	appUpdated := exportentities.Application{
		Name: "myNewApp",
		DeploymentStrategies: map[string]map[string]exportentities.VariableValue{
			pp.Name: {
				"url": exportentities.VariableValue{
					Type:  sdk.IntegrationConfigTypeString,
					Value: "my-url-2",
				},
			},
		},
	}

	uri := api.Router.GetRoute("POST", api.postApplicationImportHandler, map[string]string{
		"permProjectKey": proj.Key,
	})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri+"?force=true", nil)

	buf, err := yaml.Marshal(appUpdated)
	test.NoError(t, err)
	req.Body = ioutil.NopCloser(bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/x-yaml")

	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	t.Logf(">>%s", rec.Body.String())

	// check that the token is still present in the application

	uri = api.Router.GetRoute("GET", api.getApplicationExportHandler, map[string]string{
		"permProjectKey":  proj.Key,
		"applicationName": app.Name,
	})
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	rec = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	t.Logf(">>%s", rec.Body.String())

	actualApp, err := application.LoadByName(api.mustDB(), api.Cache, proj.Key, app.Name, application.LoadOptions.WithClearDeploymentStrategies)
	test.NoError(t, err)
	assert.Equal(t, "my-secret-token-2", actualApp.DeploymentStrategies[pfname]["token"].Value)
	assert.Equal(t, "my-url-2", actualApp.DeploymentStrategies[pfname]["url"].Value)
}
