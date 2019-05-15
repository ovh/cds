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

func Test_addWorkerModelAsAdmin(t *testing.T) {
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
		RegisteredCapabilities: sdk.RequirementList{
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

	var newModel sdk.Model
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &newModel))

	test.Equal(t, "worker --api={{.API}}", newModel.ModelDocker.Cmd, "Main worker command is not good")
	test.Equal(t, "THIS IS A TEST", newModel.ModelDocker.Envs["CDS_TEST"], "Worker model envs are not good")
}

func Test_addWorkerModelWithPrivateRegistryAsAdmin(t *testing.T) {
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
			Private:  true,
			Username: "test",
			Password: "pwtest",
		},
		RegisteredCapabilities: sdk.RequirementList{
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

	var newModel sdk.Model
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &newModel))

	test.Equal(t, "worker --api={{.API}}", newModel.ModelDocker.Cmd, "Main worker command is not good")
	test.Equal(t, "THIS IS A TEST", newModel.ModelDocker.Envs["CDS_TEST"], "Worker model envs are not good")
	test.Equal(t, sdk.PasswordPlaceholder, newModel.ModelDocker.Password, "Worker model password returned are not placeholder")
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
		RegisteredCapabilities: sdk.RequirementList{
			{
				Name:  "capa1",
				Type:  sdk.BinaryRequirement,
				Value: "1",
			},
		},
	}
	test.NoError(t, worker.InsertWorkerModel(db, &model))

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)
	test.NoError(t, group.InsertUserInGroup(db, proj.ProjectGroups[0].Group.ID, u.OldUserStruct.ID, true))

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}

	test.NoError(t, pipeline.InsertPipeline(db, api.Cache, proj, &pip))

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
		"modelID": fmt.Sprintf("%d", model.ID),
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

