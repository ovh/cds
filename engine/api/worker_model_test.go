package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workermodel"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_DeleteAllWorkerModels(t *testing.T) {
	api, _, _ := newTestAPI(t)

	// Load and delete all worker
	workers, err := worker.LoadAll(context.Background(), api.mustDB())
	require.NoError(t, err, "unable to load workers")
	for _, w := range workers {
		assert.NoError(t, worker.Delete(api.mustDB(), w.ID))
	}

	// Load and delete all worker models
	models, err := workermodel.LoadAll(context.Background(), api.mustDB(), nil)
	require.NoError(t, err)

	for _, m := range models {
		assert.NoError(t, workermodel.DeleteByID(api.mustDB(), m.ID))
	}

	// Load and delete all worker model patterns
	modelPatterns, err := workermodel.LoadPatterns(context.TODO(), api.mustDB())
	require.NoError(t, err)

	for _, wmp := range modelPatterns {
		assert.NoError(t, workermodel.DeletePatternByID(api.mustDB(), wmp.ID))
	}
}

func Test_postWorkerModelAsAdmin(t *testing.T) {
	Test_DeleteAllWorkerModels(t)

	api, db, _ := newTestAPI(t)

	_, jwtRaw := assets.InsertAdminUser(t, db)

	groupShared, err := group.LoadByName(context.TODO(), api.mustDB(), sdk.SharedInfraGroupName)
	require.NoError(t, err)

	model := sdk.Model{
		Name:    "Test1",
		GroupID: groupShared.ID,
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

	// Send POST model request
	uri := api.Router.GetRoute("POST", api.postWorkerModelHandler, nil)
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtRaw, "POST", uri, model)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var newModel sdk.Model
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &newModel))

	assert.Equal(t, groupShared.ID, newModel.GroupID)
	assert.Equal(t, "worker --api={{.API}}", newModel.ModelDocker.Cmd, "Main worker command is not good")
	assert.Equal(t, "THIS IS A TEST", newModel.ModelDocker.Envs["CDS_TEST"], "Worker model envs are not good")
}

func Test_addWorkerModelWithPrivateRegistryAsAdmin(t *testing.T) {
	api, db, _ := newTestAPI(t)

	//Loading all models
	models, errlw := workermodel.LoadAll(context.Background(), api.mustDB(), nil)
	if errlw != nil {
		t.Fatalf("Error getting models : %s", errlw)
	}

	//Delete all of them
	for _, m := range models {
		if err := workermodel.DeleteByID(api.mustDB(), m.ID); err != nil {
			t.Fatalf("Error deleting model : %s", err)
		}
	}

	//Create admin user
	u, jwt := assets.InsertAdminUser(t, db)
	assert.NotZero(t, u)
	assert.NotZero(t, jwt)

	g, err := group.LoadByName(context.TODO(), api.mustDB(), "shared.infra")
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
	uri := api.Router.GetRoute("POST", api.postWorkerModelHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var newModel sdk.Model
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &newModel))

	test.Equal(t, "worker --api={{.API}}", newModel.ModelDocker.Cmd, "Main worker command is not good")
	test.Equal(t, "THIS IS A TEST", newModel.ModelDocker.Envs["CDS_TEST"], "Worker model envs are not good")
	test.Equal(t, "{{.secrets.registry_password}}", newModel.ModelDocker.Password)

	secrets, err := workermodel.LoadSecretsByModelID(context.TODO(), api.mustDB(), newModel.ID)
	require.NoError(t, err)
	require.Len(t, secrets, 1)
	assert.Equal(t, "secrets.registry_password", secrets[0].Name)
	assert.Equal(t, "pwtest", secrets[0].Value)
}

