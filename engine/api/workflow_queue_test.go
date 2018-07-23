package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/golang/protobuf/ptypes"
	"github.com/ovh/venom"
	"github.com/sguiheux/go-coverage"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/permission"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
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
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), api.Cache, proj, &pip, u))
	test.NoError(t, group.InsertGroupInPipeline(api.mustDB(), pip.ID, proj.ProjectGroups[0].Group.ID, permission.PermissionReadExecute))

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
			PipelineID:   pip.ID,
			PipelineName: pip.Name,
		},
	}

	proj2, errP := project.Load(api.mustDB(), api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups)
	test.NoError(t, errP)

	test.NoError(t, workflow.Insert(api.mustDB(), api.Cache, &w, proj2, u))
	test.NoError(t, workflow.AddGroup(api.mustDB(), &w, proj.ProjectGroups[0]))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, proj, "test_1", u, workflow.LoadOptions{})
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

func testCountGetWorkflowJob(t *testing.T, api *API, router *Router, ctx *testRunWorkflowCtx) {
	uri := router.GetRoute("GET", api.countWorkflowJobQueueHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, ctx.user, ctx.password, "GET", uri, nil)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	count := sdk.WorkflowNodeJobRunCount{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &count))
	assert.True(t, count.Count > 0)

	if t.Failed() {
		t.FailNow()
	}
}

func testGetWorkflowJobAsRegularUser(t *testing.T, api *API, router *Router, ctx *testRunWorkflowCtx) {
	uri := router.GetRoute("GET", api.getWorkflowJobQueueHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewAuthentifiedRequest(t, ctx.user, ctx.password, "GET", uri, nil)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	jobs := []sdk.WorkflowNodeJobRun{}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &jobs))
	assert.True(t, len(jobs) > 1)

	if t.Failed() {
		t.FailNow()
	}

	ctx.job = &jobs[0]
}

func testGetWorkflowJobAsWorker(t *testing.T, api *API, router *Router, ctx *testRunWorkflowCtx) {
	uri := router.GetRoute("GET", api.getWorkflowJobQueueHandler, nil)
	test.NotEmpty(t, uri)

	//Register the worker
	testRegisterWorker(t, api, router, ctx)

	req := assets.NewAuthentifiedRequestFromWorker(t, ctx.worker, "GET", uri, nil)
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

func testGetWorkflowJobAsHatchery(t *testing.T, api *API, router *Router, ctx *testRunWorkflowCtx) {
	uri := router.GetRoute("GET", api.getWorkflowJobQueueHandler, nil)
	test.NotEmpty(t, uri)

	//Register the worker
	testRegisterHatchery(t, api, router, ctx)

	req := assets.NewAuthentifiedRequestFromHatchery(t, ctx.hatchery, "GET", uri, nil)
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
	test.NoError(t, token.InsertToken(api.mustDB(), ctx.user.Groups[0].ID, ctx.workerToken, sdk.Persistent, "", ""))
	//Register the worker
	params := &sdk.WorkerRegistrationForm{
		Name:  sdk.RandomString(10),
		Token: ctx.workerToken,
		OS:    "linux",
		Arch:  "amd64",
	}
	ctx.worker, err = worker.RegisterWorker(api.mustDB(), params.Name, params.Token, params.ModelID, nil, params.BinaryCapabilities, params.OS, params.Arch)
	test.NoError(t, err)
}

func testRegisterHatchery(t *testing.T, api *API, router *Router, ctx *testRunWorkflowCtx) {
	//Generate token
	tk, err := token.GenerateToken()
	test.NoError(t, err)
	//Insert token
	test.NoError(t, token.InsertToken(api.mustDB(), ctx.user.Groups[0].ID, tk, sdk.Persistent, "", ""))

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
	testGetWorkflowJobAsWorker(t, api, router, &ctx)
	assert.NotNil(t, ctx.job)

	// count job in queue
	testCountGetWorkflowJob(t, api, router, &ctx)

	// Get workflow run number

	//Prepare request
	vars := map[string]string{
		"key":              ctx.project.Key,
		"permWorkflowName": ctx.workflow.Name,
	}
	uri := router.GetRoute("GET", api.getWorkflowRunNumHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, ctx.user, ctx.password, "GET", uri, nil)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	var n struct {
		Num int `json:"num"`
	}
	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &n))
	assert.Equal(t, 1, n.Num)

	// Update workflow run number

	//Prepare request
	uri = router.GetRoute("POST", api.postWorkflowRunNumHandler, vars)
	test.NotEmpty(t, uri)

	n.Num = 10
	req = assets.NewAuthentifiedRequest(t, ctx.user, ctx.password, "POST", uri, n)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	uri = router.GetRoute("GET", api.getWorkflowRunNumHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewAuthentifiedRequest(t, ctx.user, ctx.password, "GET", uri, nil)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	test.NoError(t, json.Unmarshal(rec.Body.Bytes(), &n))
	assert.Equal(t, 10, n.Num)
}

