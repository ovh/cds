package api

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
)

func Test_DeleteAllWorkerModel(t *testing.T) {
	api, _, _ := newTestAPI(t, bootstrap.InitiliazeDB)

	//Loading all models
	models, err := worker.LoadWorkerModels(api.mustDB())
	if err != nil {
		t.Fatalf("Error getting models : %s", err)
	}

	//Delete all of them
	for _, m := range models {
		if err := worker.DeleteWorkerModel(api.mustDB(), m.ID); err != nil {
			t.Fatalf("Error deleting model : %s", err)
		}
	}

}

func Test_addWorkerModelAsAdmin(t *testing.T) {
	api, _, _ := newTestAPI(t, bootstrap.InitiliazeDB)

	//Loading all models
	models, errlw := worker.LoadWorkerModels(api.mustDB())
	if errlw != nil {
		t.Fatalf("Error getting models : %s", errlw)
	}

	//Delete all of them
	for _, m := range models {
		if err := worker.DeleteWorkerModel(api.mustDB(), m.ID); err != nil {
			t.Fatalf("Error deleting model : %s", err)
		}
	}

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())
	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	g, err := group.LoadGroup(api.mustDB(), "shared.infra")
	if err != nil {
		t.Fatalf("Error getting group : %s", err)
	}

	model := sdk.Model{
		Name:    "Test1",
		GroupID: g.ID,
		Type:    sdk.Docker,
		Image:   "buildpack-deps:jessie",
		Capabilities: []sdk.Requirement{
			{
				Name:  "capa1",
				Type:  sdk.BinaryRequirement,
				Value: "1",
			},
		},
	}

	//Prepare request
	uri := api.Router.GetRoute("POST", api.addWorkerModelHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())
}

func Test_addWorkerModelWithWrongRequest(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	api, _, router := newTestAPI(t, bootstrap.InitiliazeDB)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())
	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	g, err := group.LoadGroup(api.mustDB(), "shared.infra")
	if err != nil {
		t.Fatalf("Error getting group : %s", err)
	}

	//Type is mandatory
	model := sdk.Model{
		Name:    "Test1",
		Image:   "buildpack-deps:jessie",
		GroupID: g.ID,
		Capabilities: []sdk.Requirement{
			{
				Name:  "capa1",
				Type:  sdk.BinaryRequirement,
				Value: "1",
			},
		},
	}

	//Prepare request
	uri := api.Router.GetRoute("POST", api.addWorkerModelHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)

	t.Logf("Body: %s", w.Body.String())

	//Name is mandatory
	model = sdk.Model{
		GroupID: g.ID,
		Type:    sdk.Docker,
		Capabilities: []sdk.Requirement{
			{
				Name:  "capa1",
				Type:  sdk.BinaryRequirement,
				Value: "1",
			},
		},
	}

	//Prepare request
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)

	t.Logf("Body: %s", w.Body.String())

	//GroupID is mandatory
	model = sdk.Model{
		Name:  "Test1",
		Type:  sdk.Docker,
		Image: "buildpack-deps:jessie",
		Capabilities: []sdk.Requirement{
			{
				Name:  "capa1",
				Type:  sdk.BinaryRequirement,
				Value: "1",
			},
		},
	}

	//Prepare request
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)

	t.Logf("Body: %s", w.Body.String())

	//SendBadRequest

	//Prepare request
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, "blabla")

	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)

	t.Logf("Body: %s", w.Body.String())
}

func Test_addWorkerModelAsAGroupMember(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	api, _, router := newTestAPI(t, bootstrap.InitiliazeDB)

	//Create group
	g := &sdk.Group{
		Name: sdk.RandomString(10),
	}

	//Create user
	u, pass := assets.InsertLambdaUser(api.mustDB(), g)
	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	model := sdk.Model{
		Name:    "Test1",
		GroupID: g.ID,
		Type:    sdk.Docker,
		Image:   "buildpack-deps:jessie",
		Capabilities: []sdk.Requirement{
			{
				Name:  "capa1",
				Type:  sdk.BinaryRequirement,
				Value: "1",
			},
		},
	}

	//Prepare request
	uri := router.GetRoute("POST", api.addWorkerModelHandler, nil)
	test.NotEmpty(t, uri, "Route route found")

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 403, w.Code, "Status code should be 403")

	t.Logf("Body: %s", w.Body.String())
}

