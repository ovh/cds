package api

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func Test_DeleteAllWorkerModel(t *testing.T) {
	api, _, _, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()

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

	modelPatterns, err := worker.LoadWorkerModelPatterns(api.mustDB())
	test.NoError(t, err)

	for _, wmp := range modelPatterns {
		test.NoError(t, worker.DeleteWorkerModelPattern(api.mustDB(), wmp.ID))
	}
}

func Test_postWorkerModelAsAdmin(t *testing.T) {
	api, _, _, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()

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
		ModelDocker: sdk.ModelDocker{
			Image: "buildpack-deps:jessie",
			Shell: "sh -c",
			Cmd:   "worker --api={{.API}}",
			Envs: map[string]string{
				"CDS_TEST": "THIS IS A TEST",
			},
		},
	}

	//Prepare request
	uri := api.Router.GetRoute("POST", api.postWorkerModelHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var newModel sdk.Model
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &newModel))

	test.Equal(t, "worker --api={{.API}}", newModel.ModelDocker.Cmd, "Main worker command is not good")
	test.Equal(t, "THIS IS A TEST", newModel.ModelDocker.Envs["CDS_TEST"], "Worker model envs are not good")
}

func Test_WorkerModelUsage(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	api, db, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()

	u, pass := assets.InsertAdminUser(db)
	assert.NotZero(t, u)

	grName := sdk.RandomString(10)
	gr := assets.InsertTestGroup(t, db, grName)
	test.NotNil(t, gr)

	model := sdk.Model{
		Name:    "Test1" + grName,
		GroupID: gr.ID,
		Type:    sdk.Docker,
		ModelDocker: sdk.ModelDocker{
			Image: "buildpack-deps:jessie",
			Shell: "sh -c",
			Cmd:   "worker --api={{.API}}",
			Envs: map[string]string{
				"CDS_TEST": "THIS IS A TEST",
			},
		},
	}
	test.NoError(t, worker.InsertWorkerModel(db, &model))

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)
	test.NoError(t, group.InsertUserInGroup(db, proj.ProjectGroups[0].Group.ID, u.ID, true))

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}

	test.NoError(t, pipeline.InsertPipeline(db, api.Cache, proj, &pip, u))

	//Insert Stage
	stage := &sdk.Stage{
		Name:          "stage_Test_0",
		PipelineID:    pip.ID,
		BuildOrder:    1,
		Enabled:       true,
		Prerequisites: []sdk.Prerequisite{},
	}
	pip.Stages = append(pip.Stages, *stage)

	t.Logf("Insert Stage %s for Pipeline %s of Project %s", stage.Name, pip.Name, proj.Name)
	test.NoError(t, pipeline.InsertStage(db, stage))

	//Insert Action
	t.Logf("Insert Action script on Stage %s for Pipeline %s of Project %s", stage.Name, pip.Name, proj.Name)

	job := &sdk.Job{
		Action: sdk.Action{
			Name:    "NewAction",
			Enabled: true,
			Requirements: []sdk.Requirement{
				{
					Name:  "Test1" + grName,
					Type:  sdk.ModelRequirement,
					Value: "Test1" + grName,
				},
			},
		},
		Enabled: true,
	}
	errJob := pipeline.InsertJob(db, job, stage.ID, &pip)
	test.NoError(t, errJob)
	assert.NotZero(t, job.PipelineActionID)
	assert.NotZero(t, job.Action.ID)

	proj, _ = project.LoadByID(db, api.Cache, proj.ID, u, project.LoadOptions.WithApplications, project.LoadOptions.WithPipelines, project.LoadOptions.WithEnvironments, project.LoadOptions.WithGroups)

	wf := sdk.Workflow{
		Name:       "workflow1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}
	(&wf).RetroMigrate()

	test.NoError(t, workflow.Insert(db, api.Cache, &wf, proj, u))

	//Prepare request
	vars := map[string]string{
		"groupName":     gr.Name,
		"permModelName": model.Name,
	}
	uri := router.GetRoute("GET", api.getWorkerModelUsageHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, vars)
	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var pipelines []sdk.Pipeline
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &pipelines))

	test.NotNil(t, pipelines)
	test.Equal(t, 1, len(pipelines))
	test.Equal(t, "pip1", pipelines[0].Name)
	test.Equal(t, proj.Key, pipelines[0].ProjectKey)
}

