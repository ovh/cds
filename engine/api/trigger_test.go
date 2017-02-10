package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/sdk"
)

func TestAddTriggerHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/TestAddJobHandler"}
	router.init()

	//1. Create admin user
	u, pass := assets.InsertAdminUser(t, db)

	//2. Create project
	proj := assets.InsertTestProject(t, db, assets.RandomString(t, 10), assets.RandomString(t, 10))
	test.NotNil(t, proj)

	//3. Create Pipeline 1
	pipelineKey1 := assets.RandomString(t, 10)
	pip1 := &sdk.Pipeline{
		Name:       pipelineKey1,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(db, pip1))

	//4. Create Pipeline 2
	pipelineKey2 := assets.RandomString(t, 10)
	pip2 := &sdk.Pipeline{
		Name:       pipelineKey2,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	err := pipeline.InsertPipeline(db, pip2)
	test.NoError(t, err)

	//5. Create Application
	applicationName := assets.RandomString(t, 10)
	app := &sdk.Application{
		Name: applicationName,
	}
	err = application.InsertApplication(db, proj, app)
	test.NoError(t, err)

	//6. Attach pipeline 1
	err = application.AttachPipeline(db, app.ID, pip1.ID)
	test.NoError(t, err)

	//7. Attach pipeline 2
	err = application.AttachPipeline(db, app.ID, pip2.ID)
	test.NoError(t, err)

	// 8. Prepare the request
	addTriggerRequest := sdk.PipelineTrigger{
		SrcProject:      *proj,
		SrcApplication:  *app,
		SrcPipeline:     *pip1,
		DestProject:     *proj,
		DestApplication: *app,
		DestPipeline:    *pip2,
		Manual:          false,
	}
	jsonBody, _ := json.Marshal(addTriggerRequest)
	body := bytes.NewBuffer(jsonBody)
	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
		"permPipelineKey":     pip1.Name,
	}

	uri := router.getRoute("POST", addTriggerHandler, vars)
	test.NotEmpty(t, uri)

	req, err := http.NewRequest("POST", uri, body)
	assets.AuthentifyRequest(t, req, u, pass)

	//9. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	// 10
	ts, err := trigger.LoadTriggerByApp(db, app.ID)
	test.NoError(t, err)
	assert.Equal(t, len(ts), 1)
}

func TestUpdateTriggerHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/TestUpdateTriggerHandler"}
	router.init()

	//1. Create admin user
	u, pass := assets.InsertAdminUser(t, db)

	//2. Create project
	proj := assets.InsertTestProject(t, db, assets.RandomString(t, 10), assets.RandomString(t, 10))
	test.NotNil(t, proj)

	//3. Create Pipeline 1
	pipelineKey1 := assets.RandomString(t, 10)
	pip1 := &sdk.Pipeline{
		Name:       pipelineKey1,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	err := pipeline.InsertPipeline(db, pip1)
	test.NoError(t, err)

	//4. Create Pipeline 2
	pipelineKey2 := assets.RandomString(t, 10)
	pip2 := &sdk.Pipeline{
		Name:       pipelineKey2,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	err = pipeline.InsertPipeline(db, pip2)
	test.NoError(t, err)

	//5. Create Application
	applicationName := assets.RandomString(t, 10)
	app := &sdk.Application{
		Name: applicationName,
	}
	err = application.InsertApplication(db, proj, app)
	test.NoError(t, err)

	//6. Attach pipeline 1
	err = application.AttachPipeline(db, app.ID, pip1.ID)
	test.NoError(t, err)

	//7. Attach pipeline 2
	err = application.AttachPipeline(db, app.ID, pip2.ID)
	test.NoError(t, err)

	// 8. InsertTrigger
	triggerData := &sdk.PipelineTrigger{
		SrcProject:      *proj,
		SrcApplication:  *app,
		SrcPipeline:     *pip1,
		DestProject:     *proj,
		DestApplication: *app,
		DestPipeline:    *pip2,
		Manual:          false,
	}

	err = trigger.InsertTrigger(db, triggerData)
	test.NoError(t, err)

	triggerData.Manual = true
	jsonBody, _ := json.Marshal(triggerData)
	body := bytes.NewBuffer(jsonBody)
	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
		"permPipelineKey":     pip1.Name,
		"id":                  strconv.FormatInt(triggerData.ID, 10),
	}

	uri := router.getRoute("PUT", updateTriggerHandler, vars)
	test.NotEmpty(t, uri)

	req, err := http.NewRequest("PUT", uri, body)
	assets.AuthentifyRequest(t, req, u, pass)

	//9. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	// 10
	ts, err := trigger.LoadTriggerByApp(db, app.ID)
	test.NoError(t, err)
	assert.Equal(t, len(ts), 1)
	assert.Equal(t, ts[0].Manual, true)
}

func TestRemoveTriggerHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/TestRemoveTriggerHandler"}
	router.init()

	//1. Create admin user
	u, pass := assets.InsertAdminUser(t, db)

	//2. Create project
	proj := assets.InsertTestProject(t, db, assets.RandomString(t, 10), assets.RandomString(t, 10))
	test.NotNil(t, proj)

	//3. Create Pipeline 1
	pipelineKey1 := assets.RandomString(t, 10)
	pip1 := &sdk.Pipeline{
		Name:       pipelineKey1,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	err := pipeline.InsertPipeline(db, pip1)

	//4. Create Pipeline 2
	pipelineKey2 := assets.RandomString(t, 10)
	pip2 := &sdk.Pipeline{
		Name:       pipelineKey2,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	err = pipeline.InsertPipeline(db, pip2)

	//5. Create Application
	applicationName := assets.RandomString(t, 10)
	app := &sdk.Application{
		Name: applicationName,
	}
	err = application.InsertApplication(db, proj, app)
	test.NoError(t, err)

	//6. Attach pipeline 1
	err = application.AttachPipeline(db, app.ID, pip1.ID)
	test.NoError(t, err)

	//7. Attach pipeline 2
	err = application.AttachPipeline(db, app.ID, pip2.ID)
	test.NoError(t, err)

	// 8. InsertTrigger
	triggerData := &sdk.PipelineTrigger{
		SrcProject:      *proj,
		SrcApplication:  *app,
		SrcPipeline:     *pip1,
		DestProject:     *proj,
		DestApplication: *app,
		DestPipeline:    *pip2,
		Manual:          false,
	}

	err = trigger.InsertTrigger(db, triggerData)
	test.NoError(t, err)

	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
		"permPipelineKey":     pip1.Name,
		"id":                  strconv.FormatInt(triggerData.ID, 10),
	}

	uri := router.getRoute("DELETE", deleteTriggerHandler, vars)
	test.NotEmpty(t, uri)

	req, err := http.NewRequest("DELETE", uri, nil)
	assets.AuthentifyRequest(t, req, u, pass)

	//9. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	// 10
	ts, err := trigger.LoadTriggerByApp(db, app.ID)
	test.NoError(t, err)
	assert.Equal(t, len(ts), 0)
}
