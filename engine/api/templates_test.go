package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

const (
	cdsGoBuildAction = "https://raw.githubusercontent.com/ovh/cds/0.8.1/contrib/actions/cds-go-build.hcl"
)

func Test_getTemplatesHandler(t *testing.T) {
	api, _, router := newTestAPI(t, bootstrap.InitiliazeDB)

	if router.Mux == nil {
		t.Fatal("Router cannot be nil")
		return
	}

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	//Prepare request
	vars := map[string]string{}
	uri := router.GetRoute("GET", api.getTemplatesHandler, vars)
	test.NotEmpty(t, uri)

	req, _ := http.NewRequest("GET", uri, nil)
	assets.AuthentifyRequest(t, req, u, pass)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())
}

func downloadPublicAction(t *testing.T, u *sdk.User, pass string, api *API) {
	//Load the gitclone public action
	//Prepare request
	uri := api.Router.GetRoute("POST", api.importActionHandler, nil)
	test.NotEmpty(t, uri)

	req, _ := http.NewRequest("POST", uri, nil)
	req.Form = url.Values{}
	req.Form.Add("url", cdsGoBuildAction)
	assets.AuthentifyRequest(t, req, u, pass)

	//Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.True(t, w.Code >= 200)
	assert.True(t, w.Code < 400)
}
