package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/gorilla/mux"
	"github.com/loopfz/gadgeto/iffy"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/sdk"
)

func Test_updateStepStatusHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/Test_updateStepStatusHandler"}
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

	//Insert Application
	app := &sdk.Application{
		Name: "TEST_APP",
	}

	if err := application.Insert(db, proj, app); err != nil {
		t.Fatal(err)
	}

	if _, err := application.AttachPipeline(db, app.ID, pip.ID); err != nil {
		t.Fatal(err)
	}

	pb, err := pipeline.InsertPipelineBuild(db, proj, pip, app, []sdk.Parameter{}, []sdk.Parameter{}, &sdk.DefaultEnv, 0, sdk.PipelineBuildTrigger{})
	if err != nil {
		t.Fatal(err)
	}

	pbJob := &sdk.PipelineBuildJob{
		Status:          "Building",
		PipelineBuildID: pb.ID,
		Job: sdk.ExecutedJob{
			Job:        sdk.Job{},
			Reason:     "",
			StepStatus: []sdk.StepStatus{},
		},
	}
	if err := pipeline.InsertPipelineBuildJob(db, pbJob); err != nil {
		t.Fatal(err)
	}

	request := sdk.StepStatus{
		Status:    "Building",
		StepOrder: 0,
	}

	vars := map[string]string{
		"id": strconv.FormatInt(pbJob.ID, 10),
	}
	route := router.getRoute("POST", updateStepStatusHandler, vars)
	headers := assets.AuthHeaders(t, u, pass)
	tester.AddCall("Test_updateStepStatusHandler", "POST", route, request).Headers(headers).Checkers(iffy.ExpectStatus(200), iffy.DumpResponse(t))
	tester.Run()
	tester.Reset()

	request.Status = "Success"
	tester.AddCall("Test_updateStepStatusHandler", "POST", route, request).Headers(headers).Checkers(iffy.ExpectStatus(200), iffy.DumpResponse(t))
	tester.Run()

	pbJobCheck, errC := pipeline.GetPipelineBuildJob(db, pbJob.ID)
	if errC != nil {
		t.Fatal(errC)
	}

	assert.Equal(t, len(pbJobCheck.Job.StepStatus), 1)
	assert.Equal(t, pbJobCheck.Job.StepStatus[0].Status, "Success")
}

func Test_addSpawnInfosPipelineBuildJobHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/Test_addSpawnInfosPipelineBuildJobHandler"}
	router.init()

	//Create admin user
	u, _ := assets.InsertAdminUser(t, db)

	g := &sdk.Group{
		Name: assets.RandomString(t, 10),
	}
	if err := group.InsertGroup(db, g); err != nil {
		t.Fatal(err)
	}
	if err := group.InsertUserInGroup(db, g.ID, u.ID, true); err != nil {
		t.Fatal(err)
	}

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

	//Insert Application
	app := &sdk.Application{
		Name: "TEST_APP",
	}

	if err := application.Insert(db, proj, app); err != nil {
		t.Fatal(err)
	}

	if _, err := application.AttachPipeline(db, app.ID, pip.ID); err != nil {
		t.Fatal(err)
	}

	pb, erri := pipeline.InsertPipelineBuild(db, proj, pip, app, []sdk.Parameter{}, []sdk.Parameter{}, &sdk.DefaultEnv, 0, sdk.PipelineBuildTrigger{})
	if erri != nil {
		t.Fatal(erri)
	}

	pbJob := &sdk.PipelineBuildJob{
		Status:          "Building",
		PipelineBuildID: pb.ID,
		Job: sdk.ExecutedJob{
			Job:        sdk.Job{},
			Reason:     "",
			StepStatus: []sdk.StepStatus{},
		},
	}
	if err := pipeline.InsertPipelineBuildJob(db, pbJob); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"id": strconv.FormatInt(pbJob.ID, 10),
	}
	route := router.getRoute("POST", addSpawnInfosPipelineBuildJobHandler, vars)

	h := http.Header{}
	h.Set("User-Agent", string(sdk.HatcheryAgent))

	tk, errg := worker.GenerateToken()
	if errg != nil {
		t.Fatal(errg)
	}
	if err := worker.InsertToken(db, g.ID, tk, sdk.Daily); err != nil {
		t.Fatal(err)
	}

	hatch := sdk.Hatchery{
		Name:    "HATCHERY_TEST",
		GroupID: g.ID,
	}
	if err := hatchery.InsertHatchery(db, &hatch); err != nil {
		t.Fatal(err)
	}

	request := []sdk.SpawnInfo{
		{
			Message: sdk.SpawnMsg{ID: sdk.MsgSpawnInfoHatcheryStarts.ID, Args: []interface{}{fmt.Sprintf("%d", hatch.ID), "model.name"}},
		},
	}

	basedHash := base64.StdEncoding.EncodeToString([]byte(hatch.UID))
	h.Set(sdk.AuthHeader, basedHash)
	h.Add(sdk.SessionTokenHeader, tk)

	tester.AddCall("Test_addSpawnInfosPipelineBuildJobHandler", "POST", route, request).Headers(h).Checkers(iffy.ExpectStatus(200), iffy.DumpResponse(t))
	tester.Run()

	pbJobCheck, errC := pipeline.GetPipelineBuildJob(db, pbJob.ID)
	if errC != nil {
		t.Fatal(errC)
	}

	assert.Equal(t, len(pbJobCheck.SpawnInfos), 1)
	assert.Equal(t, pbJobCheck.SpawnInfos[0].Message.ID, sdk.MsgSpawnInfoHatcheryStarts.ID)
}