func Test_WorkerModelUsage(t *testing.T) {
	Test_DeleteAllWorkerModels(t)

	api, db, router := newTestAPI(t)

	u, jwt := assets.InsertAdminUser(t, db)
	assert.NotZero(t, u)

	grName := sdk.RandomString(10)
	gr := assets.InsertTestGroup(t, db, grName)
	test.NotNil(t, gr)

	model := sdk.Model{
		Name:    sdk.RandomString(10),
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
	test.NoError(t, workermodel.Insert(context.TODO(), db, &model))

	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey)
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
		GroupID:            proj.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
	}))

	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}

	test.NoError(t, pipeline.InsertPipeline(db, &pip))

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
					Name:  fmt.Sprintf("%s/%s", grName, model.Name),
					Type:  sdk.ModelRequirement,
					Value: fmt.Sprintf("%s/%s", grName, model.Name),
				},
			},
		},
		Enabled: true,
	}
	errJob := pipeline.InsertJob(db, job, stage.ID, &pip)
	test.NoError(t, errJob)
	assert.NotZero(t, job.PipelineActionID)
	assert.NotZero(t, job.Action.ID)

	proj, _ = project.LoadByID(db, proj.ID,
		project.LoadOptions.WithApplications,
		project.LoadOptions.WithPipelines,
		project.LoadOptions.WithEnvironments,
		project.LoadOptions.WithGroups,
	)

	wf := sdk.Workflow{
		Name:       "workflow1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "root",
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}

	test.NoError(t, workflow.Insert(context.Background(), db, api.Cache, *proj, &wf))

	//Prepare request
	vars := map[string]string{
		"permGroupName": gr.Name,
		"permModelName": model.Name,
	}
	uri := router.GetRoute("GET", api.getWorkerModelUsageHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, vars)
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
	Test_DeleteAllWorkerModels(t)

	api, db, router := newTestAPI(t)

	//Create admin user
	u, jwt := assets.InsertAdminUser(t, db)
	assert.NotZero(t, u)
	assert.NotZero(t, jwt)

	g, err := group.LoadByName(context.TODO(), api.mustDB(), "shared.infra")
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
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, model)
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
	req = assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, model)

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
	req = assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, model)

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
	req = assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, model)

	//Do the request
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)

	t.Logf("Body: %s", w.Body.String())

	//SendBadRequest

	//Prepare request
	req = assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, "blabla")

	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)

	t.Logf("Body: %s", w.Body.String())
}

func Test_postWorkerModelAsAGroupMember(t *testing.T) {
	Test_DeleteAllWorkerModels(t)

	api, db, router := newTestAPI(t)

	//Create group
	g := &sdk.Group{
		Name: sdk.RandomString(10),
	}

	//Create user
	u, jwt := assets.InsertLambdaUser(t, db, g)
	assert.NotZero(t, u)
	assert.NotZero(t, jwt)

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

	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 403, w.Code, "Status code should be 403 because only a group admin can create a model")

	t.Logf("Body: %s", w.Body.String())
}

func Test_postWorkerModelAsAGroupAdmin(t *testing.T) {
	Test_DeleteAllWorkerModels(t)

	api, db, router := newTestAPI(t)

	//Create group
	g := &sdk.Group{
		Name: sdk.RandomString(10),
	}

	//Create user
	u, jwt := assets.InsertLambdaUser(t, db, g)
	assets.SetUserGroupAdmin(t, db, g.ID, u.ID)

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

	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 403, w.Code, "Status code should equal 403 because the worker model haven't pattern and is not restricted")

	t.Logf("Body: %s", w.Body.String())
}

func Test_postWorkerModelAsAGroupAdminWithRestrict(t *testing.T) {
	Test_DeleteAllWorkerModels(t)

	api, db, router := newTestAPI(t)

	//Create group
	g := &sdk.Group{
		Name: sdk.RandomString(10),
	}

	//Create user
	u, jwt := assets.InsertLambdaUser(t, db, g)
	assets.SetUserGroupAdmin(t, db, g.ID, u.ID)

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

	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code, "Status code should equal 200")

	var newModel sdk.Model
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &newModel))

	test.Equal(t, "worker --api={{.API}}", newModel.ModelDocker.Cmd, "Main worker command is not good")
}

func Test_postWorkerModelAsAGroupAdminWithoutRestrictWithPattern(t *testing.T) {
	Test_DeleteAllWorkerModels(t)

	api, db, router := newTestAPI(t)

	//Create group
	g := &sdk.Group{
		Name: sdk.RandomString(10),
	}

	//Create user
	u, jwt := assets.InsertLambdaUser(t, db, g)
	assert.NotZero(t, u)
	assert.NotZero(t, jwt)
	assets.SetUserGroupAdmin(t, db, g.ID, u.ID)

	pattern := sdk.ModelPattern{
		Name: "test",
		Type: sdk.Openstack,
		Model: sdk.ModelCmds{
			PreCmd: "apt-get install curl -y",
			Cmd:    "./worker",
		},
	}

	test.NoError(t, workermodel.InsertPattern(api.mustDB(), &pattern))

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

	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code, "Status code should equal 200")

	var newModel sdk.Model
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &newModel))

	test.Equal(t, "./worker", newModel.ModelVirtualMachine.Cmd, "Main worker command is not good")
	test.Equal(t, "apt-get install curl -y", newModel.ModelVirtualMachine.PreCmd, "Pre worker command is not good")
}

