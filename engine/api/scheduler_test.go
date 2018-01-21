package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yesnault/gadgeto/iffy"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/scheduler"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func Test_getSchedulerApplicationPipelineHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       pkey + "_PIP",
		Type:       sdk.TestingPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	if err := pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, pip, nil); err != nil {
		t.Fatal(err)
	}

	//Insert Application
	app := &sdk.Application{
		Name: "TEST_APP",
	}
	t.Logf("Insert Application %s for Project %s", app.Name, proj.Name)
	if err := application.Insert(api.mustDB(), api.Cache, proj, app, u); err != nil {
		t.Fatal(err)
	}

	t.Logf("Attach Pipeline %s on Application %s", pip.Name, app.Name)
	if _, err := application.AttachPipeline(api.mustDB(), app.ID, pip.ID); err != nil {
		t.Fatal(err)
	}

	s := &sdk.PipelineScheduler{
		ApplicationID: app.ID,
		EnvironmentID: sdk.DefaultEnv.ID,
		PipelineID:    pip.ID,
		Crontab:       "@hourly",
		Disabled:      false,
		Args: []sdk.Parameter{
			{
				Name:  "p1",
				Type:  sdk.StringParameter,
				Value: "v1",
			},
			{
				Name:  "p2",
				Type:  sdk.StringParameter,
				Value: "v2",
			},
		},
	}
	if err := scheduler.Insert(api.mustDB(), s); err != nil {
		t.Fatal(err)
	}

	scheduler.Run(api.mustDB())
	scheduler.ExecuterRun(api.mustDB, api.Cache)

	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
		"permPipelineKey":     pip.Name,
	}

	uri := router.GetRoute("GET", api.getSchedulerApplicationPipelineHandler, vars)
	req, err := http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var schedulers []sdk.PipelineScheduler
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &schedulers))
	assert.Equal(t, len(schedulers), 1)
}

func Test_addSchedulerApplicationPipelineHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)

	env := &sdk.Environment{
		Name:      pkey + "-env",
		ProjectID: proj.ID,
	}

	if err := environment.InsertEnvironment(api.mustDB(), env); err != nil {
		t.Fatal(err)
	}

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       pkey + "_PIP",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}

	if err := pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, pip, nil); err != nil {
		t.Fatal(err)
	}

	//Insert Application
	app := &sdk.Application{
		Name: "TEST_APP",
	}

	if err := application.Insert(api.mustDB(), api.Cache, proj, app, u); err != nil {
		t.Fatal(err)
	}

	if _, err := application.AttachPipeline(api.mustDB(), app.ID, pip.ID); err != nil {
		t.Fatal(err)
	}

	s := &sdk.PipelineScheduler{
		Crontab: "@hourly",
	}

	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
		"permPipelineKey":     pip.Name,
	}

	jsonBody, _ := json.Marshal(s)
	body := bytes.NewBuffer(jsonBody)
	uri := router.GetRoute("POST", api.addSchedulerApplicationPipelineHandler, vars)
	req, err := http.NewRequest("POST", uri+"?envName="+env.Name, body)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)

	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &app))

	scheduler.Run(api.mustDB())
	scheduler.ExecuterRun(api.mustDB, api.Cache)

	uri = router.GetRoute("GET", api.getSchedulerApplicationPipelineHandler, vars)
	req, err = http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var schedulers []sdk.PipelineScheduler
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &schedulers))
	assert.Equal(t, len(schedulers), 1)

}

func Test_updateSchedulerApplicationPipelineHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       pkey + "_PIP",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}

	if err := pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, pip, nil); err != nil {
		t.Fatal(err)
	}

	//Insert Application
	app := &sdk.Application{
		Name: "TEST_APP",
	}

	if err := application.Insert(api.mustDB(), api.Cache, proj, app, u); err != nil {
		t.Fatal(err)
	}

	if _, err := application.AttachPipeline(api.mustDB(), app.ID, pip.ID); err != nil {
		t.Fatal(err)
	}

	s := &sdk.PipelineScheduler{
		Crontab: "@hourly",
	}

	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
		"permPipelineKey":     pip.Name,
	}

	jsonBody, _ := json.Marshal(s)
	body := bytes.NewBuffer(jsonBody)
	uri := router.GetRoute("POST", api.addSchedulerApplicationPipelineHandler, vars)
	req, err := http.NewRequest("POST", uri, body)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)

	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &app))

	assert.Equal(t, len(app.Workflows), 1)
	assert.Equal(t, len(app.Workflows[0].Schedulers), 1)
	s = &app.Workflows[0].Schedulers[0]

	log.Warning(">>%+v", s)

	jsonBody, _ = json.Marshal(s)
	body = bytes.NewBuffer(jsonBody)
	uri = router.GetRoute("PUT", api.updateSchedulerApplicationPipelineHandler, vars)
	req, err = http.NewRequest("PUT", uri, body)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &app))

	assert.Equal(t, len(app.Workflows), 1)
	assert.Equal(t, len(app.Workflows[0].Schedulers), 1)
	// scheduler is here: &app.Workflows[0].Schedulers[0]

	scheduler.Run(api.mustDB())
	scheduler.ExecuterRun(api.mustDB, api.Cache)

	uri = router.GetRoute("GET", api.getSchedulerApplicationPipelineHandler, vars)
	req, err = http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var apps []sdk.Application
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &apps))
	assert.Equal(t, len(apps), 1)

}

func Test_deleteSchedulerApplicationPipelineHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//Insert Project
	pkey := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, pkey, pkey, u)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       pkey + "_PIP",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}

	if err := pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, pip, nil); err != nil {
		t.Fatal(err)
	}

	//Insert Application
	app := &sdk.Application{
		Name: "TEST_APP",
	}

	if err := application.Insert(api.mustDB(), api.Cache, proj, app, u); err != nil {
		t.Fatal(err)
	}

	if _, err := application.AttachPipeline(api.mustDB(), app.ID, pip.ID); err != nil {
		t.Fatal(err)
	}

	s := &sdk.PipelineScheduler{
		Crontab: "@hourly",
	}

	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
		"permPipelineKey":     pip.Name,
	}

	jsonBody, _ := json.Marshal(s)
	body := bytes.NewBuffer(jsonBody)
	uri := router.GetRoute("POST", api.addSchedulerApplicationPipelineHandler, vars)
	req, err := http.NewRequest("POST", uri, body)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 201, w.Code)

	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &app))

	assert.Equal(t, len(app.Workflows), 1)
	assert.Equal(t, len(app.Workflows[0].Schedulers), 1)
	s = &app.Workflows[0].Schedulers[0]

	vars["id"] = strconv.FormatInt(s.ID, 10)

	jsonBody, _ = json.Marshal(s)
	body = bytes.NewBuffer(jsonBody)
	uri = router.GetRoute("DELETE", api.deleteSchedulerApplicationPipelineHandler, vars)
	req, err = http.NewRequest("DELETE", uri, body)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	uri = router.GetRoute("GET", api.getSchedulerApplicationPipelineHandler, vars)
	req, err = http.NewRequest("GET", uri, nil)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	var apps []sdk.Application
	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &apps))
	assert.Equal(t, len(apps), 0)
}
