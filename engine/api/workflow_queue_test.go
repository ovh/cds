package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"path"
	"testing"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/golang/protobuf/ptypes"
	"github.com/gorilla/mux"
	"github.com/ovh/venom"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/token"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

type test_runWorkflowCtx struct {
	user        *sdk.User
	password    string
	project     *sdk.Project
	workflow    *sdk.Workflow
	run         *sdk.WorkflowRun
	job         *sdk.WorkflowNodeJobRun
	worker      *sdk.Worker
	workerToken string
	hatchery    *sdk.Hatchery
}

func test_runWorkflow(t *testing.T, db *gorp.DbMap, testName string) test_runWorkflowCtx {
	u, pass := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, key, key, u)
	group.InsertUserInGroup(db, proj.ProjectGroups[0].Group.ID, u.ID, true)
	u.Groups = append(u.Groups, proj.ProjectGroups[0].Group)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(db, proj, &pip, u))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	pipeline.InsertStage(db, s)
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
			Actions: []sdk.Action{
				sdk.NewScriptAction("echo lol"),
			},
		},
	}
	pipeline.InsertJob(db, j, s.ID, &pip)
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Pipeline: pip,
		},
	}

	test.NoError(t, workflow.Insert(db, &w, u))
	w1, err := workflow.Load(db, key, "test_1", u)
	test.NoError(t, err)

	// Init router
	router = newRouter(auth.TestLocalAuth(t), mux.NewRouter(), testName)
	router.init()
	//Prepare request
	vars := map[string]string{
		"permProjectKey": proj.Key,
		"workflowName":   w1.Name,
	}
	uri := router.getRoute("POST", postWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)

	opts := &postWorkflowRunHandlerOption{}
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, opts)

	//Do the request
	rec := httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	wr := &sdk.WorkflowRun{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), wr))
	assert.Equal(t, int64(1), wr.Number)

	if t.Failed() {
		t.FailNow()
	}

	c, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	workflow.Scheduler(c, func() *gorp.DbMap { return db })
	time.Sleep(1 * time.Second)

	return test_runWorkflowCtx{
		user:     u,
		password: pass,
		project:  proj,
		workflow: w1,
		run:      wr,
	}
}

func test_getWorkflowJob(t *testing.T, db *gorp.DbMap, ctx *test_runWorkflowCtx) {
	uri := router.getRoute("GET", getWorkflowJobQueueHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, ctx.user, ctx.password, "GET", uri, nil)
	rec := httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	jobs := []sdk.WorkflowNodeJobRun{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &jobs))
	assert.Len(t, jobs, 1)

	if t.Failed() {
		t.FailNow()
	}

	ctx.job = &jobs[0]
}

func test_registerWorker(t *testing.T, db *gorp.DbMap, ctx *test_runWorkflowCtx) {
	var err error
	//Generate token
	ctx.workerToken, err = token.GenerateToken()
	test.NoError(t, err)
	//Insert token
	test.NoError(t, token.InsertToken(db, ctx.user.Groups[0].ID, ctx.workerToken, sdk.Persistent))
	//Register the worker
	params := &worker.RegistrationForm{
		Name:  sdk.RandomString(10),
		Token: ctx.workerToken,
	}
	ctx.worker, err = worker.RegisterWorker(db, params.Name, params.Token, params.Model, nil, params.BinaryCapabilities)
	test.NoError(t, err)
}

func test_registerHatchery(t *testing.T, db *gorp.DbMap, ctx *test_runWorkflowCtx) {
	//Generate token
	tk, err := token.GenerateToken()
	test.NoError(t, err)
	//Insert token
	test.NoError(t, token.InsertToken(db, ctx.user.Groups[0].ID, tk, sdk.Persistent))

	ctx.hatchery = &sdk.Hatchery{
		UID:      tk,
		LastBeat: time.Now(),
		Name:     sdk.RandomString(10),
		GroupID:  ctx.user.Groups[0].ID,
	}

	err = hatchery.InsertHatchery(db, ctx.hatchery)
	test.NoError(t, err)
}

func Test_getWorkflowJobQueueHandler(t *testing.T) {
	db := test.SetupPG(t)
	ctx := test_runWorkflow(t, db, "/Test_getWorkflowJobQueueHandler")
	test_getWorkflowJob(t, db, &ctx)
	assert.NotNil(t, ctx.job)
}

