package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/moby/moby/pkg/namesgenerator"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/token"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

func Test_updateStepStatusHandler(t *testing.T) {
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

	jsonBody, _ := json.Marshal(request)
	body := bytes.NewBuffer(jsonBody)
	uri := router.GetRoute("POST", api.updateStepStatusHandler, vars)
	req, err := http.NewRequest("POST", uri, body)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 204, w.Code)

	request.Status = "Success"
	jsonBody, _ = json.Marshal(request)
	body = bytes.NewBuffer(jsonBody)
	uri = router.GetRoute("POST", api.updateStepStatusHandler, vars)
	req, err = http.NewRequest("POST", uri, body)
	test.NoError(t, err)
	assets.AuthentifyRequest(t, req, u, pass)

	// Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 204, w.Code)

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

	h := http.Header{}
	h.Set("User-Agent", string(sdk.HatcheryAgent))

	tk, errg := token.GenerateToken()
	if errg != nil {
		t.Fatal(errg)
	}
	if err := token.InsertToken(api.mustDB(), g.ID, tk, sdk.Daily, "", ""); err != nil {
		t.Fatal(err)
	}

	name := "HATCHERY_TEST_" + namesgenerator.GetRandomName(0)
	hatch := sdk.Hatchery{
		Name:    name,
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

	jsonBody, _ := json.Marshal(request)
	body := bytes.NewBuffer(jsonBody)
	uri := router.GetRoute("POST", api.addSpawnInfosPipelineBuildJobHandler, vars)
	req, err := http.NewRequest("POST", uri, body)
	test.NoError(t, err)
	basedHash := base64.StdEncoding.EncodeToString([]byte(hatch.UID))
	req.Header.Add(sdk.AuthHeader, basedHash)
	req.Header.Add(cdsclient.RequestedNameHeader, name)
	req.Header.Add(sdk.SessionTokenHeader, tk)
	req.Header.Add("User-Agent", sdk.HatcheryAgent)

	// Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	test.NoError(t, json.Unmarshal(w.Body.Bytes(), &app))

	pbJobCheck, errC := pipeline.GetPipelineBuildJob(api.mustDB(), api.Cache, pbJob.ID)
	if errC != nil {
		t.Fatal(errC)
	}

	assert.Equal(t, len(pbJobCheck.SpawnInfos), 1)
	assert.Equal(t, pbJobCheck.SpawnInfos[0].Message.ID, sdk.MsgSpawnInfoHatcheryStarts.ID)
}