func Test_addWorkerModelWithWrongRequest(t *testing.T) {
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
		RegisteredCapabilities: sdk.RequirementList{
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
		RegisteredCapabilities: sdk.RequirementList{
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
		Name: "Test1",
		Type: sdk.Docker,
		ModelDocker: sdk.ModelDocker{
			Image: "buildpack-deps:jessie",
			Cmd:   "worker",
		},
		RegisteredCapabilities: sdk.RequirementList{
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

	//Cmd is mandatory
	model = sdk.Model{
		Name: "Test1",
		Type: sdk.Docker,
		ModelDocker: sdk.ModelDocker{
			Image: "buildpack-deps:jessie",
		},
		RegisteredCapabilities: sdk.RequirementList{
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
		},
		RegisteredCapabilities: sdk.RequirementList{
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
	test.NoError(t, group.SetUserGroupAdmin(api.mustDB(), g.ID, u.OldUserStruct.ID))

	model := sdk.Model{
		Name:    "Test1",
		GroupID: g.ID,
		Type:    sdk.Docker,
		ModelDocker: sdk.ModelDocker{
			Image: "buildpack-deps:jessie",
			Cmd:   "worker",
		},
		RegisteredCapabilities: sdk.RequirementList{
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

	assert.Equal(t, 403, w.Code, "Status code should equal 403 because the worker model haven't pattern and is not restricted")

	t.Logf("Body: %s", w.Body.String())
}

func Test_addWorkerModelAsAGroupAdminWithRestrict(t *testing.T) {
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
	test.NoError(t, group.SetUserGroupAdmin(api.mustDB(), g.ID, u.OldUserStruct.ID))

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
		RegisteredCapabilities: sdk.RequirementList{
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

	var newModel sdk.Model
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &newModel))

	test.Equal(t, "worker --api={{.API}}", newModel.ModelDocker.Cmd, "Main worker command is not good")
}

func Test_addWorkerModelAsAGroupAdminWithoutRestrictWithPattern(t *testing.T) {
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
	test.NoError(t, group.SetUserGroupAdmin(api.mustDB(), g.ID, u.OldUserStruct.ID))

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
		RegisteredCapabilities: sdk.RequirementList{
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

	var newModel sdk.Model
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &newModel))

	test.Equal(t, "./worker", newModel.ModelVirtualMachine.Cmd, "Main worker command is not good")
	test.Equal(t, "apt-get install curl -y", newModel.ModelVirtualMachine.PreCmd, "Pre worker command is not good")
}

// Test_addWorkerModelAsAGroupAdminWithProvision test the provioning
// For a group Admin, it is allowed to set a provision only for restricted model
func Test_addWorkerModelAsAGroupAdminWithProvision(t *testing.T) {
	Test_DeleteAllWorkerModel(t)
	api, _, router, end := newTestAPI(t, bootstrap.InitiliazeDB)
	defer end()

	//Create group
	g := &sdk.Group{Name: sdk.RandomString(10)}

	//Create user
	u, pass := assets.InsertLambdaUser(api.mustDB(), g)
	assert.NotZero(t, u)
	assert.NotZero(t, pass)
	test.NoError(t, group.SetUserGroupAdmin(api.mustDB(), g.ID, u.OldUserStruct.ID))

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
	uri := router.GetRoute("POST", api.addWorkerModelHandler, nil)
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
		"permModelID": fmt.Sprintf("%d", wm.ID),
	}
	uri = router.GetRoute("PUT", api.updateWorkerModelHandler, vars)
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

func Test_addWorkerModelAsAWrongGroupMember(t *testing.T) {
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

	if err := group.SetUserGroupAdmin(api.mustDB(), g.ID, u.OldUserStruct.ID); err != nil {
		t.Fatal(err)
	}

	model := sdk.Model{
		Name:    "Test1",
		GroupID: g1.ID,
		Type:    sdk.Docker,
		ModelDocker: sdk.ModelDocker{
			Image: "buildpack-deps:jessie",
			Cmd:   "worker",
		},
		RegisteredCapabilities: sdk.RequirementList{
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

	if err := group.SetUserGroupAdmin(api.mustDB(), g.ID, u.OldUserStruct.ID); err != nil {
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
		RegisteredCapabilities: sdk.RequirementList{
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
		Name:       "Test1bis",
		Restricted: true,
		ModelDocker: sdk.ModelDocker{
			Image: "buildpack-deps:jessie",
			Cmd:   "worker",
			Shell: "sh -c",
		},
		RegisteredCapabilities: sdk.RequirementList{
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

func Test_updateWorkerModelWithPassword(t *testing.T) {
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

	if err := group.SetUserGroupAdmin(api.mustDB(), g.ID, u.OldUserStruct.ID); err != nil {
		t.Fatal(err)
	}

	model := sdk.Model{
		Name:       "Test1",
		GroupID:    g.ID,
		Type:       sdk.Docker,
		Restricted: true,
		ModelDocker: sdk.ModelDocker{
			Image:    "buildpack-deps:jessie",
			Shell:    "sh -c",
			Cmd:      "worker",
			Private:  true,
			Username: "test",
			Password: "testpw",
		},
		RegisteredCapabilities: sdk.RequirementList{
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
		Name:       "Test1bis",
		Restricted: true,
		ModelDocker: sdk.ModelDocker{
			Image:    "buildpack-deps:jessie",
			Cmd:      "worker",
			Shell:    "sh -c",
			Private:  true,
			Username: "test",
			Password: sdk.PasswordPlaceholder,
		},
		RegisteredCapabilities: sdk.RequirementList{
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
	var resp sdk.Model
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	test.Equal(t, sdk.PasswordPlaceholder, resp.ModelDocker.Password, "Worker model should not return password, but placeholder")

	wm, errL := worker.LoadWorkerModelByIDWithPassword(api.mustDB(), resp.ID)
	test.NoError(t, errL)

	pw, errPw := worker.DecryptValue(wm.ModelDocker.Password)
	test.NoError(t, errPw)

	test.Equal(t, "testpw", pw)
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

	if err := group.SetUserGroupAdmin(api.mustDB(), g.ID, u.OldUserStruct.ID); err != nil {
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
		RegisteredCapabilities: sdk.RequirementList{
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
		RegisteredCapabilities: sdk.RequirementList{
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
		RegisteredCapabilities: sdk.RequirementList{
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
	assert.Equal(t, 1, len(results[0].RegisteredCapabilities))
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
