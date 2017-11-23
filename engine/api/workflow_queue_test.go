package api

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
	"github.com/ovh/venom"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/token"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

type testRunWorkflowCtx struct {
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

func testRunWorkflow(t *testing.T, api *API, router *Router, db *gorp.DbMap) testRunWorkflowCtx {
	u, pass := assets.InsertAdminUser(api.mustDB())
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)
	group.InsertUserInGroup(api.mustDB(), proj.ProjectGroups[0].Group.ID, u.ID, true)
	u.Groups = append(u.Groups, proj.ProjectGroups[0].Group)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
		Type:       sdk.BuildPipeline,
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), proj, &pip, u))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	pipeline.InsertStage(api.mustDB(), s)
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
			Actions: []sdk.Action{
				sdk.NewScriptAction("echo lol"),
			},
		},
	}
	pipeline.InsertJob(api.mustDB(), j, s.ID, &pip)
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

	proj2, errP := project.Load(api.mustDB(), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines)
	test.NoError(t, errP)

	test.NoError(t, workflow.Insert(api.mustDB(), api.Cache, &w, proj2, u))
	w1, err := workflow.Load(api.mustDB(), api.Cache, key, "test_1", u)
	test.NoError(t, err)

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
	}
	uri := router.GetRoute("POST", api.postWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)

	opts := &sdk.WorkflowRunPostHandlerOption{}
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, opts)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 202, rec.Code)

	wr := &sdk.WorkflowRun{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), wr))
	assert.Equal(t, int64(1), wr.Number)

	if t.Failed() {
		t.FailNow()
	}

	return testRunWorkflowCtx{
		user:     u,
		password: pass,
		project:  proj,
		workflow: w1,
		run:      wr,
	}
}

func testGetWorkflowJob(t *testing.T, api *API, router *Router, ctx *testRunWorkflowCtx) {
	uri := router.GetRoute("GET", api.getWorkflowJobQueueHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, ctx.user, ctx.password, "GET", uri, nil)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	jobs := []sdk.WorkflowNodeJobRun{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &jobs))
	assert.Len(t, jobs, 1)

	if t.Failed() {
		t.FailNow()
	}

	ctx.job = &jobs[0]
}

func testRegisterWorker(t *testing.T, api *API, router *Router, ctx *testRunWorkflowCtx) {
	var err error
	//Generate token
	ctx.workerToken, err = token.GenerateToken()
	test.NoError(t, err)
	//Insert token
	test.NoError(t, token.InsertToken(api.mustDB(), ctx.user.Groups[0].ID, ctx.workerToken, sdk.Persistent))
	//Register the worker
	params := &worker.RegistrationForm{
		Name:  sdk.RandomString(10),
		Token: ctx.workerToken,
	}
	ctx.worker, err = worker.RegisterWorker(api.mustDB(), params.Name, params.Token, params.ModelID, nil, params.BinaryCapabilities)
	test.NoError(t, err)
}

func testRegisterHatchery(t *testing.T, api *API, router *Router, ctx *testRunWorkflowCtx) {
	//Generate token
	tk, err := token.GenerateToken()
	test.NoError(t, err)
	//Insert token
	test.NoError(t, token.InsertToken(api.mustDB(), ctx.user.Groups[0].ID, tk, sdk.Persistent))

	ctx.hatchery = &sdk.Hatchery{
		UID:      tk,
		LastBeat: time.Now(),
		Name:     sdk.RandomString(10),
		GroupID:  ctx.user.Groups[0].ID,
	}

	err = hatchery.InsertHatchery(api.mustDB(), ctx.hatchery)
	test.NoError(t, err)
}

func TestGetWorkflowJobQueueHandler(t *testing.T) {
	api, db, router := newTestAPI(t)
	ctx := testRunWorkflow(t, api, router, db)
	testGetWorkflowJob(t, api, router, &ctx)
	assert.NotNil(t, ctx.job)
}