func Test_postTakeWorkflowJobHandler(t *testing.T) {
	api, db, router := newTestAPI(t)
	ctx := testRunWorkflow(t, api, router, db)
	testGetWorkflowJobAsWorker(t, api, router, &ctx)
	assert.NotNil(t, ctx.job)

	takeForm := sdk.WorkerTakeForm{
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
	testGetWorkflowJobAsHatchery(t, api, router, &ctx)
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
	testGetWorkflowJobAsWorker(t, api, router, &ctx)
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

	takeForm := sdk.WorkerTakeForm{
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
	assert.Equal(t, 204, rec.Code)
}

func Test_postWorkflowJobTestsResultsHandler(t *testing.T) {
	api, db, router := newTestAPI(t)
	ctx := testRunWorkflow(t, api, router, db)
	testGetWorkflowJobAsWorker(t, api, router, &ctx)
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

	takeForm := sdk.WorkerTakeForm{
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
	assert.Equal(t, 204, rec.Code)

	step := sdk.StepStatus{
		Status:    sdk.StatusSuccess.String(),
		StepOrder: 0,
	}

	uri = router.GetRoute("POST", api.postWorkflowJobStepStatusHandler, vars)
	test.NotEmpty(t, uri)

	req = assets.NewAuthentifiedRequestFromWorker(t, ctx.worker, "POST", uri, step)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 204, rec.Code)

	wNodeJobRun, errJ := workflow.LoadNodeJobRun(api.mustDB(), api.Cache, ctx.job.ID)
	test.NoError(t, errJ)
	nodeRun, errN := workflow.LoadNodeRunByID(api.mustDB(), wNodeJobRun.WorkflowNodeRunID, workflow.LoadRunOptions{WithArtifacts: true, WithTests: true})
	test.NoError(t, errN)

	assert.NotNil(t, nodeRun.Tests)
	assert.Equal(t, 2, nodeRun.Tests.Total)
}

func Test_postWorkflowJobVariableHandler(t *testing.T) {
	api, db, router := newTestAPI(t)
	ctx := testRunWorkflow(t, api, router, db)
	testGetWorkflowJobAsWorker(t, api, router, &ctx)
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

	takeForm := sdk.WorkerTakeForm{
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
	assert.Equal(t, 204, rec.Code)
}

func Test_postWorkflowJobArtifactHandler(t *testing.T) {
	api, db, router := newTestAPI(t)
	ctx := testRunWorkflow(t, api, router, db)
	testGetWorkflowJobAsWorker(t, api, router, &ctx)
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

	takeForm := sdk.WorkerTakeForm{
		BookedJobID: ctx.job.ID,
		Time:        time.Now(),
	}

	req := assets.NewAuthentifiedRequestFromWorker(t, ctx.worker, "POST", uri, takeForm)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	vars = map[string]string{
		"ref":    base64.RawURLEncoding.EncodeToString([]byte("latest")),
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
	params["sha512sum"] = "1234"
	req = assets.NewAuthentifiedMultipartRequestFromWorker(t, ctx.worker, "POST", uri, path.Join(os.TempDir(), "myartifact"), "myartifact", params)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 204, rec.Code)

	time.Sleep(1 * time.Second)

	wNodeJobRun, errJ := workflow.LoadNodeJobRun(api.mustDB(), api.Cache, ctx.job.ID)
	test.NoError(t, errJ)

	updatedNodeRun, errN2 := workflow.LoadNodeRunByID(api.mustDB(), wNodeJobRun.WorkflowNodeRunID, workflow.LoadRunOptions{WithArtifacts: true})
	test.NoError(t, errN2)

	assert.NotNil(t, updatedNodeRun.Artifacts)
	assert.Equal(t, 1, len(updatedNodeRun.Artifacts))

	//Prepare request
	vars = map[string]string{
		"key":              ctx.project.Key,
		"permWorkflowName": ctx.workflow.Name,
		"number":           fmt.Sprintf("%d", updatedNodeRun.Number),
	}
	uri = router.GetRoute("GET", api.getWorkflowRunArtifactsHandler, vars)
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

func TestInsertNewCodeCoverageReport(t *testing.T) {
	api, db, router := newTestAPI(t)

	// Create user
	u, pass := assets.InsertAdminUser(api.mustDB())

	// Create project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key, u)

	// add group
	assert.NoError(t, group.InsertUserInGroup(api.mustDB(), proj.ProjectGroups[0].Group.ID, u.ID, true))
	u.Groups = append(u.Groups, proj.ProjectGroups[0].Group)

	// Add repo manager
	proj.VCSServers = make([]sdk.ProjectVCSServer, 0, 1)
	proj.VCSServers = append(proj.VCSServers)
	assert.NoError(t, repositoriesmanager.InsertForProject(db, proj, &sdk.ProjectVCSServer{
		Name:     "repoManServ",
		Username: "foo",
	}))

	// Create pipeline
	pip := &sdk.Pipeline{
		ProjectID: proj.ID,
		Name:      sdk.RandomString(10),
		Type:      "build",
	}
	assert.NoError(t, pipeline.InsertPipeline(db, api.Cache, proj, pip, u))

	s := sdk.Stage{
		PipelineID: pip.ID,
		Name:       "foo",
		Enabled:    true,
	}

	assert.NoError(t, pipeline.InsertStage(db, &s))

	j := sdk.Job{
		Enabled:         true,
		PipelineStageID: s.ID,
		Action: sdk.Action{
			Name: "script",
			Actions: []sdk.Action{
				{
					Name: "Script",
					Parameters: []sdk.Parameter{
						{
							Name:  "script",
							Value: "echo lol",
						},
					},
				},
			},
		},
	}
	assert.NoError(t, pipeline.InsertJob(db, &j, s.ID, pip))

	var errPip error
	pip, errPip = pipeline.LoadPipelineByID(db, pip.ID, true)
	assert.NoError(t, errPip)

	// Create application
	app := sdk.Application{
		ProjectID:          proj.ID,
		Name:               sdk.RandomString(10),
		RepositoryFullname: "foo/bar",
		VCSServer:          "repoManServ",
	}
	assert.NoError(t, application.Insert(db, api.Cache, proj, &app, u))
	assert.NoError(t, repositoriesmanager.InsertForApplication(db, &app, proj.Key))

	// Create workflow
	w := sdk.Workflow{
		Name:       sdk.RandomString(10),
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Root: &sdk.WorkflowNode{
			Name:         "node1",
			PipelineID:   pip.ID,
			PipelineName: pip.Name,
			Context: &sdk.WorkflowNodeContext{
				Application: &app,
			},
		},
	}
	p, err := project.Load(db, api.Cache, proj.Key, u, project.LoadOptions.WithPipelines, project.LoadOptions.WithApplications)
	assert.NoError(t, err)
	assert.NoError(t, workflow.Insert(db, api.Cache, &w, p, u))

	repositoryService := services.NewRepository(func() *gorp.DbMap {
		return db
	}, api.Cache)
	mockVCSservice := &sdk.Service{Name: "TestInsertNewCodeCoverageReport", Type: services.TypeVCS}
	assert.NoError(t, repositoryService.Delete(mockVCSservice))
	test.NoError(t, repositoryService.Insert(mockVCSservice))

	//This is a mock for the repositories service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			body := new(bytes.Buffer)
			wri := new(http.Response)
			enc := json.NewEncoder(body)
			wri.Body = ioutil.NopCloser(body)

			switch r.URL.String() {
			case "/vcs/repoManServ/repos/foo/bar":
				repo := sdk.VCSRepo{
					ID:           "1",
					Name:         "bar",
					URL:          "url",
					Fullname:     "foo/bar",
					HTTPCloneURL: "",
					Slug:         "",
					SSHCloneURL:  "",
				}
				if err := enc.Encode(repo); err != nil {
					return writeError(wri, err)
				}
			case "/vcs/repoManServ/repos/foo/bar/branches":
				bs := []sdk.VCSBranch{}
				b := sdk.VCSBranch{
					DisplayID: "master",
					Default:   true,
				}
				bs = append(bs, b)
				b2 := sdk.VCSBranch{
					DisplayID: "my-branch",
					Default:   false,
				}
				bs = append(bs, b2)
				if err := enc.Encode(bs); err != nil {
					return writeError(wri, err)
				}
			case "/vcs/repoManServ/repos/foo/bar/branches/?branch=master":
				b := sdk.VCSBranch{
					DisplayID: "master",
					Default:   true,
				}
				if err := enc.Encode(b); err != nil {
					return writeError(wri, err)
				}
			case "/vcs/repoManServ/repos/foo/bar/commits/":
				c := sdk.VCSCommit{
					URL:       "url",
					Message:   "Msg",
					Timestamp: time.Now().Unix(),
					Hash:      "123",
				}
				if err := enc.Encode(c); err != nil {
					return writeError(wri, err)
				}
			case "/vcs/repoManServ/repos/foo/bar/branches/?branch=my-branch":
				b := sdk.VCSBranch{
					DisplayID: "my-branch",
					Default:   true,
				}
				if err := enc.Encode(b); err != nil {
					return writeError(wri, err)
				}
				wri.StatusCode = http.StatusCreated
			}
			return wri, nil
		},
	)

	// Create previous run on default branch
	wrDB, _, errmr := workflow.ManualRun(context.Background(), db, api.Cache, p, &w, &sdk.WorkflowNodeRunManual{
		Payload: map[string]string{
			"git.branch": "master",
		},
	}, nil)
	assert.NoError(t, errmr)

	// Create previous run on a branch
	wrCB, _, errm := workflow.ManualRun(context.Background(), db, api.Cache, p, &w, &sdk.WorkflowNodeRunManual{
		Payload: map[string]string{
			"git.branch": "my-branch",
		},
	}, nil)
	assert.NoError(t, errm)

	// Add a coverage report on default branch node run
	coverateReportDefaultBranch := sdk.WorkflowNodeRunCoverage{
		WorkflowID:        w.ID,
		WorkflowRunID:     wrDB.ID,
		WorkflowNodeRunID: wrDB.WorkflowNodeRuns[w.RootID][0].ID,
		ApplicationID:     app.ID,
		Num:               wrDB.Number,
		Branch:            wrDB.WorkflowNodeRuns[w.RootID][0].VCSBranch,
		Repository:        wrDB.WorkflowNodeRuns[w.RootID][0].VCSRepository,
		Report: coverage.Report{
			CoveredBranches:  20,
			TotalBranches:    30,
			CoveredLines:     20,
			TotalLines:       23,
			TotalFunctions:   25,
			CoveredFunctions: 30,
		},
	}
	assert.NoError(t, workflow.InsertCoverage(db, coverateReportDefaultBranch))

	// Add a coverage report on current branch node run
	coverateReportCurrentBranch := sdk.WorkflowNodeRunCoverage{
		WorkflowID:        w.ID,
		WorkflowRunID:     wrCB.ID,
		WorkflowNodeRunID: wrCB.WorkflowNodeRuns[w.RootID][0].ID,
		ApplicationID:     app.ID,
		Num:               wrCB.Number,
		Branch:            wrCB.WorkflowNodeRuns[w.RootID][0].VCSBranch,
		Repository:        wrCB.WorkflowNodeRuns[w.RootID][0].VCSRepository,
		Report: coverage.Report{
			CoveredBranches:  0,
			TotalBranches:    30,
			CoveredLines:     0,
			TotalLines:       23,
			TotalFunctions:   25,
			CoveredFunctions: 0,
		},
	}
	assert.NoError(t, workflow.InsertCoverage(db, coverateReportCurrentBranch))

	// Run test

	// Create a workflow run
	wrToTest, _, errT := workflow.ManualRun(context.Background(), db, api.Cache, p, &w, &sdk.WorkflowNodeRunManual{
		Payload: map[string]string{
			"git.branch": "my-branch",
		},
	}, nil)
	assert.NoError(t, errT)

	wrr, err := workflow.LoadRunByID(db, wrToTest.ID, workflow.LoadRunOptions{})
	assert.NoError(t, err)

	// Call post coverage report handler
	// Prepare request
	vars := map[string]string{
		"permID": fmt.Sprintf("%d", wrr.WorkflowNodeRuns[w.RootID][0].Stages[0].RunJobs[0].ID),
	}

	request := coverage.Report{
		CoveredBranches:  1,
		TotalBranches:    30,
		CoveredLines:     1,
		TotalLines:       23,
		TotalFunctions:   25,
		CoveredFunctions: 1,
	}

	ctx := testRunWorkflowCtx{
		user:     u,
		password: pass,
		project:  proj,
		workflow: &w,
		run:      wrr,
	}
	testRegisterWorker(t, api, router, &ctx)
	ctx.worker.ActionBuildID = wrr.WorkflowNodeRuns[w.RootID][0].Stages[0].RunJobs[0].ID
	assert.NoError(t, worker.SetToBuilding(db, ctx.worker.ID, wrr.WorkflowNodeRuns[w.RootID][0].Stages[0].RunJobs[0].ID, sdk.JobTypeWorkflowNode))

	uri := router.GetRoute("POST", api.postWorkflowJobCoverageResultsHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequestFromWorker(t, ctx.worker, "POST", uri, request)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 204, rec.Code)

	covDB, errL := workflow.LoadCoverageReport(db, wrToTest.WorkflowNodeRuns[w.RootID][0].ID)
	assert.NoError(t, errL)

	assert.Equal(t, coverateReportDefaultBranch.Report.CoveredBranches, covDB.Trend.DefaultBranch.CoveredBranches)
}