func Test_addWorkerModelAsAGroupAdmin(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	api, _, router := newTestAPI(t, bootstrap.InitiliazeDB)

	//Create group
	g := &sdk.Group{
		Name: sdk.RandomString(10),
	}

	//Create user
	u, pass := assets.InsertLambdaUser(api.mustDB(), g)
	assert.NotZero(t, u)
	assert.NotZero(t, pass)
	test.NoError(t, group.SetUserGroupAdmin(api.mustDB(), g.ID, u.ID))

	model := sdk.Model{
		Name:    "Test1",
		GroupID: g.ID,
		Type:    sdk.Docker,
		Image:   "buildpack-deps:jessie",
		Capabilities: []sdk.Requirement{
			{
				Name:  "capa1",
				Type:  sdk.BinaryRequirement,
				Value: "1",
			},
		},
	}

	//Prepare request
	uri := router.GetRoute("POST", api.addWorkerModelHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code, "Status code should equal 200")

	t.Logf("Body: %s", w.Body.String())
}

func Test_addWorkerModelAsAWrongGroupMember(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	api, _, router := newTestAPI(t, bootstrap.InitiliazeDB)

	//Create group
	g := &sdk.Group{
		Name: sdk.RandomString(10),
	}

	//Create group
	g1 := &sdk.Group{
		Name: sdk.RandomString(10),
	}

	if err := group.InsertGroup(api.mustDB(), g1); err != nil {
		t.Fatal(err)
	}

	//Create user
	u, pass := assets.InsertLambdaUser(api.mustDB(), g)
	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	if err := group.SetUserGroupAdmin(api.mustDB(), g.ID, u.ID); err != nil {
		t.Fatal(err)
	}

	model := sdk.Model{
		Name:    "Test1",
		GroupID: g1.ID,
		Type:    sdk.Docker,
		Image:   "buildpack-deps:jessie",
		Capabilities: []sdk.Requirement{
			{
				Name:  "capa1",
				Type:  sdk.BinaryRequirement,
				Value: "1",
			},
		},
	}

	//Prepare request
	uri := router.GetRoute("POST", api.addWorkerModelHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 403, w.Code)

	t.Logf("Body: %s", w.Body.String())
}

func Test_updateWorkerModel(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	api, _, router := newTestAPI(t, bootstrap.InitiliazeDB)

	//Create group
	g := &sdk.Group{
		Name: sdk.RandomString(10),
	}

	//Create user
	u, pass := assets.InsertLambdaUser(api.mustDB(), g)
	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	if err := group.SetUserGroupAdmin(api.mustDB(), g.ID, u.ID); err != nil {
		t.Fatal(err)
	}

	model := sdk.Model{
		Name:    "Test1",
		GroupID: g.ID,
		Type:    sdk.Docker,
		Image:   "buildpack-deps:jessie",
		Capabilities: []sdk.Requirement{
			{
				Name:  "capa1",
				Type:  sdk.BinaryRequirement,
				Value: "1",
			},
		},
	}

	//Prepare request
	uri := router.GetRoute("POST", api.addWorkerModelHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())

	json.Unmarshal(w.Body.Bytes(), &model)

	model2 := sdk.Model{
		Name: "Test1bis",
		Capabilities: []sdk.Requirement{
			{
				Name:  "capa1",
				Type:  sdk.BinaryRequirement,
				Value: "1",
			},
			{
				Name:  "capa2",
				Type:  sdk.BinaryRequirement,
				Value: "2",
			},
		},
	}

	//Prepare request
	vars := map[string]string{
		"permModelID": fmt.Sprintf("%d", model.ID),
	}
	uri = router.GetRoute("PUT", api.updateWorkerModelHandler, vars)
	test.NotEmpty(t, uri)

	req = assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, model2)

	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())

}