func Test_postWorkerModelAsAWrongGroupMember(t *testing.T) {
	Test_DeleteAllWorkerModels(t)

	api, db, router := newTestAPI(t)

	//Create group
	g := &sdk.Group{
		Name: sdk.RandomString(10),
	}

	//Create group
	g1 := &sdk.Group{
		Name: sdk.RandomString(10),
	}

	require.NoError(t, group.Insert(context.TODO(), db, g1))

	//Create user
	u, jwt := assets.InsertLambdaUser(t, db, g)
	assets.SetUserGroupAdmin(t, db, g.ID, u.ID)

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

	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 403, w.Code, "Status code should be 403 because only a group admin can create a model")

	t.Logf("Body: %s", w.Body.String())
}

func Test_putWorkerModel(t *testing.T) {
	Test_DeleteAllWorkerModels(t)

	api, db, router := newTestAPI(t)

	//Create group
	g := &sdk.Group{
		Name: sdk.RandomString(10),
	}

	//Create user
	u, jwt := assets.InsertLambdaUser(t, db, g)
	assets.SetUserGroupAdmin(t, db, g.ID, u.ID)

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

	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, model)

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
		"permGroupName": g.Name,
		"permModelName": model.Name,
	}
	uri = router.GetRoute("PUT", api.putWorkerModelHandler, vars)
	test.NotEmpty(t, uri)

	req = assets.NewJWTAuthentifiedRequest(t, jwt, "PUT", uri, model2)

	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())
}

func Test_putWorkerModelWithPassword(t *testing.T) {
	Test_DeleteAllWorkerModels(t)

	api, db, router := newTestAPI(t)

	//Create group
	g := &sdk.Group{
		Name: sdk.RandomString(10),
	}

	//Create user
	u, jwt := assets.InsertLambdaUser(t, db, g)
	assets.SetUserGroupAdmin(t, db, g.ID, u.ID)

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
	uri := router.GetRoute("POST", api.postWorkerModelHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, model)

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
		"permGroupName": g.Name,
		"permModelName": model.Name,
	}
	uri = router.GetRoute("PUT", api.putWorkerModelHandler, vars)
	test.NotEmpty(t, uri)

	req = assets.NewJWTAuthentifiedRequest(t, jwt, "PUT", uri, model2)

	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	var resp sdk.Model
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	test.Equal(t, "{{.secrets.registry_password}}", resp.ModelDocker.Password)

	secrets, err := workermodel.LoadSecretsByModelID(context.TODO(), api.mustDB(), resp.ID)
	require.NoError(t, err)
	require.Len(t, secrets, 1)
	assert.Equal(t, "secrets.registry_password", secrets[0].Name)
	assert.Equal(t, "pwtest", secrets[0].Value)
}

func Test_deleteWorkerModel(t *testing.T) {
	Test_DeleteAllWorkerModels(t)

	api, db, router := newTestAPI(t)

	//Create group
	g := &sdk.Group{
		Name: sdk.RandomString(10),
	}

	//Create user
	u, jwt := assets.InsertLambdaUser(t, db, g)
	assets.SetUserGroupAdmin(t, db, g.ID, u.ID)

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

	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())

	json.Unmarshal(w.Body.Bytes(), &model)

	//Prepare request
	vars := map[string]string{
		"permGroupName": g.Name,
		"permModelName": model.Name,
	}
	uri = router.GetRoute("DELETE", api.deleteWorkerModelHandler, vars)
	test.NotEmpty(t, uri)

	req = assets.NewJWTAuthentifiedRequest(t, jwt, "DELETE", uri, nil)

	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 204, w.Code)

	t.Logf("Body: %s", w.Body.String())
}

func Test_getWorkerModel(t *testing.T) {
	Test_DeleteAllWorkerModels(t)

	api, db, router := newTestAPI(t)

	//Create admin user
	u, jwt := assets.InsertAdminUser(t, db)
	assert.NotZero(t, u)
	assert.NotZero(t, jwt)

	g, err := group.LoadByName(context.TODO(), api.mustDB(), "shared.infra")
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

	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, model)

	//Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())

	//Prepare request
	uri = router.GetRoute("GET", api.getWorkerModelsHandler, nil)
	test.NotEmpty(t, uri)

	req = assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri+"?name=Test1", nil)

	//Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	t.Logf("Body: %s", w.Body.String())
}