func Test_postWorkflowJobRequirementsErrorHandler(t *testing.T) {
	db := test.SetupPG(t)
	ctx := test_runWorkflow(t, db, "/Test_postWorkflowJobRequirementsErrorHandler")

	uri := router.getRoute("POST", postWorkflowJobRequirementsErrorHandler, nil)
	test.NotEmpty(t, uri)

	//This will check the needWorker() auth
	req := assets.NewAuthentifiedRequest(t, ctx.user, ctx.password, "POST", uri, "This is a requirement log error")
	rec := httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 403, rec.Code)

	//Register the worker
	test_registerWorker(t, db, &ctx)

	//This call must work
	req = assets.NewAuthentifiedRequestFromWorker(t, ctx.worker, "POST", uri, "This is a requirement log error")
	rec = httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

}
func Test_postTakeWorkflowJobHandler(t *testing.T) {
	db := test.SetupPG(t)
	ctx := test_runWorkflow(t, db, "/Test_postTakeWorkflowJobHandler")
	test_getWorkflowJob(t, db, &ctx)
	assert.NotNil(t, ctx.job)

	takeForm := worker.TakeForm{
		BookedJobID: ctx.job.ID,
		Time:        time.Now(),
	}

	//Prepare request
	vars := map[string]string{
		"permProjectKey": ctx.project.Key,
		"workflowName":   ctx.workflow.Name,
		"id":             fmt.Sprintf("%d", ctx.job.ID),
	}

	//Register the worker
	test_registerWorker(t, db, &ctx)

	uri := router.getRoute("POST", postTakeWorkflowJobHandler, vars)
	test.NotEmpty(t, uri)

	//This will check the needWorker() auth
	req := assets.NewAuthentifiedRequest(t, ctx.user, ctx.password, "POST", uri, takeForm)
	rec := httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 403, rec.Code)

	//This call must work
	req = assets.NewAuthentifiedRequestFromWorker(t, ctx.worker, "POST", uri, takeForm)
	rec = httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	run, err := workflow.LoadNodeJobRun(db, ctx.job.ID)
	test.NoError(t, err)
	assert.Equal(t, "Building", run.Status)

}
func Test_postBookWorkflowJobHandler(t *testing.T) {
	db := test.SetupPG(t)
	ctx := test_runWorkflow(t, db, "/Test_postBookWorkflowJobHandler")
	test_getWorkflowJob(t, db, &ctx)
	assert.NotNil(t, ctx.job)

	//Prepare request
	vars := map[string]string{
		"permProjectKey": ctx.project.Key,
		"workflowName":   ctx.workflow.Name,
		"id":             fmt.Sprintf("%d", ctx.job.ID),
	}

	//Register the hatchery
	test_registerHatchery(t, db, &ctx)

	//TakeBook
	uri := router.getRoute("POST", postBookWorkflowJobHandler, vars)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequestFromHatchery(t, ctx.hatchery, "POST", uri, nil)
	rec := httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

}

func Test_postWorkflowJobResultHandler(t *testing.T) {
	db := test.SetupPG(t)
	ctx := test_runWorkflow(t, db, "/Test_postWorkflowJobResultHandler")
	test_getWorkflowJob(t, db, &ctx)
	assert.NotNil(t, ctx.job)

	//Prepare request
	vars := map[string]string{
		"permProjectKey": ctx.project.Key,
		"workflowName":   ctx.workflow.Name,
		"id":             fmt.Sprintf("%d", ctx.job.ID),
	}

	//Register the worker
	test_registerWorker(t, db, &ctx)

	//Take
	uri := router.getRoute("POST", postTakeWorkflowJobHandler, vars)
	test.NotEmpty(t, uri)

	takeForm := worker.TakeForm{
		BookedJobID: ctx.job.ID,
		Time:        time.Now(),
	}

	req := assets.NewAuthentifiedRequestFromWorker(t, ctx.worker, "POST", uri, takeForm)
	rec := httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	vars = map[string]string{
		"permProjectKey": ctx.project.Key,
		"workflowName":   ctx.workflow.Name,
		"permID":         fmt.Sprintf("%d", ctx.job.ID),
	}

	//Send logs
	logs := sdk.Log{
		Val: "This is a log",
	}

	uri = router.getRoute("POST", postWorkflowJobLogsHandler, vars)
	test.NotEmpty(t, uri)

	req = assets.NewAuthentifiedRequestFromWorker(t, ctx.worker, "POST", uri, logs)
	rec = httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	now, _ := ptypes.TimestampProto(time.Now())

	//Send result
	res := sdk.Result{
		Duration:   "10",
		Status:     sdk.StatusSuccess.String(),
		RemoteTime: now,
	}

	uri = router.getRoute("POST", postWorkflowJobResultHandler, vars)
	test.NotEmpty(t, uri)

	req = assets.NewAuthentifiedRequestFromWorker(t, ctx.worker, "POST", uri, res)
	rec = httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

}