func Test_postWorkflowJobRequirementsErrorHandler(t *testing.T) {
	api, db, router := newTestAPI(t)
	ctx := testRunWorkflow(t, api, router, db)

	uri := router.GetRoute("POST", api.postWorkflowJobRequirementsErrorHandler, nil)
	test.NotEmpty(t, uri)

	//This will check the needWorker() auth
	req := assets.NewAuthentifiedRequest(t, ctx.user, ctx.password, "POST", uri, "This is a requirement log error")
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 403, rec.Code)

	//Register the worker
	testRegisterWorker(t, api, router, &ctx)

	//This call must work
	req = assets.NewAuthentifiedRequestFromWorker(t, ctx.worker, "POST", uri, "This is a requirement log error")
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

}
func Test_postTakeWorkflowJobHandler(t *testing.T) {
	api, db, router := newTestAPI(t)
	ctx := testRunWorkflow(t, api, router, db)
	testGetWorkflowJob(t, api, router, &ctx)
	assert.NotNil(t, ctx.job)

	takeForm := worker.TakeForm{
		BookedJobID: ctx.job.ID,
		Time:        time.Now(),
	}

	//Prepare request
	vars := map[string]string{
		"key":              ctx.project.Key,
		"permWorkflowName": ctx.workflow.Name,
		"id":               fmt.Sprintf("%d", ctx.job.ID),
	}

	//Register the worker
	testRegisterWorker(t, api, router, &ctx)

	uri := router.GetRoute("POST", api.postTakeWorkflowJobHandler, vars)
	test.NotEmpty(t, uri)

	//This will check the needWorker() auth
	req := assets.NewAuthentifiedRequest(t, ctx.user, ctx.password, "POST", uri, takeForm)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 403, rec.Code)

	//This call must work
	req = assets.NewAuthentifiedRequestFromWorker(t, ctx.worker, "POST", uri, takeForm)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	run, err := workflow.LoadNodeJobRun(api.mustDB(), api.Cache, ctx.job.ID)
	test.NoError(t, err)
	assert.Equal(t, "Building", run.Status)

}
func Test_postBookWorkflowJobHandler(t *testing.T) {
	api, db, router := newTestAPI(t)
	ctx := testRunWorkflow(t, api, router, db)
	testGetWorkflowJob(t, api, router, &ctx)
	assert.NotNil(t, ctx.job)

	//Prepare request
	vars := map[string]string{
		"key":              ctx.project.Key,
		"permWorkflowName": ctx.workflow.Name,
		"id":               fmt.Sprintf("%d", ctx.job.ID),
	}

	//Register the hatchery
	testRegisterHatchery(t, api, router, &ctx)

	//TakeBook
	uri := router.GetRoute("POST", api.postBookWorkflowJobHandler, vars)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequestFromHatchery(t, ctx.hatchery, "POST", uri, nil)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

}

func Test_postWorkflowJobResultHandler(t *testing.T) {
	api, db, router := newTestAPI(t)
	ctx := testRunWorkflow(t, api, router, db)
	testGetWorkflowJob(t, api, router, &ctx)
	assert.NotNil(t, ctx.job)

	//Prepare request
	vars := map[string]string{
		"key":              ctx.project.Key,
		"permWorkflowName": ctx.workflow.Name,
		"id":               fmt.Sprintf("%d", ctx.job.ID),
	}

	//Register the worker
	testRegisterWorker(t, api, router, &ctx)

	//Take
	uri := router.GetRoute("POST", api.postTakeWorkflowJobHandler, vars)
	test.NotEmpty(t, uri)

	takeForm := worker.TakeForm{
		BookedJobID: ctx.job.ID,
		Time:        time.Now(),
	}

	req := assets.NewAuthentifiedRequestFromWorker(t, ctx.worker, "POST", uri, takeForm)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	//Send logs
	logs := sdk.Log{
		Val: "This is a log",
	}

	uri = router.Prefix + fmt.Sprintf("/queue/workflows/%d/log", ctx.job.ID)

	req = assets.NewAuthentifiedRequestFromWorker(t, ctx.worker, "POST", uri, logs)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 202, rec.Code)

	now, _ := ptypes.TimestampProto(time.Now())

	//Send result
	res := sdk.Result{
		Duration:   "10",
		Status:     sdk.StatusSuccess.String(),
		RemoteTime: now,
		BuildID:    ctx.job.ID,
	}

	vars = map[string]string{
		"key":              ctx.project.Key,
		"permWorkflowName": ctx.workflow.Name,
		"permID":           fmt.Sprintf("%d", ctx.job.ID),
	}

	uri = router.GetRoute("POST", api.postWorkflowJobResultHandler, vars)
	test.NotEmpty(t, uri)

	req = assets.NewAuthentifiedRequestFromWorker(t, ctx.worker, "POST", uri, res)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)
}

