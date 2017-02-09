package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-gorp/gorp"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/sdk"
)

func insertTestPipeline(db *gorp.DbMap, t *testing.T, name string) (*sdk.Project, *sdk.Pipeline, *sdk.Application) {
	pkey := assets.RandomString(t, 10)
	projectFoo := assets.InsertTestProject(t, db, pkey, pkey)

	p := &sdk.Pipeline{
		Name:      name,
		ProjectID: projectFoo.ID,
		Type:      sdk.BuildPipeline,
	}

	app := &sdk.Application{
		Name: "App1",
	}

	test.NoError(t, application.InsertApplication(db, projectFoo, app))
	test.NoError(t, pipeline.InsertPipeline(db, p))

	return projectFoo, p, app
}

func Test_runPipelineHandler(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)

	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/Test_runPipelineHandler"}
	router.init()

	//1. Create admin user
	u, pass := assets.InsertAdminUser(t, db)
	//2. Create project
	proj := assets.InsertTestProject(t, db, assets.RandomString(t, 10), assets.RandomString(t, 10))
	test.NotNil(t, proj)

	//3. Create Pipeline
	pipelineKey := assets.RandomString(t, 10)
	pip := &sdk.Pipeline{
		Name:       pipelineKey,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(db, pip))

	//4. Insert Application
	appName := assets.RandomString(t, 10)
	app := &sdk.Application{
		Name: appName,
	}
	test.NoError(t, application.InsertApplication(db, proj, app))

	//5. Attach pipeline to application
	test.NoError(t, application.AttachPipeline(db, app.ID, pip.ID))

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
	uri := router.getRoute("POST", runPipelineHandler, vars)
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
	router.mux.ServeHTTP(w, req)

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
	db := test.SetupPG(t, bootstrap.InitiliazeDB)

	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/Test_runPipelineHandler"}
	router.init()

	//1. Create admin user
	u, pass := assets.InsertAdminUser(t, db)

	//2. Create project
	proj := assets.InsertTestProject(t, db, assets.RandomString(t, 10), assets.RandomString(t, 10))
	test.NotNil(t, proj)

	//3. Create Pipeline
	pipelineKey := assets.RandomString(t, 10)
	pip := &sdk.Pipeline{
		Name:       pipelineKey,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	test.NoError(t, pipeline.InsertPipeline(db, pip))

	//4. Insert Application
	appName := assets.RandomString(t, 10)
	app := &sdk.Application{
		Name: appName,
	}
	test.NoError(t, application.InsertApplication(db, proj, app))

	//5. Attach pipeline to application
	test.NoError(t, application.AttachPipeline(db, app.ID, pip.ID))

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
	uri := router.getRoute("POST", runPipelineHandler, vars)
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
	router.mux.ServeHTTP(w, req)

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
	err = pipeline.UpdatePipelineBuildStatusAndStage(db, &pb, sdk.StatusSuccess)
	test.NoError(t, err)

	//10. Create another Pipeline
	pip2 := &sdk.Pipeline{
		Name:       assets.RandomString(t, 10),
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	err = pipeline.InsertPipeline(db, pip2)
	test.NoError(t, err)

	//11. Insert another Application
	app2 := &sdk.Application{
		Name: assets.RandomString(t, 10),
	}
	err = application.InsertApplication(db, proj, app2)

	//12. Attach pipeline to application
	err = application.AttachPipeline(db, app2.ID, pip2.ID)
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
	tx, _ := db.Begin()
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
	uri = router.getRoute("POST", runPipelineWithLastParentHandler, vars)
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
	router.mux.ServeHTTP(w, req)

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
