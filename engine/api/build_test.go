package api

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"github.com/loopfz/gadgeto/iffy"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/token"
	"github.com/ovh/cds/sdk"
)

func Test_updateStepStatusHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.Mux)

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

	if err := pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, pip, u); err != nil {
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

	pb, err := pipeline.InsertPipelineBuild(api.mustDB(), api.Cache, proj, pip, app, []sdk.Parameter{}, []sdk.Parameter{}, &sdk.DefaultEnv, 0, sdk.PipelineBuildTrigger{})
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
	if err := pipeline.InsertPipelineBuildJob(api.mustDB(), pbJob); err != nil {
		t.Fatal(err)
	}

	request := sdk.StepStatus{
		Status:    "Building",
		StepOrder: 0,
	}

	vars := map[string]string{
		"id": strconv.FormatInt(pbJob.ID, 10),
	}
	route := router.GetRoute("POST", api.updateStepStatusHandler, vars)
	headers := assets.AuthHeaders(t, u, pass)
	tester.AddCall("Test_updateStepStatusHandler", "POST", route, request).Headers(headers).Checkers(iffy.ExpectStatus(204), iffy.DumpResponse(t))
	tester.Run()
	tester.Reset()

	request.Status = "Success"
	tester.AddCall("Test_updateStepStatusHandler", "POST", route, request).Headers(headers).Checkers(iffy.ExpectStatus(204), iffy.DumpResponse(t))
	tester.Run()

	pbJobCheck, errC := pipeline.GetPipelineBuildJob(api.mustDB(), api.Cache, pbJob.ID)
	if errC != nil {
		t.Fatal(errC)
	}

	assert.Equal(t, len(pbJobCheck.Job.StepStatus), 1)
	assert.Equal(t, pbJobCheck.Job.StepStatus[0].Status, "Success")
}

func Test_addSpawnInfosPipelineBuildJobHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	//Create admin user
	u, _ := assets.InsertAdminUser(api.mustDB())

	g := &sdk.Group{
		Name: sdk.RandomString(10),
	}
	if err := group.InsertGroup(api.mustDB(), g); err != nil {
		t.Fatal(err)
	}
	if err := group.InsertUserInGroup(api.mustDB(), g.ID, u.ID, true); err != nil {
		t.Fatal(err)
	}

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.Mux)

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

	if err := pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, pip, u); err != nil {
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

	pb, erri := pipeline.InsertPipelineBuild(api.mustDB(), api.Cache, proj, pip, app, []sdk.Parameter{}, []sdk.Parameter{}, &sdk.DefaultEnv, 0, sdk.PipelineBuildTrigger{})
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
	if err := pipeline.InsertPipelineBuildJob(api.mustDB(), pbJob); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"id": strconv.FormatInt(pbJob.ID, 10),
	}
	route := router.GetRoute("POST", api.addSpawnInfosPipelineBuildJobHandler, vars)

	h := http.Header{}
	h.Set("User-Agent", string(sdk.HatcheryAgent))

	tk, errg := token.GenerateToken()
	if errg != nil {
		t.Fatal(errg)
	}
	if err := token.InsertToken(api.mustDB(), g.ID, tk, sdk.Daily); err != nil {
		t.Fatal(err)
	}

	hatch := sdk.Hatchery{
		Name:    "HATCHERY_TEST",
		GroupID: g.ID,
	}
	if err := hatchery.InsertHatchery(api.mustDB(), &hatch); err != nil {
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
	h.Add("User-Agent", sdk.HatcheryAgent)

	tester.AddCall("Test_addSpawnInfosPipelineBuildJobHandler", "POST", route, request).Headers(h).Checkers(iffy.ExpectStatus(200), iffy.DumpResponse(t))
	tester.Run()

	pbJobCheck, errC := pipeline.GetPipelineBuildJob(api.mustDB(), api.Cache, pbJob.ID)
	if errC != nil {
		t.Fatal(errC)
	}

	assert.Equal(t, len(pbJobCheck.SpawnInfos), 1)
	assert.Equal(t, pbJobCheck.SpawnInfos[0].Message.ID, sdk.MsgSpawnInfoHatcheryStarts.ID)
}
