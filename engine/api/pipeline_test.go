package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func Test_runPipelineHandler(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)

	//1. Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())
	//2. Create project
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	//3. Create Pipeline
	pipelineKey := sdk.RandomString(10)
	pip := &sdk.Pipeline{
		Name:       pipelineKey,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, pip, nil))

	//4. Insert Application
	appName := sdk.RandomString(10)
	app := &sdk.Application{
		Name: appName,
	}
	test.NoError(t, application.Insert(api.mustDB(), api.Cache, proj, app, nil))

	//5. Attach pipeline to application
	_, err := application.AttachPipeline(api.mustDB(), app.ID, pip.ID)
	test.NoError(t, err)

	//6. Prepare the run request
	runRequest := sdk.RunRequest{}

	jsonBody, _ := json.Marshal(runRequest)
	body := bytes.NewBuffer(jsonBody)

	//7. Prepare the route
	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
		"permPipelineKey":     pip.Name,
	}
	uri := router.GetRoute("POST", api.runPipelineHandler, vars)
	test.NotEmpty(t, uri)

	//8. Send the request
	req, err := http.NewRequest("POST", uri, body)
	if err != nil {
		t.FailNow()
		return
	}
	assets.AuthentifyRequest(t, req, u, pass)

	//8. Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	//9. Check response
	assert.Equal(t, 200, w.Code)
	t.Logf("Response : %s", string(w.Body.Bytes()))

	pb := sdk.PipelineBuild{}
	if err := json.Unmarshal(w.Body.Bytes(), &pb); err != nil {
		t.Error(err)
		t.FailNow()
		return
	}

	assert.Equal(t, int64(1), pb.Version)
	assert.Equal(t, int64(1), pb.BuildNumber)
	assert.Equal(t, "NoEnv", pb.Environment.Name)

}

func Test_runPipelineWithLastParentHandler(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)

	//1. Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//2. Create project
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	//3. Create Pipeline
	pipelineKey := sdk.RandomString(10)
	pip := &sdk.Pipeline{
		Name:       pipelineKey,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, pip, nil))

	//4. Insert Application
	appName := sdk.RandomString(10)
	app := &sdk.Application{
		Name: appName,
	}
	test.NoError(t, application.Insert(api.mustDB(), api.Cache, proj, app, nil))

	//5. Attach pipeline to application
	_, err := application.AttachPipeline(api.mustDB(), app.ID, pip.ID)
	test.NoError(t, err)

	//6. Prepare the run request
	runRequest := sdk.RunRequest{}

	jsonBody, _ := json.Marshal(runRequest)
	body := bytes.NewBuffer(jsonBody)

	//7. Prepare the route
	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
		"permPipelineKey":     pip.Name,
	}
	uri := router.GetRoute("POST", api.runPipelineHandler, vars)
	test.NotEmpty(t, uri)

	//8. Send the request
	req, err := http.NewRequest("POST", uri, body)
	if err != nil {
		t.FailNow()
		return
	}
	assets.AuthentifyRequest(t, req, u, pass)

	//8. Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	//9. Check response
	assert.Equal(t, 200, w.Code)
	t.Logf("Response : %s", string(w.Body.Bytes()))

	pb := sdk.PipelineBuild{}
	if err := json.Unmarshal(w.Body.Bytes(), &pb); err != nil {
		t.Error(err)
		t.FailNow()
		return
	}

	assert.Equal(t, int64(1), pb.Version)
	assert.Equal(t, int64(1), pb.BuildNumber)
	assert.Equal(t, "NoEnv", pb.Environment.Name)

	//9. Update build status to Success
	err = pipeline.UpdatePipelineBuildStatusAndStage(api.mustDB(), &pb, sdk.StatusSuccess)
	test.NoError(t, err)

	//10. Create another Pipeline
	pip2 := &sdk.Pipeline{
		Name:       sdk.RandomString(10),
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	err = pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, pip2, u)
	test.NoError(t, err)

	//11. Insert another Application
	app2 := &sdk.Application{
		Name: sdk.RandomString(10),
	}
	err = application.Insert(api.mustDB(), api.Cache, proj, app2, nil)
	test.NoError(t, err)

	//12. Attach pipeline to application
	_, err = application.AttachPipeline(api.mustDB(), app2.ID, pip2.ID)
	test.NoError(t, err)

	//13. Prepare the pipelne trigger
	tigrou := sdk.PipelineTrigger{
		DestApplication: *app2,
		DestEnvironment: sdk.DefaultEnv,
		DestPipeline:    *pip2,
		DestProject:     *proj,
		SrcApplication:  *app,
		SrcEnvironment:  sdk.DefaultEnv,
		SrcPipeline:     *pip,
		SrcProject:      *proj,
	}

	//14. Insert the pipeline trigger
	tx, _ := api.mustDB().Begin()
	defer tx.Rollback()
	err = trigger.InsertTrigger(tx, &tigrou)
	test.NoError(t, err)
	tx.Commit()

	//15. Prepare the run request
	runRequest2 := sdk.RunRequest{
		ParentApplicationID: app.ID,
		ParentPipelineID:    pip.ID,
	}

	jsonBody, _ = json.Marshal(runRequest2)
	body = bytes.NewBuffer(jsonBody)

	t.Logf("Request : %s", string(jsonBody))

	//16. Prepare the route
	vars = map[string]string{
		"key": proj.Key,
		"permApplicationName": app2.Name,
		"permPipelineKey":     pip2.Name,
	}
	uri = router.GetRoute("POST", api.runPipelineWithLastParentHandler, vars)
	test.NotEmpty(t, uri)

	//17. Send the request
	req, err = http.NewRequest("POST", uri, body)
	if err != nil {
		t.FailNow()
		return
	}
	assets.AuthentifyRequest(t, req, u, pass)

	//18. Do the request
	w = httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	//19. Check response
	assert.Equal(t, 200, w.Code)
	t.Logf("Response : %s", string(w.Body.Bytes()))

	pb1 := sdk.PipelineBuild{}
	if err := json.Unmarshal(w.Body.Bytes(), &pb1); err != nil {
		t.Error(err)
		t.FailNow()
		return
	}

	assert.Equal(t, int64(1), pb1.Version)
	assert.Equal(t, int64(1), pb1.BuildNumber)
	assert.Equal(t, "NoEnv", pb1.Environment.Name)

	assert.Equal(t, pb.ID, pb1.Trigger.ParentPipelineBuild.ID)
	assert.Equal(t, pb.Version, pb1.Trigger.ParentPipelineBuild.Version)
	assert.Equal(t, pb.BuildNumber, pb1.Trigger.ParentPipelineBuild.BuildNumber)
	assert.Equal(t, pb.Application.ID, pb1.Trigger.ParentPipelineBuild.Application.ID)
	assert.Equal(t, pb.Pipeline.ID, pb1.Trigger.ParentPipelineBuild.Pipeline.ID)
	assert.Equal(t, pb.Environment.ID, pb1.Trigger.ParentPipelineBuild.Environment.ID)

}

