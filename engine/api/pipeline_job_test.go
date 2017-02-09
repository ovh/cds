package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func TestAddJobHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/TestAddJobHandler"}
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

	//4. Add Stage
	stage := &sdk.Stage{
		BuildOrder: 1,
		Enabled:    true,
		Name:       "Stage1",
		PipelineID: pip.ID,
	}
	test.NoError(t, pipeline.InsertStage(db, stage))
	assert.NotZero(t, stage.ID)

	// 5. Prepare the request
	addJobRequest := sdk.Job{
		Enabled:         true,
		PipelineStageID: stage.ID,
		Action: sdk.Action{
			Name: "myJob",
		},
	}
	jsonBody, _ := json.Marshal(addJobRequest)
	body := bytes.NewBuffer(jsonBody)

	vars := map[string]string{
		"key":             proj.Key,
		"permPipelineKey": pip.Name,
		"stageID":         strconv.FormatInt(stage.ID, 10),
	}

	uri := router.getRoute("POST", addJobToStageHandler, vars)
	test.NotEmpty(t, uri)

	req, _ := http.NewRequest("POST", uri, body)
	assets.AuthentifyRequest(t, req, u, pass)

	//6. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	pipResult := &sdk.Pipeline{}
	json.Unmarshal(res, &pipResult)
	assert.Equal(t, len(pipResult.Stages), 1)
	assert.Equal(t, len(pipResult.Stages[0].Jobs), 1)
	assert.Equal(t, pipResult.Stages[0].Jobs[0].Action.Name, addJobRequest.Action.Name)
}

func TestUpdateJobHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/TestUpdateJobHandler"}
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

	//4. Add Stage
	stage := &sdk.Stage{
		BuildOrder: 1,
		Enabled:    true,
		Name:       "Stage1",
		PipelineID: pip.ID,
	}
	test.NoError(t, pipeline.InsertStage(db, stage))

	//5. Prepare the request
	job := &sdk.Job{
		Enabled:         true,
		PipelineStageID: stage.ID,
		Action: sdk.Action{
			Name: "myJob",
		},
	}
	test.NoError(t, pipeline.InsertJob(db, job, stage.ID, pip))
	assert.NotZero(t, job.PipelineActionID)
	assert.NotZero(t, job.Action.ID)

	// 6. Prepare the request
	addJobRequest := sdk.Job{
		Enabled:          true,
		PipelineStageID:  stage.ID,
		PipelineActionID: job.PipelineActionID,
		Action: sdk.Action{
			ID:   job.Action.ID,
			Name: "myJobUpdated",
		},
	}
	jsonBody, _ := json.Marshal(addJobRequest)
	body := bytes.NewBuffer(jsonBody)

	vars := map[string]string{
		"key":             proj.Key,
		"permPipelineKey": pip.Name,
		"stageID":         strconv.FormatInt(stage.ID, 10),
		"jobID":           strconv.FormatInt(job.PipelineActionID, 10),
	}

	uri := router.getRoute("PUT", updateJobHandler, vars)
	test.NotEmpty(t, uri)

	req, _ := http.NewRequest("PUT", uri, body)
	assets.AuthentifyRequest(t, req, u, pass)

	//7. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	pipResult := &sdk.Pipeline{}
	json.Unmarshal(res, &pipResult)
	assert.Equal(t, len(pipResult.Stages), 1)
	assert.Equal(t, len(pipResult.Stages[0].Jobs), 1)
	assert.Equal(t, pipResult.Stages[0].Jobs[0].Action.Name, addJobRequest.Action.Name)
}

func TestDeleteJobHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/TestDeleteJobHandler"}
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

	//4. Add Stage
	stage := &sdk.Stage{
		BuildOrder: 1,
		Enabled:    true,
		Name:       "Stage1",
		PipelineID: pip.ID,
	}
	test.NoError(t, pipeline.InsertStage(db, stage))

	//5. Prepare the request
	job := &sdk.Job{
		Enabled:         true,
		PipelineStageID: stage.ID,
		Action: sdk.Action{
			Name: "myJob",
		},
	}
	test.NoError(t, pipeline.InsertJob(db, job, stage.ID, pip))
	assert.NotZero(t, job.PipelineActionID)
	assert.NotZero(t, job.Action.ID)

	vars := map[string]string{
		"key":             proj.Key,
		"permPipelineKey": pip.Name,
		"stageID":         strconv.FormatInt(stage.ID, 10),
		"jobID":           strconv.FormatInt(job.PipelineActionID, 10),
	}

	uri := router.getRoute("DELETE", deleteJobHandler, vars)
	test.NotEmpty(t, uri)

	req, _ := http.NewRequest("DELETE", uri, nil)
	assets.AuthentifyRequest(t, req, u, pass)

	//7. Do the request
	w := httptest.NewRecorder()
	router.mux.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)
	res, _ := ioutil.ReadAll(w.Body)
	pipResult := &sdk.Pipeline{}
	json.Unmarshal(res, &pipResult)
	assert.Equal(t, len(pipResult.Stages), 1)
	assert.Equal(t, len(pipResult.Stages[0].Jobs), 0)
}
