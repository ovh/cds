package api

import (
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func Test_postEnvironmentImportHandler_NewEnvFromYAMLWithoutSecret(t *testing.T) {
	api, db, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := api.Router.GetRoute("POST", api.postEnvironmentImportHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, nil)

	body := `version: v1.0
name: myNewEnv
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

	env, err := environment.LoadEnvironmentByName(db, proj.Key, "myNewEnv")
	test.NoError(t, err)

	assert.NotNil(t, env)
	assert.Equal(t, "myNewEnv", env.Name)

	//Check default permission which should be set to the project ones
	for _, perm := range proj.ProjectGroups {
		var found bool
		for _, aperm := range env.EnvironmentGroups {
			if aperm.Group.Name == perm.Group.Name && aperm.Permission == perm.Permission {
				found = true
			}
		}
		assert.True(t, found, "Group %s - %d not found", perm.Group.Name, perm.Permission)
	}

	//Check variables
	for _, v := range env.Variable {
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

func Test_postEnvironmentImportHandler_NewEnvFromYAMLWithKeysAndSecrets(t *testing.T) {
	api, db, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	//We will create an env, with a pgp key, export it then import as a new environment(with a different name)
	//This is also a good test for export secrets

	env := &sdk.Environment{
		Name:      "myNewEnv",
		ProjectID: proj.ID,
	}
	test.NoError(t, environment.InsertEnvironment(db, env))

	k := &sdk.EnvironmentKey{
		Key: sdk.Key{
			Name: "mykey",
			Type: "pgp",
		},
		EnvironmentID: env.ID,
	}

	kpgp, err := keys.GeneratePGPKeyPair(k.Name)
	test.NoError(t, err)
	k.Public = kpgp.Public
	k.Private = kpgp.Private
	k.KeyID = kpgp.KeyID
	if err := environment.InsertKey(api.mustDB(), k); err != nil {
		t.Fatal(err)
	}

	test.NoError(t, environment.InsertVariable(db, env.ID, &sdk.Variable{
		Name:  "myPassword",
		Type:  sdk.SecretVariable,
		Value: "MySecretValue",
	}, u))

	//Export all the things
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
	body := rec.Body.String()
	t.Logf(">>export:%s", body)

	//Change the name of the environment
	body = strings.Replace(body, env.Name, "myNewEnv-1", 1)

	//Import the new environment
	vars = map[string]string{
		"permProjectKey": proj.Key,
	}
	uri = api.Router.GetRoute("POST", api.postEnvironmentImportHandler, vars)
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

	env, err = environment.LoadEnvironmentByName(db, proj.Key, "myNewEnv")
	test.NoError(t, err)
	// reload variables with clear password
	variables, errLoadVars := environment.GetAllVariable(db, proj.Key, "myNewEnv", environment.WithClearPassword())
	test.NoError(t, errLoadVars)
	env.Variable = variables
	test.NoError(t, environment.LoadAllDecryptedKeys(db, env))

	env1, err := environment.LoadEnvironmentByName(db, proj.Key, "myNewEnv-1")
	test.NoError(t, err)
	// reload variables with clear password
	variables1, errLoadVariables := environment.GetAllVariable(db, proj.Key, "myNewEnv-1", environment.WithClearPassword())
	test.NoError(t, errLoadVariables)
	env1.Variable = variables1
	test.NoError(t, environment.LoadAllDecryptedKeys(db, env1))

	assert.NotNil(t, env1)
	assert.Equal(t, "myNewEnv-1", env1.Name)

	//Check keys
	for _, k := range env.Keys {
		var keyFound bool
		for _, kk := range env1.Keys {
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
	for _, v := range env1.Variable {
		switch v.Name {
		case "myPassword":
			assert.True(t, v.Type == sdk.SecretVariable, "myPassword.type should be type password")
			assert.True(t, v.Value == "MySecretValue", "myPassword.value is wrong")
		default:
			t.Errorf("Unexpected variable %+v", v)
		}
	}

}

func Test_postEnvironmentImportHandler_NewEnvFromYAMLWithKeysAndSecretsAndReImport(t *testing.T) {
	api, db, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	env := &sdk.Environment{
		Name:      "myNewEnv",
		ProjectID: proj.ID,
	}
	test.NoError(t, environment.InsertEnvironment(db, env))

	k := &sdk.EnvironmentKey{
		Key: sdk.Key{
			Name: "mykey",
			Type: "pgp",
		},
		EnvironmentID: env.ID,
	}

	kpgp, err := keys.GeneratePGPKeyPair(k.Name)
	test.NoError(t, err)
	k.Public = kpgp.Public
	k.Private = kpgp.Private
	k.KeyID = kpgp.KeyID
	if err := environment.InsertKey(api.mustDB(), k); err != nil {
		t.Fatal(err)
	}

	test.NoError(t, environment.InsertVariable(db, env.ID, &sdk.Variable{
		Name:  "myPassword",
		Type:  sdk.SecretVariable,
		Value: "MySecretValue",
	}, u))

	//Export all the things
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
	body := rec.Body.String()
	t.Logf(">>%s", body)

	//Change the name of the environment
	body = strings.Replace(body, env.Name, "myNewEnv-1", 1)

	//Import the new environment
	vars = map[string]string{
		"permProjectKey": proj.Key,
	}
	uri = api.Router.GetRoute("POST", api.postEnvironmentImportHandler, vars)
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

	env, err = environment.LoadEnvironmentByName(db, proj.Key, "myNewEnv")
	test.NoError(t, err)
	// reload variables with clear password
	variables, errLoadVars := environment.GetAllVariable(db, proj.Key, "myNewEnv", environment.WithClearPassword())
	test.NoError(t, errLoadVars)
	env.Variable = variables
	test.NoError(t, environment.LoadAllDecryptedKeys(db, env))

	env1, err := environment.LoadEnvironmentByName(db, proj.Key, "myNewEnv-1")
	test.NoError(t, err)
	// reload variables with clear password
	variables1, errLoadVariables := environment.GetAllVariable(db, proj.Key, "myNewEnv-1", environment.WithClearPassword())
	test.NoError(t, errLoadVariables)
	env1.Variable = variables1
	test.NoError(t, environment.LoadAllDecryptedKeys(db, env1))

	assert.NotNil(t, env1)
	assert.Equal(t, "myNewEnv-1", env1.Name)

	//Check keys
	for _, k := range env.Keys {
		var keyFound bool
		for _, kk := range env1.Keys {
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
	for _, v := range env1.Variable {
		switch v.Name {
		case "myPassword":
			assert.True(t, v.Type == sdk.SecretVariable, "myPassword.type should be type password")
			assert.True(t, v.Value == "MySecretValue", "myPassword.value is wrong")
		default:
			t.Errorf("Unexpected variable %+v", v)
		}
	}

	//Reimport
	//Import the new environment
	vars = map[string]string{
		"permProjectKey": proj.Key,
	}
	uri = api.Router.GetRoute("POST", api.postEnvironmentImportHandler, vars)
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

	env, err = environment.LoadEnvironmentByName(db, proj.Key, "myNewEnv")
	test.NoError(t, err)
	// reload variables with clear password
	variables, errLoadVars = environment.GetAllVariable(db, proj.Key, "myNewEnv", environment.WithClearPassword())
	test.NoError(t, errLoadVars)
	env.Variable = variables
	test.NoError(t, environment.LoadAllDecryptedKeys(db, env))

	env1, err = environment.LoadEnvironmentByName(db, proj.Key, "myNewEnv-1")
	test.NoError(t, err)
	// reload variables with clear password
	variables1, errLoadVariables = environment.GetAllVariable(db, proj.Key, "myNewEnv-1", environment.WithClearPassword())
	test.NoError(t, errLoadVariables)
	env1.Variable = variables1
	test.NoError(t, environment.LoadAllDecryptedKeys(db, env1))

	assert.NotNil(t, env1)
	assert.Equal(t, "myNewEnv-1", env1.Name)

	//Check keys
	for _, k := range env.Keys {
		var keyFound bool
		for _, kk := range env1.Keys {
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
	for _, v := range env1.Variable {
		switch v.Name {
		case "myPassword":
			assert.True(t, v.Type == sdk.SecretVariable, "myPassword.type should be type password")
			assert.True(t, v.Value == "MySecretValue", "myPassword.value is wrong")
		default:
			t.Errorf("Unexpected variable %+v", v)
		}
	}
}

func Test_postEnvironmentImportHandler_NewEnvFromYAMLWithEmptyKey(t *testing.T) {
	api, db, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := api.Router.GetRoute("POST", api.postEnvironmentImportHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, nil)

	body := `version: v1.0
name: myNewEnv
keys:
  myPGPkey:
    type: pgp
  mySSHKey:
    type: ssh`
	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())

	env, err := environment.LoadEnvironmentByName(db, proj.Key, "myNewEnv")
	test.NoError(t, err)
	// reload variables with clear password
	variables, errLoadVars := environment.GetAllVariable(db, proj.Key, "myNewEnv", environment.WithClearPassword())
	test.NoError(t, errLoadVars)
	env.Variable = variables
	test.NoError(t, environment.LoadAllDecryptedKeys(db, env))

	assert.NotNil(t, env)
	assert.Equal(t, "myNewEnv", env.Name)

	var myPGPkey, mySSHKey bool
	for _, k := range env.Keys {
		switch k.Name {
		case "myPGPkey":
			myPGPkey = true
			assert.NotEmpty(t, k.KeyID)
			assert.NotEmpty(t, k.Private)
			assert.NotEmpty(t, k.Public)
			assert.NotEmpty(t, k.Type)
		case "mySSHKey":
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

func Test_postEnvironmentImportHandler_ExistingAppFromYAMLWithoutForce(t *testing.T) {
	api, db, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	env := sdk.Environment{
		Name:      "myNewEnv",
		ProjectID: proj.ID,
	}
	test.NoError(t, environment.InsertEnvironment(db, &env))

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := api.Router.GetRoute("POST", api.postEnvironmentImportHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, nil)

	body := `version: v1.0
name: myNewEnv`
	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 409, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())
}

func Test_postEnvironmentImportHandler_ExistingAppFromYAMLInheritPermissions(t *testing.T) {
	api, db, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	env := sdk.Environment{
		Name:      "myNewEnv",
		ProjectID: proj.ID,
	}
	test.NoError(t, environment.InsertEnvironment(db, &env))

	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
	}
	uri := api.Router.GetRoute("POST", api.postEnvironmentImportHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri+"?force=true", nil)

	body := `version: v1.0
name: myNewEnv`
	req.Body = ioutil.NopCloser(strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())
}