func Test_postWorkflowJobTestsResultsHandler(t *testing.T) {
	db := test.SetupPG(t)
	ctx := test_runWorkflow(t, db, "/Test_postWorkflowJobTestsResultsHandler")
	test_getWorkflowJob(t, db, &ctx)
	assert.NotNil(t, ctx.job)

	//Prepare request
	vars := map[string]string{
		"permProjectKey": ctx.project.Key,
		"workflowName":   ctx.workflow.Name,
		"id":             fmt.Sprintf("%d", ctx.job.ID),
	}

	//Register the worker
	test_registerWorker(t, db, &ctx)
	//Register the hatchery
	test_registerHatchery(t, db, &ctx)

	//Send spawninfo
	info := []sdk.SpawnInfo{}
	uri := router.getRoute("POST", postSpawnInfosWorkflowJobHandler, vars)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequestFromHatchery(t, ctx.hatchery, "POST", uri, info)
	rec := httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//spawn
	uri = router.getRoute("POST", postTakeWorkflowJobHandler, vars)
	test.NotEmpty(t, uri)

	takeForm := worker.TakeForm{
		BookedJobID: ctx.job.ID,
		Time:        time.Now(),
	}

	req = assets.NewAuthentifiedRequestFromWorker(t, ctx.worker, "POST", uri, takeForm)
	rec = httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	vars = map[string]string{
		"permID": fmt.Sprintf("%d", ctx.job.ID),
	}

	//Send test
	tests := venom.Tests{
		Total:        2,
		TotalKO:      1,
		TotalOK:      1,
		TotalSkipped: 0,
		TestSuites: []venom.TestSuite{
			{
				Total: 1,
				Name:  "TestSuite1",
				TestCases: []venom.TestCase{
					{
						Name:   "TestCase1",
						Status: "OK",
					},
				},
			},
			{
				Total: 1,
				Name:  "TestSuite2",
				TestCases: []venom.TestCase{
					{
						Name:   "TestCase1",
						Status: "KO",
						Failures: []venom.Failure{
							{
								Value:   "Fail",
								Type:    "Assertion error",
								Message: "Error occured",
							},
						},
					},
				},
			},
		},
	}

	uri = router.getRoute("POST", postWorkflowJobTestsResultsHandler, vars)
	test.NotEmpty(t, uri)

	req = assets.NewAuthentifiedRequestFromWorker(t, ctx.worker, "POST", uri, tests)
	rec = httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	step := sdk.StepStatus{
		Status:    sdk.StatusSuccess.String(),
		StepOrder: 0,
	}

	uri = router.getRoute("POST", postWorkflowJobStepStatusHandler, vars)
	test.NotEmpty(t, uri)

	req = assets.NewAuthentifiedRequestFromWorker(t, ctx.worker, "POST", uri, step)
	rec = httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	wNodeJobRun, errJ := workflow.LoadNodeJobRun(db, ctx.job.ID)
	test.NoError(t, errJ)
	nodeRun, errN := workflow.LoadNodeRunByID(db, wNodeJobRun.WorkflowNodeRunID)
	test.NoError(t, errN)

	assert.NotNil(t, nodeRun.Tests)
	assert.Equal(t, 2, nodeRun.Tests.Total)
}
func Test_postWorkflowJobVariableHandler(t *testing.T) {
	db := test.SetupPG(t)
	ctx := test_runWorkflow(t, db, "/Test_postWorkflowJobVariableHandler")
	test_getWorkflowJob(t, db, &ctx)
	assert.NotNil(t, ctx.job)

	//Prepare request
	vars := map[string]string{
		"permProjectKey": ctx.project.Key,
		"workflowName":   ctx.workflow.Name,
		"id":             fmt.Sprintf("%d", ctx.job.ID),
	}

	//Register the worker
	test_registerWorker(t, db, &ctx)

	//Take
	uri := router.getRoute("POST", postTakeWorkflowJobHandler, vars)
	test.NotEmpty(t, uri)

	takeForm := worker.TakeForm{
		BookedJobID: ctx.job.ID,
		Time:        time.Now(),
	}

	req := assets.NewAuthentifiedRequestFromWorker(t, ctx.worker, "POST", uri, takeForm)
	rec := httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	vars = map[string]string{
		"permProjectKey": ctx.project.Key,
		"workflowName":   ctx.workflow.Name,
		"permID":         fmt.Sprintf("%d", ctx.job.ID),
	}

	//Send result
	v := sdk.Variable{
		Name:  "var",
		Value: "value",
	}

	uri = router.getRoute("POST", postWorkflowJobVariableHandler, vars)
	test.NotEmpty(t, uri)

	req = assets.NewAuthentifiedRequestFromWorker(t, ctx.worker, "POST", uri, v)
	rec = httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

}
func Test_postWorkflowJobArtifactHandler(t *testing.T) {
	db := test.SetupPG(t)
	ctx := test_runWorkflow(t, db, "/Test_postWorkflowJobArtifactHandler")
	test_getWorkflowJob(t, db, &ctx)
	assert.NotNil(t, ctx.job)

	// Init store
	cfg := objectstore.Config{
		Kind: objectstore.Filesystem,
		Options: objectstore.ConfigOptions{
			Filesystem: objectstore.ConfigOptionsFilesystem{
				Basedir: path.Join(os.TempDir(), "store"),
			},
		},
	}

	errO := objectstore.Initialize(context.Background(), cfg)
	test.NoError(t, errO)

	//Prepare request
	vars := map[string]string{
		"permProjectKey": ctx.project.Key,
		"workflowName":   ctx.workflow.Name,
		"id":             fmt.Sprintf("%d", ctx.job.ID),
	}

	//Register the worker
	test_registerWorker(t, db, &ctx)

	//Take
	uri := router.getRoute("POST", postTakeWorkflowJobHandler, vars)
	test.NotEmpty(t, uri)

	takeForm := worker.TakeForm{
		BookedJobID: ctx.job.ID,
		Time:        time.Now(),
	}

	req := assets.NewAuthentifiedRequestFromWorker(t, ctx.worker, "POST", uri, takeForm)
	rec := httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	vars = map[string]string{
		"tag":    "latest",
		"permID": fmt.Sprintf("%d", ctx.job.ID),
	}

	uri = router.getRoute("POST", postWorkflowJobArtifactHandler, vars)
	test.NotEmpty(t, uri)

	myartifact, errF := os.Create(path.Join(os.TempDir(), "myartifact"))
	defer os.RemoveAll(path.Join(os.TempDir(), "myartifact"))
	test.NoError(t, errF)
	_, errW := myartifact.Write([]byte("Hi, I am foo"))
	test.NoError(t, errW)

	errClose := myartifact.Close()
	test.NoError(t, errClose)

	params := map[string]string{}
	params["size"] = "12"
	params["perm"] = "7"
	params["md5sum"] = "123"
	req = assets.NewAuthentifiedMultipartRequestFromWorker(t, ctx.worker, "POST", uri, "/tmp/myartifact", "myartifact", params)
	rec = httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	wNodeJobRun, errJ := workflow.LoadNodeJobRun(db, ctx.job.ID)
	test.NoError(t, errJ)

	updatedNodeRun, errN2 := workflow.LoadNodeRunByID(db, wNodeJobRun.WorkflowNodeRunID)
	test.NoError(t, errN2)

	assert.NotNil(t, updatedNodeRun.Artifacts)
	assert.Equal(t, 1, len(updatedNodeRun.Artifacts))

	//Prepare request
	vars = map[string]string{
		"permProjectKey": ctx.project.Key,
		"workflowName":   ctx.workflow.Name,
		"number":         fmt.Sprintf("%d", updatedNodeRun.Number),
		"id":             fmt.Sprintf("%d", wNodeJobRun.WorkflowNodeRunID),
	}
	uri = router.getRoute("GET", getWorkflowNodeRunArtifactsHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, ctx.user, ctx.password, "GET", uri, nil)
	rec = httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	var arts []sdk.WorkflowNodeRunArtifact
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &arts))
	assert.Equal(t, 1, len(arts))
	assert.Equal(t, "myartifact", arts[0].Name)

	// Download artifact
	//Prepare request
	vars = map[string]string{
		"permProjectKey": ctx.project.Key,
		"workflowName":   ctx.workflow.Name,
		"artifactId":     fmt.Sprintf("%d", arts[0].ID),
	}
	uri = router.getRoute("GET", getDownloadArtifactHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, ctx.user, ctx.password, "GET", uri, nil)
	rec = httptest.NewRecorder()
	router.mux.ServeHTTP(rec, req)

	resp := rec.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	assert.Equal(t, 200, rec.Code)
	assert.Equal(t, "Hi, I am foo", string(body))
}
func Test_getWorkflowJobArtifactsHandler(t *testing.T) {
	//db := test.SetupPG(t)
	//ctx := runWorkflow(t, db, "Test_postWorkflowJobRequirementsErrorHandler")
}
func Test_getDownloadArtifactHandler(t *testing.T) {
	//db := test.SetupPG(t)
	//ctx := runWorkflow(t, db, "Test_postWorkflowJobRequirementsErrorHandler")
}