func Test_postWorkflowJobTestsResultsHandler(t *testing.T) {
	api, db, router := newTestAPI(t)
	ctx := testRunWorkflow(t, api, router, db)
	testGetWorkflowJob(t, api, router, &ctx)
	assert.NotNil(t, ctx.job)

	//Prepare request
	vars := map[string]string{
		"key":              ctx.project.Key,
		"permWorkflowName": ctx.workflow.Name,
		"id":               fmt.Sprintf("%d", ctx.job.ID),
	}

	//Register the worker
	testRegisterWorker(t, api, router, &ctx)
	//Register the hatchery
	testRegisterHatchery(t, api, router, &ctx)

	//Send spawninfo
	info := []sdk.SpawnInfo{}
	uri := router.Prefix + fmt.Sprintf("/queue/workflows/%d/spawn/infos", ctx.job.ID)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequestFromHatchery(t, ctx.hatchery, "POST", uri, info)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 202, rec.Code)

	//spawn
	uri = router.GetRoute("POST", api.postTakeWorkflowJobHandler, vars)
	test.NotEmpty(t, uri)

	takeForm := worker.TakeForm{
		BookedJobID: ctx.job.ID,
		Time:        time.Now(),
	}

	req = assets.NewAuthentifiedRequestFromWorker(t, ctx.worker, "POST", uri, takeForm)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
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
	uri = router.GetRoute("POST", api.postWorkflowJobTestsResultsHandler, vars)
	test.NotEmpty(t, uri)

	req = assets.NewAuthentifiedRequestFromWorker(t, ctx.worker, "POST", uri, tests)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	step := sdk.StepStatus{
		Status:    sdk.StatusSuccess.String(),
		StepOrder: 0,
	}

	uri = router.GetRoute("POST", api.postWorkflowJobStepStatusHandler, vars)
	test.NotEmpty(t, uri)

	req = assets.NewAuthentifiedRequestFromWorker(t, ctx.worker, "POST", uri, step)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	wNodeJobRun, errJ := workflow.LoadNodeJobRun(api.mustDB(), api.Cache, ctx.job.ID)
	test.NoError(t, errJ)
	nodeRun, errN := workflow.LoadNodeRunByID(api.mustDB(), wNodeJobRun.WorkflowNodeRunID)
	test.NoError(t, errN)

	assert.NotNil(t, nodeRun.Tests)
	assert.Equal(t, 2, nodeRun.Tests.Total)
	//t.Logf("%+v", nodeRun.Tests)
}

func Test_postWorkflowJobVariableHandler(t *testing.T) {
	api, db, router := newTestAPI(t)
	ctx := testRunWorkflow(t, api, router, db)
	testGetWorkflowJob(t, api, router, &ctx)
	assert.NotNil(t, ctx.job)

	//Prepare request
	vars := map[string]string{
		"key":              ctx.project.Key,
		"permWorkflowName": ctx.workflow.Name,
		"id":               fmt.Sprintf("%d", ctx.job.ID),
	}

	//Register the worker
	testRegisterWorker(t, api, router, &ctx)

	//Take
	uri := router.GetRoute("POST", api.postTakeWorkflowJobHandler, vars)
	test.NotEmpty(t, uri)

	takeForm := worker.TakeForm{
		BookedJobID: ctx.job.ID,
		Time:        time.Now(),
	}

	req := assets.NewAuthentifiedRequestFromWorker(t, ctx.worker, "POST", uri, takeForm)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	vars = map[string]string{
		"key":              ctx.project.Key,
		"permWorkflowName": ctx.workflow.Name,
		"permID":           fmt.Sprintf("%d", ctx.job.ID),
	}

	//Send result
	v := sdk.Variable{
		Name:  "var",
		Value: "value",
	}

	uri = router.GetRoute("POST", api.postWorkflowJobVariableHandler, vars)
	test.NotEmpty(t, uri)

	req = assets.NewAuthentifiedRequestFromWorker(t, ctx.worker, "POST", uri, v)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)
}

