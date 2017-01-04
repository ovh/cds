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
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/testwithdb"
	test "github.com/ovh/cds/engine/api/testwithdb"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/sdk"
)

func TestAddTriggerHandler(t *testing.T) {
	if test.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := test.SetupPG(t)
	assert.NoError(t, err)

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local"})
	router = &Router{authDriver, mux.NewRouter(), "/TestAddJobHandler"}
	router.init()

	//1. Create admin user
	u, pass, err := test.InsertAdminUser(t, db)
	assert.NoError(t, err)

	//2. Create project
	proj, _ := testwithdb.InsertTestProject(t, db, test.RandomString(t, 10), test.RandomString(t, 10))
	assert.NotNil(t, proj)
	if proj == nil {
		t.Fail()
		return
	}

	//3. Create Pipeline 1
	pipelineKey1 := test.RandomString(t, 10)
	pip1 := &sdk.Pipeline{
		Name:       pipelineKey1,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	err = pipeline.InsertPipeline(db, pip1)
	assert.NoError(t, err)

	//4. Create Pipeline 2
	pipelineKey2 := test.RandomString(t, 10)
	pip2 := &sdk.Pipeline{
		Name:       pipelineKey2,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	err = pipeline.InsertPipeline(db, pip2)
	assert.NoError(t, err)

	//5. Create Application
	applicationName := test.RandomString(t, 10)
	app := &sdk.Application{
		Name: applicationName,
	}
	err = application.InsertApplication(db, proj, app)
	assert.NoError(t, err)

	//6. Attach pipeline 1
	err = application.AttachPipeline(db, app.ID, pip1.ID)
	assert.NoError(t, err)

	//7. Attach pipeline 2
	err = application.AttachPipeline(db, app.ID, pip2.ID)
	assert.NoError(t, err)

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
	if uri == "" {
		t.Fail()
		return
	}
	req, err := http.NewRequest("POST", uri, body)
	test.AuthentifyRequest(t, req, u, pass)

	//9. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	// 10
	ts, err := trigger.LoadTriggerByApp(db, app.ID)
	assert.NoError(t, err)
	assert.Equal(t, len(ts), 1)
}

func TestUpdateTriggerHandler(t *testing.T) {
	if test.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := test.SetupPG(t)
	assert.NoError(t, err)

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local"})
	router = &Router{authDriver, mux.NewRouter(), "/TestUpdateTriggerHandler"}
	router.init()

	//1. Create admin user
	u, pass, err := test.InsertAdminUser(t, db)
	assert.NoError(t, err)

	//2. Create project
	proj, _ := testwithdb.InsertTestProject(t, db, test.RandomString(t, 10), test.RandomString(t, 10))
	assert.NotNil(t, proj)
	if proj == nil {
		t.Fail()
		return
	}

	//3. Create Pipeline 1
	pipelineKey1 := test.RandomString(t, 10)
	pip1 := &sdk.Pipeline{
		Name:       pipelineKey1,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	err = pipeline.InsertPipeline(db, pip1)
	assert.NoError(t, err)

	//4. Create Pipeline 2
	pipelineKey2 := test.RandomString(t, 10)
	pip2 := &sdk.Pipeline{
		Name:       pipelineKey2,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	err = pipeline.InsertPipeline(db, pip2)
	assert.NoError(t, err)

	//5. Create Application
	applicationName := test.RandomString(t, 10)
	app := &sdk.Application{
		Name: applicationName,
	}
	err = application.InsertApplication(db, proj, app)
	assert.NoError(t, err)

	//6. Attach pipeline 1
	err = application.AttachPipeline(db, app.ID, pip1.ID)
	assert.NoError(t, err)

	//7. Attach pipeline 2
	err = application.AttachPipeline(db, app.ID, pip2.ID)
	assert.NoError(t, err)

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
	assert.NoError(t, err)

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
	if uri == "" {
		t.Fail()
		return
	}
	req, err := http.NewRequest("PUT", uri, body)
	test.AuthentifyRequest(t, req, u, pass)

	//9. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	// 10
	ts, err := trigger.LoadTriggerByApp(db, app.ID)
	assert.NoError(t, err)
	assert.Equal(t, len(ts), 1)
	assert.Equal(t, ts[0].Manual, true)
}

func TestRemoveTriggerHandler(t *testing.T) {
	if test.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := test.SetupPG(t)
	assert.NoError(t, err)

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local"})
	router = &Router{authDriver, mux.NewRouter(), "/TestRemoveTriggerHandler"}
	router.init()

	//1. Create admin user
	u, pass, err := test.InsertAdminUser(t, db)
	assert.NoError(t, err)

	//2. Create project
	proj, _ := testwithdb.InsertTestProject(t, db, test.RandomString(t, 10), test.RandomString(t, 10))
	assert.NotNil(t, proj)
	if proj == nil {
		t.Fail()
		return
	}

	//3. Create Pipeline 1
	pipelineKey1 := test.RandomString(t, 10)
	pip1 := &sdk.Pipeline{
		Name:       pipelineKey1,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	err = pipeline.InsertPipeline(db, pip1)
	assert.NoError(t, err)

	//4. Create Pipeline 2
	pipelineKey2 := test.RandomString(t, 10)
	pip2 := &sdk.Pipeline{
		Name:       pipelineKey2,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	err = pipeline.InsertPipeline(db, pip2)
	assert.NoError(t, err)

	//5. Create Application
	applicationName := test.RandomString(t, 10)
	app := &sdk.Application{
		Name: applicationName,
	}
	err = application.InsertApplication(db, proj, app)
	assert.NoError(t, err)

	//6. Attach pipeline 1
	err = application.AttachPipeline(db, app.ID, pip1.ID)
	assert.NoError(t, err)

	//7. Attach pipeline 2
	err = application.AttachPipeline(db, app.ID, pip2.ID)
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
		"permPipelineKey":     pip1.Name,
		"id":                  strconv.FormatInt(triggerData.ID, 10),
	}

	uri := router.getRoute("DELETE", deleteTriggerHandler, vars)
	if uri == "" {
		t.Fail()
		return
	}
	req, err := http.NewRequest("DELETE", uri, nil)
	test.AuthentifyRequest(t, req, u, pass)

	//9. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	// 10
	ts, err := trigger.LoadTriggerByApp(db, app.ID)
	assert.NoError(t, err)
	assert.Equal(t, len(ts), 0)
}