func Test_postWorkerModelWithWrongRequest(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	api, _, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()

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
		Name: "Test1",
		ModelDocker: sdk.ModelDocker{
			Image: "buildpack-deps:jessie",
			Cmd:   "worker",
		},
		GroupID: g.ID,
	}

	//Prepare request
	uri := api.Router.GetRoute("POST", api.postWorkerModelHandler, nil)
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
		Name: "Test1",
		Type: sdk.Docker,
		ModelDocker: sdk.ModelDocker{
			Image: "buildpack-deps:jessie",
			Cmd:   "worker",
		},
	}

	//Prepare request
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)

	t.Logf("Body: %s", w.Body.String())

	//Cmd is mandatory
	model = sdk.Model{
		Name: "Test1",
		Type: sdk.Docker,
		ModelDocker: sdk.ModelDocker{
			Image: "buildpack-deps:jessie",
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

func Test_postWorkerModelAsAGroupMember(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	api, _, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()

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
		ModelDocker: sdk.ModelDocker{
			Image: "buildpack-deps:jessie",
			Cmd:   "worker",
			Shell: "sh",
		},
	}

	//Prepare request
	uri := router.GetRoute("POST", api.postWorkerModelHandler, nil)
	test.NotEmpty(t, uri, "Route route found")

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 403, w.Code, "Status code should be 403 because only a group admin can create a model")

	t.Logf("Body: %s", w.Body.String())
}

func Test_postWorkerModelAsAGroupAdmin(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	api, _, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()

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
		ModelDocker: sdk.ModelDocker{
			Image: "buildpack-deps:jessie",
			Cmd:   "worker",
			Shell: "sh",
		},
	}

	//Prepare request
	uri := router.GetRoute("POST", api.postWorkerModelHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 403, w.Code, "Status code should equal 403 because the worker model haven't pattern and is not restricted")

	t.Logf("Body: %s", w.Body.String())
}

func Test_postWorkerModelAsAGroupAdminWithRestrict(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	api, _, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()

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
		Name:       "Test1",
		GroupID:    g.ID,
		Type:       sdk.Docker,
		Restricted: true,
		ModelDocker: sdk.ModelDocker{
			Image: "buildpack-deps:jessie",
			Shell: "sh -c",
			Cmd:   "worker --api={{.API}}",
		},
	}

	//Prepare request
	uri := router.GetRoute("POST", api.postWorkerModelHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code, "Status code should equal 200")

	var newModel sdk.Model
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &newModel))

	test.Equal(t, "worker --api={{.API}}", newModel.ModelDocker.Cmd, "Main worker command is not good")
}

func Test_postWorkerModelAsAGroupAdminWithoutRestrictWithPattern(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	api, _, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()

	//Create group
	g := &sdk.Group{
		Name: sdk.RandomString(10),
	}

	//Create user
	u, pass := assets.InsertLambdaUser(api.mustDB(), g)
	assert.NotZero(t, u)
	assert.NotZero(t, pass)
	test.NoError(t, group.SetUserGroupAdmin(api.mustDB(), g.ID, u.ID))

	pattern := sdk.ModelPattern{
		Name: "test",
		Type: sdk.Openstack,
		Model: sdk.ModelCmds{
			PreCmd: "apt-get install curl -y",
			Cmd:    "./worker",
		},
	}

	test.NoError(t, worker.InsertWorkerModelPattern(api.mustDB(), &pattern))

	model := sdk.Model{
		Name:        "Test1",
		GroupID:     g.ID,
		Type:        sdk.Openstack,
		PatternName: "test",
		ModelVirtualMachine: sdk.ModelVirtualMachine{
			Image:  "Debian 7",
			Flavor: "vps-ssd-1",
			Cmd:    "worker --api={{.API}}",
		},
	}

	//Prepare request
	uri := router.GetRoute("POST", api.postWorkerModelHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code, "Status code should equal 200")

	var newModel sdk.Model
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &newModel))

	test.Equal(t, "./worker", newModel.ModelVirtualMachine.Cmd, "Main worker command is not good")
	test.Equal(t, "apt-get install curl -y", newModel.ModelVirtualMachine.PreCmd, "Pre worker command is not good")
}