func Test_deleteWorkerModel(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	api, _, router := newTestAPI(t, bootstrap.InitiliazeDB)

	//Create group
	g := &sdk.Group{
		Name: sdk.RandomString(10),
	}

	//Create user
	u, pass := assets.InsertLambdaUser(api.mustDB(), g)
	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	if err := group.SetUserGroupAdmin(api.mustDB(), g.ID, u.ID); err != nil {
		t.Fatal(err)
	}

	model := sdk.Model{
		Name:    "Test1",
		GroupID: g.ID,
		Type:    sdk.Docker,
		Image:   "buildpack-deps:jessie",
		Capabilities: []sdk.Requirement{
			{
				Name:  "capa1",
				Type:  sdk.BinaryRequirement,
				Value: "1",
			},
		},
	}

	//Prepare request
	uri := router.GetRoute("POST", api.addWorkerModelHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())

	json.Unmarshal(w.Body.Bytes(), &model)

	//Prepare request
	vars := map[string]string{
		"permModelID": fmt.Sprintf("%d", model.ID),
	}
	uri = router.GetRoute("DELETE", api.deleteWorkerModelHandler, vars)
	test.NotEmpty(t, uri)

	req = assets.NewAuthentifiedRequest(t, u, pass, "DELETE", uri, nil)

	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 204, w.Code)

	t.Logf("Body: %s", w.Body.String())

}

func Test_getWorkerModel(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	api, _, router := newTestAPI(t, bootstrap.InitiliazeDB)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())
	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	g, err := group.LoadGroup(api.mustDB(), "shared.infra")
	if err != nil {
		t.Fatalf("Error getting group : %s", err)
	}

	model := sdk.Model{
		Name:    "Test1",
		GroupID: g.ID,
		Type:    sdk.Docker,
		Image:   "buildpack-deps:jessie",
		Capabilities: []sdk.Requirement{
			{
				Name:  "capa1",
				Type:  sdk.BinaryRequirement,
				Value: "1",
			},
		},
	}

	//Prepare request
	uri := router.GetRoute("POST", api.addWorkerModelHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())

	//Prepare request
	uri = router.GetRoute("GET", api.getWorkerModelsHandler, nil)
	test.NotEmpty(t, uri)

	req = assets.NewAuthentifiedRequest(t, u, pass, "GET", uri+"?name=Test1", nil)

	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())

}

func Test_getWorkerModels(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	api, _, router := newTestAPI(t, bootstrap.InitiliazeDB)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())
	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	g, err := group.LoadGroup(api.mustDB(), "shared.infra")
	if err != nil {
		t.Fatalf("Error getting group : %s", err)
	}

	model := sdk.Model{
		Name:    "Test1",
		GroupID: g.ID,
		Type:    sdk.Docker,
		Image:   "buildpack-deps:jessie",
		Capabilities: []sdk.Requirement{
			{
				Name:  "capa1",
				Type:  sdk.BinaryRequirement,
				Value: "1",
			},
		},
	}

	//Prepare request
	uri := router.GetRoute("POST", api.addWorkerModelHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())

	//Prepare request
	uri = router.GetRoute("GET", api.getWorkerModelsHandler, nil)
	test.NotEmpty(t, uri)

	req = assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())

	results := []sdk.Model{}
	json.Unmarshal(w.Body.Bytes(), &results)

	assert.Equal(t, 1, len(results))
	assert.Equal(t, "Test1", results[0].Name)
	assert.Equal(t, 1, len(results[0].Capabilities))
	assert.Equal(t, u.Fullname, results[0].CreatedBy.Fullname)
}

func Test_addWorkerModelCapa(t *testing.T) {

}

func Test_getWorkerModelTypes(t *testing.T) {

}

func Test_getWorkerModelCapaTypes(t *testing.T) {

}

func Test_updateWorkerModelCapa(t *testing.T) {

}

func Test_deleteWorkerModelCapa(t *testing.T) {

}
