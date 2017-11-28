package api

import (
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

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
}

func Test_postApplicationImportHandler_NewAppFromYAMLWithSecrets(t *testing.T) {

}

func Test_postApplicationImportHandler_ExistingAppFromYAMLWithoutForce(t *testing.T) {

}

func Test_postApplicationImportHandler_ExistingAppFromYAMLWithPermissions(t *testing.T) {

}

func Test_postApplicationImportHandler_ExistingAppFromYAMLInheritPermissions(t *testing.T) {

}