// Test_postWorkerModelAsAGroupAdminWithProvision test the provioning
// For a group Admin, it is allowed to set a provision only for restricted model
func Test_postWorkerModelAsAGroupAdminWithProvision(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	api, _, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()

	//Create group
	g := &sdk.Group{Name: sdk.RandomString(10)}

	//Create user
	u, pass := assets.InsertLambdaUser(api.mustDB(), g)
	assert.NotZero(t, u)
	assert.NotZero(t, pass)
	test.NoError(t, group.SetUserGroupAdmin(api.mustDB(), g.ID, u.ID))

	model := sdk.Model{
		Name:       "Test-with-provision",
		GroupID:    g.ID,
		Type:       sdk.Docker,
		Restricted: true,
		Provision:  1,
		ModelDocker: sdk.ModelDocker{
			Image: "buildpack-deps:jessie",
			Shell: "sh -c",
			Cmd:   "worker",
		},
	}

	//Prepare request
	uri := router.GetRoute("POST", api.postWorkerModelHandler, nil)
	test.NotEmpty(t, uri)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, model))

	assert.Equal(t, 200, w.Code, "Status code should equal 200")

	t.Logf("Body: %s", w.Body.String())

	var wm sdk.Model
	json.Unmarshal(w.Body.Bytes(), &wm)
	assert.Equal(t, 1, int(wm.Provision))

	// update restricted flag -> provioning will be reset

	pattern := sdk.ModelPattern{
		Name: "test",
		Type: sdk.Docker,
		Model: sdk.ModelCmds{
			Cmd:   "./worker",
			Shell: "sh -c",
		},
	}

	test.NoError(t, worker.InsertWorkerModelPattern(api.mustDB(), &pattern))

	vars := map[string]string{
		"groupName":     g.Name,
		"permModelName": model.Name,
	}
	uri = router.GetRoute("PUT", api.putWorkerModelHandler, vars)
	test.NotEmpty(t, uri)

	// API will set provisioning to 0 for a non-restricted model
	wm.Restricted = false
	wm.PatternName = "test"
	req := assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, wm)

	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var wmUpdated sdk.Model
	json.Unmarshal(w.Body.Bytes(), &wmUpdated)
	assert.Equal(t, 0, int(wmUpdated.Provision))
	assert.Equal(t, "./worker", wmUpdated.ModelDocker.Cmd)
}

func Test_postWorkerModelAsAWrongGroupMember(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	api, _, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()

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
		ModelDocker: sdk.ModelDocker{
			Image: "buildpack-deps:jessie",
			Cmd:   "worker",
			Shell: "sh",
		},
	}

	//Prepare request
	uri := router.GetRoute("POST", api.postWorkerModelHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 403, w.Code, "Status code should be 403 because only a group admin can create a model")

	t.Logf("Body: %s", w.Body.String())
}

func Test_putWorkerModel(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	api, _, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()

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
		Name:       "Test1",
		GroupID:    g.ID,
		Type:       sdk.Docker,
		Restricted: true,
		ModelDocker: sdk.ModelDocker{
			Image: "buildpack-deps:jessie",
			Shell: "sh -c",
			Cmd:   "worker",
		},
	}

	//Prepare request
	uri := router.GetRoute("POST", api.postWorkerModelHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())

	json.Unmarshal(w.Body.Bytes(), &model)

	model2 := sdk.Model{
		Name:       "Test1bis",
		GroupID:    g.ID,
		Type:       sdk.Docker,
		Restricted: true,
		ModelDocker: sdk.ModelDocker{
			Image: "buildpack-deps:jessie",
			Cmd:   "worker",
			Shell: "sh -c",
		},
	}

	//Prepare request
	vars := map[string]string{
		"groupName":     g.Name,
		"permModelName": model.Name,
	}
	uri = router.GetRoute("PUT", api.putWorkerModelHandler, vars)
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
	api, _, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()

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
		Name:       "Test1",
		GroupID:    g.ID,
		Type:       sdk.Docker,
		Restricted: true,
		ModelDocker: sdk.ModelDocker{
			Image: "buildpack-deps:jessie",
			Cmd:   "worker",
			Shell: "sh -c",
		},
	}

	//Prepare request
	uri := router.GetRoute("POST", api.postWorkerModelHandler, nil)
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
		"groupName":     g.Name,
		"permModelName": model.Name,
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
	api, _, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()

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
		ModelDocker: sdk.ModelDocker{
			Image: "buildpack-deps:jessie",
			Shell: "sh -c",
			Cmd:   "worker",
		},
	}

	//Prepare request
	uri := router.GetRoute("POST", api.postWorkerModelHandler, nil)
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
	api, _, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()

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
		ModelDocker: sdk.ModelDocker{
			Image: "buildpack-deps:jessie",
			Shell: "sh -c",
			Cmd:   "worker",
		},
	}

	//Prepare request
	uri := router.GetRoute("POST", api.postWorkerModelHandler, nil)
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
	assert.Equal(t, u.Fullname, results[0].CreatedBy.Fullname)
}

