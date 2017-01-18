package main

import (
	"strconv"
	"testing"

	"github.com/gorilla/mux"
	"github.com/loopfz/gadgeto/iffy"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/scheduler"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func Test_getSchedulerApplicationPipelineHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/Test_getSchedulerApplicationPipelineHandler"}
	router.init()

	//Create admin user
	u, pass := test.InsertAdminUser(t, db)

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.mux)

	//Insert Project
	pkey := test.RandomString(t, 10)
	proj := test.InsertTestProject(t, db, pkey, pkey)

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

	scheduler.SchedulerRun()
	scheduler.ExecuterRun()

	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
		"permPipelineKey":     pip.Name,
	}
	route := router.getRoute("GET", getSchedulerApplicationPipelineHandler, vars)
	headers := test.AuthHeaders(t, u, pass)
	tester.AddCall("Test_getSchedulerApplicationPipelineHandler", "GET", route, nil).Headers(headers).Checkers(iffy.ExpectStatus(200), iffy.ExpectListLength(1), iffy.DumpResponse(t))
	tester.Run()
}

func Test_addSchedulerApplicationPipelineHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/Test_addSchedulerApplicationPipelineHandler"}
	router.init()

	//Create admin user
	u, pass := test.InsertAdminUser(t, db)

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.mux)

	//Insert Project
	pkey := test.RandomString(t, 10)
	proj := test.InsertTestProject(t, db, pkey, pkey)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       pkey + "_PIP",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}

	if err := pipeline.InsertPipeline(db, pip); err != nil {
		t.Fatal(err)
	}

	//Insert Application
	app := &sdk.Application{
		Name: "TEST_APP",
	}

	if err := application.InsertApplication(db, proj, app); err != nil {
		t.Fatal(err)
	}

	if err := application.AttachPipeline(db, app.ID, pip.ID); err != nil {
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
	route := router.getRoute("POST", addSchedulerApplicationPipelineHandler, vars)
	headers := test.AuthHeaders(t, u, pass)
	tester.AddCall("Test_addSchedulerApplicationPipelineHandler", "POST", route, s).Headers(headers).Checkers(iffy.ExpectStatus(201), iffy.DumpResponse(t))
	tester.Run()
	tester.Reset()

	scheduler.SchedulerRun()
	scheduler.ExecuterRun()

	route = router.getRoute("GET", getSchedulerApplicationPipelineHandler, vars)
	tester.AddCall("Test_getSchedulerApplicationPipelineHandler", "GET", route, nil).Headers(headers).Checkers(iffy.ExpectStatus(200), iffy.ExpectListLength(1), iffy.DumpResponse(t))
	tester.Run()
}

func Test_updateSchedulerApplicationPipelineHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/Test_updatechedulerApplicationPipelineHandler"}
	router.init()

	//Create admin user
	u, pass := test.InsertAdminUser(t, db)

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.mux)

	//Insert Project
	pkey := test.RandomString(t, 10)
	proj := test.InsertTestProject(t, db, pkey, pkey)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       pkey + "_PIP",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}

	if err := pipeline.InsertPipeline(db, pip); err != nil {
		t.Fatal(err)
	}

	//Insert Application
	app := &sdk.Application{
		Name: "TEST_APP",
	}

	if err := application.InsertApplication(db, proj, app); err != nil {
		t.Fatal(err)
	}

	if err := application.AttachPipeline(db, app.ID, pip.ID); err != nil {
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
	route := router.getRoute("POST", addSchedulerApplicationPipelineHandler, vars)
	headers := test.AuthHeaders(t, u, pass)
	tester.AddCall("Test_updatechedulerApplicationPipelineHandler", "POST", route, s).Headers(headers).Checkers(iffy.ExpectStatus(201), iffy.DumpResponse(t), iffy.UnmarshalResponse(&s))
	route = router.getRoute("PUT", updateSchedulerApplicationPipelineHandler, vars)
	tester.AddCall("Test_updatechedulerApplicationPipelineHandler", "PUT", route, s).Headers(headers).Checkers(iffy.ExpectStatus(200), iffy.DumpResponse(t))
	tester.Run()
	tester.Reset()

	scheduler.SchedulerRun()
	scheduler.ExecuterRun()

	route = router.getRoute("GET", getSchedulerApplicationPipelineHandler, vars)
	tester.AddCall("Test_updatechedulerApplicationPipelineHandler", "GET", route, nil).Headers(headers).Checkers(iffy.ExpectStatus(200), iffy.ExpectListLength(1), iffy.DumpResponse(t))
	tester.Run()
}

func Test_deleteSchedulerApplicationPipelineHandler(t *testing.T) {
	db := test.SetupPG(t)
	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/Test_deleteSchedulerApplicationPipelineHandler"}
	router.init()

	//Create admin user
	u, pass := test.InsertAdminUser(t, db)

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.mux)

	//Insert Project
	pkey := test.RandomString(t, 10)
	proj := test.InsertTestProject(t, db, pkey, pkey)

	//Insert Pipeline
	pip := &sdk.Pipeline{
		Name:       pkey + "_PIP",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}

	if err := pipeline.InsertPipeline(db, pip); err != nil {
		t.Fatal(err)
	}

	//Insert Application
	app := &sdk.Application{
		Name: "TEST_APP",
	}

	if err := application.InsertApplication(db, proj, app); err != nil {
		t.Fatal(err)
	}

	if err := application.AttachPipeline(db, app.ID, pip.ID); err != nil {
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
	route := router.getRoute("POST", addSchedulerApplicationPipelineHandler, vars)
	headers := test.AuthHeaders(t, u, pass)
	tester.AddCall("Test_deleteSchedulerApplicationPipelineHandler", "POST", route, s).Headers(headers).Checkers(iffy.ExpectStatus(201), iffy.DumpResponse(t), iffy.UnmarshalResponse(&s))

	tester.Run()
	tester.Reset()

	vars["id"] = strconv.FormatInt(s.ID, 10)
	route = router.getRoute("DELETE", deleteSchedulerApplicationPipelineHandler, vars)
	tester.AddCall("Test_deleteSchedulerApplicationPipelineHandler", "DELETE", route, nil).Headers(headers).Checkers(iffy.ExpectStatus(200))

	tester.Run()
	tester.Reset()

	route = router.getRoute("GET", getSchedulerApplicationPipelineHandler, vars)
	tester.AddCall("Test_deleteSchedulerApplicationPipelineHandler", "GET", route, nil).Headers(headers).Checkers(iffy.ExpectStatus(200), iffy.ExpectListLength(0), iffy.DumpResponse(t))
	tester.Run()
}