func Test_deletePipelineHandler(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)

	//1. Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())
	//2. Create project
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	//3. Create Pipeline
	pipelineKey := sdk.RandomString(10)
	pip := &sdk.Pipeline{
		Name:       pipelineKey,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, pip, nil))

	vars := map[string]string{
		"key":             proj.Key,
		"permPipelineKey": pip.Name,
	}
	uri := router.GetRoute("DELETE", api.deletePipelineHandler, vars)
	test.NotEmpty(t, uri)

	req, err := http.NewRequest("DELETE", uri, nil)
	if err != nil {
		t.FailNow()
		return
	}
	assets.AuthentifyRequest(t, req, u, pass)

	//8. Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 204, w.Code)

	pip2, err := pipeline.LoadPipeline(db, proj.Key, pip.Name, false)
	assert.Nil(t, pip2)
	assert.Error(t, err)
}

func Test_deletePipelineHandlerShouldReturnError(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)

	//1. Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())
	//2. Create project
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	//3. Create Pipeline
	pipelineKey := sdk.RandomString(10)
	pip := &sdk.Pipeline{
		Name:       pipelineKey,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, pip, nil))

	wf := sdk.Workflow{
		Name:       sdk.RandomString(10),
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			PipelineName: pip.Name,
			PipelineID:   pip.ID,
			Name:         "root",
		},
	}

	proj.Pipelines = append(proj.Pipelines, *pip)

	test.NoError(t, workflow.Insert(api.mustDB(), api.Cache, &wf, proj, u))

	vars := map[string]string{
		"key":             proj.Key,
		"permPipelineKey": pip.Name,
	}
	uri := router.GetRoute("DELETE", api.deletePipelineHandler, vars)
	test.NotEmpty(t, uri)

	req, err := http.NewRequest("DELETE", uri, nil)
	if err != nil {
		t.FailNow()
		return
	}
	assets.AuthentifyRequest(t, req, u, pass)

	//8. Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 400, w.Code)
}