// This test create a worker model then an action that will use it.
// Next the model group and name will be updated and we want to check if the requirement was updated.
func Test_renameWorkerModel(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	api, db, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()

	// create new group
	g1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	// create new group
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	// create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())
	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	// prepare post model request
	uri := router.GetRoute("POST", api.postWorkerModelHandler, nil)
	test.NotEmpty(t, uri)

	initialName := sdk.RandomString(10)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, sdk.Model{
		Name:    initialName,
		GroupID: g1.ID,
		Type:    sdk.Docker,
		ModelDocker: sdk.ModelDocker{
			Image: "buildpack-deps:jessie",
			Cmd:   "worker",
			Shell: "sh",
		},
	})

	// send post model request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	// check created model
	assert.Equal(t, 200, w.Code)
	var result sdk.Model
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Equal(t, g1.Name, result.Group.Name)
	assert.Equal(t, initialName, result.Name)

	// prepare post action request
	uri = router.GetRoute("POST", api.postActionHandler, nil)
	test.NotEmpty(t, uri)

	actionName := sdk.RandomString(10)
	modelPath := fmt.Sprintf("%s/%s --privileged", result.Group.Name, result.Name)
	req = assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, sdk.Action{
		Name:    actionName,
		GroupID: &g1.ID,
		Requirements: []sdk.Requirement{{
			Type:  sdk.ModelRequirement,
			Name:  modelPath,
			Value: modelPath,
		}},
	})

	// send post action request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	// check created action
	assert.Equal(t, 201, w.Code)
	var action sdk.Action
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &action))
	assert.Equal(t, g1.Name, action.Group.Name)
	assert.Equal(t, actionName, action.Name)
	assert.Equal(t, 1, len(action.Requirements))
	assert.Equal(t, modelPath, action.Requirements[0].Value)

	// prepare put model request
	uri = router.GetRoute("PUT", api.putWorkerModelHandler, map[string]string{
		"groupName":     result.Group.Name,
		"permModelName": result.Name,
	})
	test.NotEmpty(t, uri)

	newName := sdk.RandomString(10)
	result.Name = newName
	result.GroupID = g2.ID
	req = assets.NewAuthentifiedRequest(t, u, pass, "PUT", uri, result)

	// send put model request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	// check updated model
	assert.Equal(t, 200, w.Code)
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Equal(t, g2.Name, result.Group.Name)
	assert.Equal(t, newName, result.Name)

	// prepare get action request
	uri = router.GetRoute("GET", api.getActionHandler, map[string]string{
		"groupName":      action.Group.Name,
		"permActionName": action.Name,
	})
	test.NotEmpty(t, uri)

	req = assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	// send get action request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	// check action
	updatedModelPath := fmt.Sprintf("%s/%s --privileged", result.Group.Name, result.Name)
	assert.Equal(t, 200, w.Code)
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &action))
	assert.Equal(t, g1.Name, action.Group.Name)
	assert.Equal(t, actionName, action.Name)
	assert.Equal(t, 1, len(action.Requirements))
	assert.Equal(t, updatedModelPath, action.Requirements[0].Value)
}
