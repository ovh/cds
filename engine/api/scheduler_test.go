package main

import (
	"testing"

	"github.com/gorilla/mux"
	"github.com/loopfz/gadgeto/iffy"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/scheduler"
	"github.com/ovh/cds/engine/api/sessionstore"
	"github.com/ovh/cds/engine/api/testwithdb"
	"github.com/ovh/cds/sdk"
)

func Test_getSchedulerApplicationPipelineHandler(t *testing.T) {
	db := database.DB()
	if db == nil {
		t.FailNow()
	}

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local", TTL: 30})
	router = &Router{authDriver, mux.NewRouter(), "/Test_getSchedulerApplicationPipelineHandler"}
	router.init()

	//Create admin user
	u, pass, err := testwithdb.InsertAdminUser(t, db)
	assert.NoError(t, err)
	assert.NotZero(t, u)
	assert.NotZero(t, pass)

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.mux)

	//Prepare data

	//Insert Project
	pkey := testwithdb.RandomString(t, 10)
	proj, err := testwithdb.InsertTestProject(t, db, pkey, pkey)
	assert.NoError(t, err)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       pkey + "_PIP",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	t.Logf("Insert Pipeline %s for Project %s", pip.Name, proj.Name)
	if err := pipeline.InsertPipeline(db, pip); err != nil {
		t.Fatal(err)
	}

	//Insert Application
	app := &sdk.Application{
		Name: "TEST_APP",
	}
	t.Logf("Insert Application %s for Project %s", app.Name, proj.Name)
	if err := application.InsertApplication(db, proj, app); err != nil {
		t.Fatal(err)
	}

	t.Logf("Attach Pipeline %s on Application %s", pip.Name, app.Name)
	if err := application.AttachPipeline(db, app.ID, pip.ID); err != nil {
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
	if err := scheduler.Insert(database.DBMap(db), s); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
		"permPipelineKey":     pip.Name,
	}
	route := router.getRoute("GET", getSchedulerApplicationPipelineHandler, vars)
	headers := testwithdb.AuthHeaders(t, u, pass)
	tester.AddCall("Test_getSchedulerApplicationPipelineHandler", "GET", route, nil).Headers(headers).Checkers(iffy.ExpectStatus(200))
	tester.Run()
}
