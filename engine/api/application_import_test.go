package api

import (
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
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
	assert.Equal(t, 201, rec.Code)

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

func Test_postApplicationImportHandler_NewAppFromYAMLWithSecrets(t *testing.T) {

}

func Test_postApplicationImportHandler_ExistingAppFromYAMLWithoutForce(t *testing.T) {

}

func Test_postApplicationImportHandler_ExistingAppFromYAMLWithPermissions(t *testing.T) {

}

func Test_postApplicationImportHandler_ExistingAppFromYAMLInheritPermissions(t *testing.T) {

}
