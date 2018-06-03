package api

import (
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/platform"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/stretchr/testify/assert"
)

func Test_getPlatformModelsHandler(t *testing.T) {
	api, _, router := newTestAPI(t, bootstrap.InitiliazeDB)
	u, pass := assets.InsertAdminUser(api.mustDB())

	vars := map[string]string{}

	uri := router.GetRoute("GET", api.getPlatformModelsHandler, vars)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func Test_postPlatformModelHandler(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)
	u, pass := assets.InsertAdminUser(api.mustDB())

	vars := map[string]string{}
	model := sdk.PlatformModel{
		Name: "my-model",
	}

	uri := router.GetRoute("POST", api.postPlatformModelHandler, vars)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)

	model, _ = platform.LoadModelByName(db, model.Name, false)
	test.NoError(t, platform.DeleteModel(db, model.ID))
}

func Test_putPlatformModelHandler(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)
	u, pass := assets.InsertAdminUser(api.mustDB())

	model := sdk.PlatformModel{
		Name: "my-model",
	}

	test.NoError(t, platform.InsertModel(db, &model))

	vars := map[string]string{
		"name": model.Name,
	}

	uri := router.GetRoute("PUT", api.putPlatformModelHandler, vars)
	req := assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, model)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	test.NoError(t, platform.DeleteModel(db, model.ID))
}

func Test_deletePlatformModelHandler(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)
	u, pass := assets.InsertAdminUser(api.mustDB())

	model := sdk.PlatformModel{
		Name: "my-model",
	}

	platform.InsertModel(db, &model)

	vars := map[string]string{
		"name": model.Name,
	}

	uri := router.GetRoute("DELETE", api.deletePlatformModelHandler, vars)
	req := assets.NewAuthentifiedRequest(t, u, pass, "DELETE", uri, model)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 204, w.Code)
}