func Test_postWorkflowJobArtifactHandler(t *testing.T) {
	api, db, router := newTestAPI(t)
	ctx := testRunWorkflow(t, api, router, db)
	testGetWorkflowJob(t, api, router, &ctx)
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
		"key":              ctx.project.Key,
		"permWorkflowName": ctx.workflow.Name,
		"id":               fmt.Sprintf("%d", ctx.job.ID),
	}

	//Register the worker
	testRegisterWorker(t, api, router, &ctx)

	//Take
	uri := router.GetRoute("POST", api.postTakeWorkflowJobHandler, vars)
	test.NotEmpty(t, uri)

	takeForm := worker.TakeForm{
		BookedJobID: ctx.job.ID,
		Time:        time.Now(),
	}

	req := assets.NewAuthentifiedRequestFromWorker(t, ctx.worker, "POST", uri, takeForm)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	vars = map[string]string{
		"tag":    "latest",
		"permID": fmt.Sprintf("%d", ctx.job.ID),
	}

	uri = router.GetRoute("POST", api.postWorkflowJobArtifactHandler, vars)
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
	req = assets.NewAuthentifiedMultipartRequestFromWorker(t, ctx.worker, "POST", uri, path.Join(os.TempDir(), "myartifact"), "myartifact", params)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	time.Sleep(1 * time.Second)

	wNodeJobRun, errJ := workflow.LoadNodeJobRun(api.mustDB(), api.Cache, ctx.job.ID)
	test.NoError(t, errJ)

	updatedNodeRun, errN2 := workflow.LoadNodeRunByID(api.mustDB(), wNodeJobRun.WorkflowNodeRunID)
	test.NoError(t, errN2)

	assert.NotNil(t, updatedNodeRun.Artifacts)
	assert.Equal(t, 1, len(updatedNodeRun.Artifacts))

	//Prepare request
	vars = map[string]string{
		"key":              ctx.project.Key,
		"permWorkflowName": ctx.workflow.Name,
		"number":           fmt.Sprintf("%d", updatedNodeRun.Number),
		"nodeRunID":        fmt.Sprintf("%d", wNodeJobRun.WorkflowNodeRunID),
	}
	uri = router.GetRoute("GET", api.getWorkflowNodeRunArtifactsHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, ctx.user, ctx.password, "GET", uri, nil)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	var arts []sdk.WorkflowNodeRunArtifact
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &arts))
	assert.Equal(t, 1, len(arts))
	assert.Equal(t, "myartifact", arts[0].Name)

	// Download artifact
	//Prepare request
	vars = map[string]string{
		"key":              ctx.project.Key,
		"permWorkflowName": ctx.workflow.Name,
		"artifactId":       fmt.Sprintf("%d", arts[0].ID),
	}
	uri = router.GetRoute("GET", api.getDownloadArtifactHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, ctx.user, ctx.password, "GET", uri, nil)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)

	resp := rec.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	assert.Equal(t, 200, rec.Code)
	assert.Equal(t, "Hi, I am foo", string(body))
}
func TestGetWorkflowJobArtifactsHandler(t *testing.T) {
	//api, db, router := newTestAPI(t)
	//ctx := runWorkflow(t, db, "Test_postWorkflowJobRequirementsErrorHandler")
}
func Test_getDownloadArtifactHandler(t *testing.T) {
	//api, db, router := newTestAPI(t)
	//ctx := runWorkflow(t, db, "Test_postWorkflowJobRequirementsErrorHandler")
}
