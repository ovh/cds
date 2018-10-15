package api

import (
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func Test_loginUserHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)
	u, p := assets.InsertAdminUser(db)

	//Prepare request
	uri := api.Router.GetRoute("POST", api.loginUserHandler, nil)
	test.NotEmpty(t, uri)
	data := sdk.UserLoginRequest{
		Username: u.Username,
		Password: p,
	}

	req := assets.NewRequest(t, "POST", uri, data)

	//Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	t.Logf("response: %s", w.Body.Bytes())

	//Now let's call /user/me
	//TOOD

}
