package api

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/stretchr/testify/assert"
)

func Test_getIntegrationModelsHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)

	vars := map[string]string{}

	uri := router.GetRoute("GET", api.getIntegrationModelsHandler, vars)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}

func Test_postIntegrationModelHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)

	vars := map[string]string{}
	model := sdk.IntegrationModel{
		Name: "my-model",
	}

	uri := router.GetRoute("POST", api.postIntegrationModelHandler, vars)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)

	model, _ = integration.LoadModelByName(context.TODO(), db, model.Name)
	test.NoError(t, integration.DeleteModel(context.TODO(), db, model.ID))
}

func Test_putIntegrationModelHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)

	model := sdk.IntegrationModel{
		Name: "my-model",
	}

	test.NoError(t, integration.InsertModel(db, &model))

	vars := map[string]string{
		"name": model.Name,
	}

	uri := router.GetRoute("PUT", api.putIntegrationModelHandler, vars)
	req := assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, model)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	test.NoError(t, integration.DeleteModel(context.TODO(), db, model.ID))
}

func Test_deleteIntegrationModelHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)

	model := sdk.IntegrationModel{
		Name: "my-model",
	}

	test.NoError(t, integration.InsertModel(db, &model))

	vars := map[string]string{
		"name": model.Name,
	}

	uri := router.GetRoute("DELETE", api.deleteIntegrationModelHandler, vars)
	req := assets.NewAuthentifiedRequest(t, u, pass, "DELETE", uri, model)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 204, w.Code)
}