func Test_getWorkerModels(t *testing.T) {
	Test_DeleteAllWorkerModels(t)

	api, db, router := newTestAPI(t)

	admin, jwtAdmin := assets.InsertAdminUser(t, db)

	g1 := &sdk.Group{Name: sdk.RandomString(10)}
	g2 := assets.InsertGroup(t, db)
	_, jwtGroupMember := assets.InsertLambdaUser(t, db, g1)

	// Create a hatchery for the admin user
	adminConsumer, _ := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, admin.ID, authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
	consumerOptions := builtin.NewConsumerOptions{
		Name:     sdk.RandomString(10),
		GroupIDs: []int64{g2.ID},
		Scopes:   sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeHatchery, sdk.AuthConsumerScopeRunExecution, sdk.AuthConsumerScopeService, sdk.AuthConsumerScopeWorkerModel),
	}
	hatcheryConsumer, _, err := builtin.NewConsumer(context.TODO(), db, consumerOptions, adminConsumer)
	require.NoError(t, err)
	require.NoError(t, services.Insert(context.TODO(), db, &sdk.Service{
		CanonicalService: sdk.CanonicalService{
			Name:       hatcheryConsumer.Name,
			Type:       sdk.TypeHatchery,
			ConsumerID: &hatcheryConsumer.ID,
		},
	}))
	sessionHatchery, err := authentication.NewSession(context.TODO(), db, &hatcheryConsumer.AuthConsumer, 5*time.Minute)
	require.NoError(t, err)
	jwtHatchery, err := authentication.NewSessionJWT(sessionHatchery, "")
	require.NoError(t, err)

	m1 := sdk.Model{
		Name:    "A" + sdk.RandomString(10),
		GroupID: g1.ID,
		Type:    sdk.Docker,
	}
	require.NoError(t, workermodel.Insert(context.TODO(), db, &m1))

	m2 := sdk.Model{
		Name:    "B" + sdk.RandomString(10),
		GroupID: g1.ID,
		Type:    sdk.Docker,
	}
	require.NoError(t, workermodel.Insert(context.TODO(), db, &m2))

	m3 := sdk.Model{
		Name:    "C" + sdk.RandomString(10),
		GroupID: g2.ID,
		Type:    sdk.Docker,
	}
	require.NoError(t, workermodel.Insert(context.TODO(), db, &m3))

	// getWorkerModelsHandler by admin
	uri := router.GetRoute(http.MethodGet, api.getWorkerModelsHandler, nil)
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtAdmin, http.MethodGet, uri, nil)
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	results := []sdk.Model{}
	json.Unmarshal(w.Body.Bytes(), &results)
	require.Equal(t, 3, len(results))
	assert.Equal(t, m1.Name, results[0].Name)
	assert.Equal(t, m2.Name, results[1].Name)
	assert.Equal(t, m3.Name, results[2].Name)

	// getWorkerModelsHandler by group member
	uri = router.GetRoute(http.MethodGet, api.getWorkerModelsHandler, nil)
	test.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtGroupMember, http.MethodGet, uri, nil)
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	json.Unmarshal(w.Body.Bytes(), &results)
	require.Equal(t, 2, len(results))
	assert.Equal(t, m1.Name, results[0].Name)
	assert.Equal(t, m2.Name, results[1].Name)

	// getWorkerModelsHandler by hatchery with groups
	uri = router.GetRoute(http.MethodGet, api.getWorkerModelsHandler, nil)
	test.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtHatchery, http.MethodGet, uri, nil)
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	json.Unmarshal(w.Body.Bytes(), &results)
	require.Equal(t, 1, len(results))
	assert.Equal(t, m3.Name, results[0].Name)

	// getWorkerModelsForGroupHandler
	uri = router.GetRoute(http.MethodGet, api.getWorkerModelsForGroupHandler, map[string]string{
		"permGroupName": g2.Name,
	})
	test.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, jwtAdmin, http.MethodGet, uri, nil)
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	json.Unmarshal(w.Body.Bytes(), &results)
	require.Equal(t, 1, len(results))
	assert.Equal(t, m3.Name, results[0].Name)
}

// This test create a worker model then an action that will use it.
// Next the model group and name will be updated and we want to check if the requirement was updated.
func Test_renameWorkerModel(t *testing.T) {
	Test_DeleteAllWorkerModels(t)

	api, db, router := newTestAPI(t)

	// create new group
	g1 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	// create new group
	g2 := assets.InsertTestGroup(t, db, sdk.RandomString(10))

	// create admin user
	u, jwt := assets.InsertAdminUser(t, db)
	assert.NotZero(t, u)
	assert.NotZero(t, jwt)

	// prepare post model request
	uri := router.GetRoute("POST", api.postWorkerModelHandler, nil)
	test.NotEmpty(t, uri)

	initialName := sdk.RandomString(10)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, sdk.Model{
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
	req = assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, sdk.Action{
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
		"permGroupName": result.Group.Name,
		"permModelName": result.Name,
	})
	test.NotEmpty(t, uri)

	newName := sdk.RandomString(10)
	result.Name = newName
	result.GroupID = g2.ID
	req = assets.NewJWTAuthentifiedRequest(t, jwt, "PUT", uri, result)

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
		"permGroupName":  action.Group.Name,
		"permActionName": action.Name,
	})
	test.NotEmpty(t, uri)

	req = assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil)

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
