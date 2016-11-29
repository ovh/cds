package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/sessionstore"
	test "github.com/ovh/cds/engine/api/testwithdb"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/sdk"
)

func TestGetApplicationWithTriggersHandler(t *testing.T) {
	if test.DBDriver == "" {
		t.SkipNow()
		return
	}
	db, err := test.SetupPG(t)
	assert.NoError(t, err)

	authDriver, _ := auth.GetDriver("local", nil, sessionstore.Options{Mode: "local"})
	router = &Router{authDriver, mux.NewRouter(), "/TestGetApplicationHandler"}
	router.init()

	//1. Create admin user
	u, pass, err := test.InsertAdminUser(t, db)
	assert.NoError(t, err)

	//2. Create project
	proj, _ := test.InsertTestProject(t, db, test.RandomString(t, 10), test.RandomString(t, 10))
	assert.NotNil(t, proj)
	if proj == nil {
		t.Fail()
		return
	}

	//3. Create Pipeline 1
	pipelineKey := test.RandomString(t, 10)
	pip1 := &sdk.Pipeline{
		Name:       pipelineKey,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	err = pipeline.InsertPipeline(db, pip1)
	assert.NoError(t, err)

	//4. Create Pipeline 2
	pipelineKey = test.RandomString(t, 10)
	pip2 := &sdk.Pipeline{
		Name:       pipelineKey,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	err = pipeline.InsertPipeline(db, pip2)
	assert.NoError(t, err)

	//5. Create Pipeline 3
	pipelineKey = test.RandomString(t, 10)
	pip3 := &sdk.Pipeline{
		Name:       pipelineKey,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	err = pipeline.InsertPipeline(db, pip3)
	assert.NoError(t, err)

	// 6. Create application
	appName := test.RandomString(t, 10)
	app := &sdk.Application{
		Name: appName,
	}
	err = application.InsertApplication(db, proj, app)
	assert.NoError(t, err)

	// 7. Attach pipeline to application
	err = application.AttachPipeline(db, app.ID, pip1.ID)
	assert.NoError(t, err)
	err = application.AttachPipeline(db, app.ID, pip2.ID)
	assert.NoError(t, err)
	err = application.AttachPipeline(db, app.ID, pip3.ID)
	assert.NoError(t, err)

	// 8. Create Trigger between pip1 and pip2
	t1 := &sdk.PipelineTrigger{
		SrcProject:      *proj,
		SrcApplication:  *app,
		SrcPipeline:     *pip1,
		DestProject:     *proj,
		DestApplication: *app,
		DestPipeline:    *pip2,
	}
	err = trigger.InsertTrigger(db, t1)
	assert.NoError(t, err)

	// 8. Create Trigger between pip2 and pip3
	t2 := &sdk.PipelineTrigger{
		SrcProject:      *proj,
		SrcApplication:  *app,
		SrcPipeline:     *pip2,
		DestProject:     *proj,
		DestApplication: *app,
		DestPipeline:    *pip3,
	}
	err = trigger.InsertTrigger(db, t2)
	assert.NoError(t, err)

	// 9. Prepare the request
	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
	}

	uri := fmt.Sprintf("%s?withTriggers=true", router.getRoute("GET", getApplicationHandler, vars))
	if uri == "" {
		t.Fail()
		return
	}
	req, err := http.NewRequest("GET", uri, nil)
	test.AuthentifyRequest(t, req, u, pass)

	//10. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	appResult := &sdk.Application{}
	json.Unmarshal(res, &appResult)

	assert.Equal(t, appResult.Name, appName)
	assert.Equal(t, len(appResult.Pipelines), 3)

	checkPip1 := false
	checkPip2 := false
	checkPip3 := false
	for _, ap := range appResult.Pipelines {
		switch ap.Pipeline.Name {
		case pip1.Name:
			checkPip1 = true
			assert.Equal(t, len(ap.Triggers), 1)
			assert.Equal(t, ap.Triggers[0].ID, t1.ID)
		case pip2.Name:
			checkPip2 = true
			assert.Equal(t, len(ap.Triggers), 2)
			assert.Equal(t, ap.Triggers[0].ID, t1.ID)
			assert.Equal(t, ap.Triggers[1].ID, t2.ID)
		case pip3.Name:
			checkPip3 = true
			assert.Equal(t, len(ap.Triggers), 1)
			assert.Equal(t, ap.Triggers[0].ID, t2.ID)
		}
	}
	assert.Equal(t, checkPip1, true)
	assert.Equal(t, checkPip2, true)
	assert.Equal(t, checkPip3, true)
}
