package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/trigger"
	"github.com/ovh/cds/sdk"
)

func TestGetApplicationWithTriggersHandler(t *testing.T) {
	api, db, router := newTestAPI(t)
	//1. Create admin user
	u, pass := assets.InsertAdminUser(api.mustDB())

	//2. Create project
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10), u)
	test.NotNil(t, proj)

	//3. Create Pipeline 1
	pipelineKey := sdk.RandomString(10)
	pip1 := &sdk.Pipeline{
		Name:       pipelineKey,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	if err := pipeline.InsertPipeline(api.mustDB(), proj, pip1, u); err != nil {
		t.Fatal(err)
	}

	//4. Create Pipeline 2
	pipelineKey = sdk.RandomString(10)
	pip2 := &sdk.Pipeline{
		Name:       pipelineKey,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	if err := pipeline.InsertPipeline(api.mustDB(), proj, pip2, u); err != nil {
		t.Fatal(err)
	}

	//5. Create Pipeline 3
	pipelineKey = sdk.RandomString(10)
	pip3 := &sdk.Pipeline{
		Name:       pipelineKey,
		Type:       sdk.BuildPipeline,
		ProjectKey: proj.Key,
		ProjectID:  proj.ID,
	}
	if err := pipeline.InsertPipeline(api.mustDB(), proj, pip3, u); err != nil {
		t.Fatal(err)
	}
	// 6. Create application
	appName := sdk.RandomString(10)
	app := &sdk.Application{
		Name: appName,
	}
	if err := application.Insert(api.mustDB(), api.Cache, proj, app, u); err != nil {
		t.Fatal(err)
	}

	// 7. Attach pipeline to application
	if _, err := application.AttachPipeline(api.mustDB(), app.ID, pip1.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := application.AttachPipeline(api.mustDB(), app.ID, pip2.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := application.AttachPipeline(api.mustDB(), app.ID, pip3.ID); err != nil {
		t.Fatal(err)
	}

	// 8. Create Trigger between pip1 and pip2
	t1 := &sdk.PipelineTrigger{
		SrcProject:      *proj,
		SrcApplication:  *app,
		SrcPipeline:     *pip1,
		DestProject:     *proj,
		DestApplication: *app,
		DestPipeline:    *pip2,
	}
	if err := trigger.InsertTrigger(api.mustDB(), t1); err != nil {
		t.Fatal(err)
	}

	// 8. Create Trigger between pip2 and pip3
	t2 := &sdk.PipelineTrigger{
		SrcProject:      *proj,
		SrcApplication:  *app,
		SrcPipeline:     *pip2,
		DestProject:     *proj,
		DestApplication: *app,
		DestPipeline:    *pip3,
	}
	if err := trigger.InsertTrigger(api.mustDB(), t2); err != nil {
		t.Fatal(err)
	}

	// 9. Prepare the request
	vars := map[string]string{
		"key": proj.Key,
		"permApplicationName": app.Name,
	}

	uri := fmt.Sprintf("%s?withTriggers=true", router.GetRoute("GET", api.getApplicationHandler, vars))
	test.NotEmpty(t, uri)

	req, _ := http.NewRequest("GET", uri, nil)
	assets.AuthentifyRequest(t, req, u, pass)

	//10. Do the request
	w := httptest.NewRecorder()
	router.Mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)

	t.Log(string(res))

	appResult := &sdk.Application{}
	json.Unmarshal(res, &appResult)

	assert.Equal(t, appName, appResult.Name)
	assert.Equal(t, 3, len(appResult.Pipelines))

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
			var t1Found, t2Found bool
			for _, t := range ap.Triggers {
				if t.ID == t1.ID {
					t1Found = true
					continue
				}
				if t.ID == t2.ID {
					t2Found = true
					continue
				}
			}
			assert.True(t, t1Found, "Trigger %d not found", t1.ID)
			assert.True(t, t2Found, "Trigger %d not found", t2.ID)
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
