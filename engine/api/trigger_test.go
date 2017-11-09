package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/sdk"
)

func TestAddTriggerHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//1. Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//2. Create project
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	//3. Create Pipeline 1
	pipelineKey1 := sdk.RandomString(10)
	pip1 := &sdk.Pipeline{
		Name:       pipelineKey1,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), proj, pip1, u))

	//4. Create Pipeline 2
	pipelineKey2 := sdk.RandomString(10)
	pip2 := &sdk.Pipeline{
		Name:       pipelineKey2,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	err := pipeline.InsertPipeline(api.mustDB(), proj, pip2, u)
	test.NoError(t, err)

	//5. Create Application
	applicationName := sdk.RandomString(10)
	app := &sdk.Application{
		Name: applicationName,
	}
	err = application.Insert(api.mustDB(), api.Cache, proj, app, u)
	test.NoError(t, err)

	//6. Attach pipeline 1
	_, err = application.AttachPipeline(api.mustDB(), app.ID, pip1.ID)
	test.NoError(t, err)

	//7. Attach pipeline 2
	_, err = application.AttachPipeline(api.mustDB(), app.ID, pip2.ID)
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

	uri := router.GetRoute("POST", api.addTriggerHandler, vars)
	test.NotEmpty(t, uri)

	req, err := http.NewRequest("POST", uri, body)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	//9. Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	// 10
	ts, err := trigger.LoadTriggerByApp(api.mustDB(), app.ID)
	test.NoError(t, err)
	assert.Equal(t, len(ts), 1)
}

func TestUpdateTriggerHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//1. Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//2. Create project
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	//3. Create Pipeline 1
	pipelineKey1 := sdk.RandomString(10)
	pip1 := &sdk.Pipeline{
		Name:       pipelineKey1,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	err := pipeline.InsertPipeline(api.mustDB(), proj, pip1, u)
	test.NoError(t, err)

	//4. Create Pipeline 2
	pipelineKey2 := sdk.RandomString(10)
	pip2 := &sdk.Pipeline{
		Name:       pipelineKey2,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	err = pipeline.InsertPipeline(api.mustDB(), proj, pip2, u)
	test.NoError(t, err)

	//5. Create Application
	applicationName := sdk.RandomString(10)
	app := &sdk.Application{
		Name: applicationName,
	}
	err = application.Insert(api.mustDB(), api.Cache, proj, app, u)
	test.NoError(t, err)

	//6. Attach pipeline 1
	_, err = application.AttachPipeline(api.mustDB(), app.ID, pip1.ID)
	test.NoError(t, err)

	//7. Attach pipeline 2
	_, err = application.AttachPipeline(api.mustDB(), app.ID, pip2.ID)
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

	err = trigger.InsertTrigger(api.mustDB(), triggerData)
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

	uri := router.GetRoute("PUT", api.updateTriggerHandler, vars)
	test.NotEmpty(t, uri)

	req, err := http.NewRequest("PUT", uri, body)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	//9. Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	// 10
	ts, err := trigger.LoadTriggerByApp(api.mustDB(), app.ID)
	test.NoError(t, err)
	assert.Equal(t, len(ts), 1)
	assert.Equal(t, ts[0].Manual, true)
}

func TestRemoveTriggerHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//1. Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//2. Create project
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	//3. Create Pipeline 1
	pipelineKey1 := sdk.RandomString(10)
	pip1 := &sdk.Pipeline{
		Name:       pipelineKey1,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	err := pipeline.InsertPipeline(api.mustDB(), proj, pip1, u)
	test.NoError(t, err)

	//4. Create Pipeline 2
	pipelineKey2 := sdk.RandomString(10)
	pip2 := &sdk.Pipeline{
		Name:       pipelineKey2,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	err = pipeline.InsertPipeline(api.mustDB(), proj, pip2, u)
	test.NoError(t, err)

	//5. Create Application
	applicationName := sdk.RandomString(10)
	app := &sdk.Application{
		Name: applicationName,
	}
	err = application.Insert(api.mustDB(), api.Cache, proj, app, u)
	test.NoError(t, err)

	//6. Attach pipeline 1
	_, err = application.AttachPipeline(api.mustDB(), app.ID, pip1.ID)
	test.NoError(t, err)

	//7. Attach pipeline 2
	_, err = application.AttachPipeline(api.mustDB(), app.ID, pip2.ID)
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

	err = trigger.InsertTrigger(api.mustDB(), triggerData)
	test.NoError(t, err)

	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
		"permPipelineKey":     pip1.Name,
		"id":                  strconv.FormatInt(triggerData.ID, 10),
	}

	uri := router.GetRoute("DELETE", api.deleteTriggerHandler, vars)
	test.NotEmpty(t, uri)

	req, err := http.NewRequest("DELETE", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	//9. Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	// 10
	ts, err := trigger.LoadTriggerByApp(api.mustDB(), app.ID)
	test.NoError(t, err)
	assert.Equal(t, len(ts), 0)
}
