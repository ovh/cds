package main

import (
	"testing"

	"github.com/gorilla/mux"
	"github.com/loopfz/gadgeto/iffy"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_attachPipelinesToApplicationHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/Test_attachPipelinesToApplicationHandler"}
	router.init()

	//Create admin user
	u, pass := assets.InsertAdminUser(t, db)

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.mux)

	//Insert Project
	pkey := assets.RandomString(t, 10)
	proj := assets.InsertTestProject(t, db, pkey, pkey)

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

	//Insert Pipeline
	pip2 := &sdk.Pipeline{
		Name:       pkey + "_PIP2",
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}

	if err := pipeline.InsertPipeline(db, pip2); err != nil {
		t.Fatal(err)
	}

	//Insert Application
	app := &sdk.Application{
		Name: "TEST_APP",
	}

	if err := application.Insert(db, proj, app); err != nil {
		t.Fatal(err)
	}

	request := []string{pkey + "_PIP", pkey + "_PIP2"}

	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
	}
	route := router.getRoute("POST", attachPipelinesToApplicationHandler, vars)
	headers := assets.AuthHeaders(t, u, pass)
	tester.AddCall("Test_attachPipelinesToApplicationHandler", "POST", route, request).Headers(headers).Checkers(iffy.ExpectStatus(200))
	tester.Run()

	appDB, err := application.LoadByName(db, proj.Key, app.Name, u, application.LoadOptions.WithPipelines)
	test.NoError(t, err)

	assert.Equal(t, len(appDB.Pipelines), 2)
}
