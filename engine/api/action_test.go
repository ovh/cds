package api

import (
	"bytes"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	yaml "gopkg.in/yaml.v2"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/stretchr/testify/assert"
)

func Test_getActionExportHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(db)

	//Prepare request
	vars := map[string]string{
		"permActionName": "GitClone",
	}
	uri := api.Router.GetRoute("GET", api.getActionExportHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Check result
	t.Logf(">>%s", rec.Body.String())
}

func Test_postActionImportHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)
	u, pass := assets.InsertAdminUser(db)

	uri := api.Router.GetRoute("POST", api.importActionHandler, nil)
	test.NotEmpty(t, uri)

	a := exportentities.Action{
		Name:        "myAction",
		Description: "MyDecription",
		Requirements: []exportentities.Requirement{
			{
				Binary: "bash",
			},
		},
		Parameters: map[string]exportentities.ParameterValue{
			"param1": exportentities.ParameterValue{
				Description:  "this is my param",
				DefaultValue: "default value",
			},
		},
		Steps: []exportentities.Step{
			map[string]interface{}{
				"script": "echo {{.cds.pip.param1}}",
			},
		},
	}
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, nil)

	body, _ := yaml.Marshal(a)
	req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/x-yaml")

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	if rec.Code > 201 {
		t.Errorf("http code status %d", rec.Code)
	}

	//Check result
	t.Logf(">>%s", rec.Body.String())
}
