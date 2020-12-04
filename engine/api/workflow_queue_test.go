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
	"net/url"
	"os"
	"path"
	"testing"
	"time"

	"github.com/ovh/venom"
	"github.com/sguiheux/go-coverage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/purge"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

type testRunWorkflowCtx struct {
	user          *sdk.AuthentifiedUser
	password      string
	project       *sdk.Project
	workflow      *sdk.Workflow
	run           *sdk.WorkflowRun
	job           *sdk.WorkflowNodeJobRun
	worker        *sdk.Worker
	workerToken   string
	hatchery      *sdk.Service
	hatcheryToken string
	model         *sdk.Model
}

func testRunWorkflow(t *testing.T, api *API, router *Router) testRunWorkflowCtx {
	db, err := api.mustDB().Begin()
	require.NoError(t, err)

	u, pass := assets.InsertLambdaUser(t, db)
	key := "proj-" + sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
		GroupID:            proj.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
	}))
	u.Groups = append(u.Groups, proj.ProjectGroups[0].Group)

	vcsServer := sdk.ProjectVCSServerLink{
		ProjectID: proj.ID,
		Name:      "github",
	}
	vcsServer.Set("token", "foo")
	vcsServer.Set("secret", "bar")
	require.NoError(t, repositoriesmanager.InsertProjectVCSServerLink(context.TODO(), db, &vcsServer))

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip-" + sdk.RandomString(10),
	}
	require.NoError(t, pipeline.InsertPipeline(db, &pip))

	script := assets.GetBuiltinOrPluginActionByName(t, db, sdk.ScriptAction)

	s := sdk.NewStage("stage-" + sdk.RandomString(10))
	s.Enabled = true
	s.PipelineID = pip.ID
	pipeline.InsertStage(db, s)
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
			Actions: []sdk.Action{
				assets.NewAction(script.ID, sdk.Parameter{Name: "script", Value: "echo lol"}),
			},
		},
	}
	pipeline.InsertJob(db, j, s.ID, &pip)
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	// Insert Application
	app := &sdk.Application{
		Name: "app-" + sdk.RandomString(10),
	}
	require.NoError(t, application.Insert(db, proj.ID, app))

	k := &sdk.ApplicationKey{
		Name:          "my-app-key",
		Type:          "pgp",
		ApplicationID: app.ID,
	}

	pgpK, err := keys.GeneratePGPKeyPair(k.Name)
	if err != nil {
		t.Fatal(err)
	}

	k.Public = pgpK.Public
	k.Private = pgpK.Private
	k.KeyID = pgpK.KeyID

	if err := application.InsertKey(db, k); err != nil {
		t.Fatal(err)
	}

	//Insert Application
	env := &sdk.Environment{
		Name:      "env-" + sdk.RandomString(10),
		ProjectID: proj.ID,
	}
	if err := environment.InsertEnvironment(db, env); err != nil {
		t.Fatal(err)
	}

	envk := &sdk.EnvironmentKey{
		Name:          "my-env-key",
		Type:          "pgp",
		EnvironmentID: env.ID,
	}

	kpgp, err := keys.GeneratePGPKeyPair(envk.Name)
	if err != nil {
		t.Fatal(err)
	}

	envk.Public = kpgp.Public
	envk.Private = kpgp.Private
	envk.KeyID = kpgp.KeyID

	if err := environment.InsertKey(db, envk); err != nil {
		t.Fatal(err)
	}

	w := sdk.Workflow{
		Name:       "wkf-" + sdk.RandomString(10),
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "node-1",
				Ref:  "node-1",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    pip.ID,
					ApplicationID: app.ID,
					EnvironmentID: env.ID,
				},
			},
		},
	}

	proj2, errP := project.Load(context.TODO(), db, proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithGroups)
	require.NoError(t, errP)

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj2, &w))
	w1, err := workflow.Load(context.TODO(), db, api.Cache, *proj, w.Name, workflow.LoadOptions{})
	require.NoError(t, err)

	log.Debug("workflow %d groups: %+v", w1.ID, w1.Groups)
	require.NoError(t, db.Commit())

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
	require.Equal(t, 202, rec.Code)

	wr := &sdk.WorkflowRun{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), wr))
	require.Equal(t, int64(1), wr.Number)

	if t.Failed() {
		t.FailNow()
	}

	waitCraftinWorkflow(t, api, api.mustDB(), wr.ID)

	// Wait building status
	cpt := 0
	for {
		varsGet := map[string]string{
			"key":              proj.Key,
			"permWorkflowName": w1.Name,
			"number":           fmt.Sprintf("%d", wr.Number),
		}
		uriGet := router.GetRoute("GET", api.getWorkflowRunHandler, varsGet)
		test.NotEmpty(t, uriGet)
		reqGet := assets.NewAuthentifiedRequest(t, u, pass, "GET", uriGet, nil)

		//Do the request
		recGet := httptest.NewRecorder()
		router.Mux.ServeHTTP(recGet, reqGet)
		require.Equal(t, 200, recGet.Code)

		wrGet := &sdk.WorkflowRun{}
		require.NoError(t, json.Unmarshal(recGet.Body.Bytes(), wrGet))
		if wrGet.Status != sdk.StatusPending {
			wr = wrGet
			break
		}
		cpt++
		if cpt == 20 {
			t.Errorf("Workflow still in checking status: %s", wrGet.Status)
			t.FailNow()
		}
		time.Sleep(500 * time.Millisecond)
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
	require.Equal(t, 200, rec.Code)

	count := sdk.WorkflowNodeJobRunCount{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &count))
	assert.True(t, count.Count > 0)

	if t.Failed() {
		t.FailNow()
	}
}

func testGetWorkflowJobAsRegularUser(t *testing.T, api *API, router *Router, jwt string, ctx *testRunWorkflowCtx) {
	uri := router.GetRoute("GET", api.getWorkflowJobQueueHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	jobs := []sdk.WorkflowNodeJobRun{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &jobs))
	require.True(t, len(jobs) >= 1)

	if t.Failed() {
		t.FailNow()
	}

	ctx.job = &jobs[len(jobs)-1]
}

func testGetWorkflowJobAsWorker(t *testing.T, api *API, db gorpmapper.SqlExecutorWithTx, router *Router, ctx *testRunWorkflowCtx) {
	testRegisterWorker(t, api, db, router, ctx)

	uri := router.GetRoute("GET", api.getWorkflowJobQueueHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewJWTAuthentifiedRequest(t, ctx.workerToken, "GET", uri, nil)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	jobs := []sdk.WorkflowNodeJobRun{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &jobs))
	require.Len(t, jobs, 1)

	if t.Failed() {
		t.FailNow()
	}

	ctx.job = &jobs[0]
}

func testGetWorkflowJobAsHatchery(t *testing.T, api *API, db gorpmapper.SqlExecutorWithTx, router *Router, ctx *testRunWorkflowCtx) {
	uri := router.GetRoute("GET", api.getWorkflowJobQueueHandler, nil)
	test.NotEmpty(t, uri)

	//Register the worker
	testRegisterHatchery(t, api, db, router, ctx)
	req := assets.NewJWTAuthentifiedRequest(t, ctx.hatcheryToken, "GET", uri, nil)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	jobs := []sdk.WorkflowNodeJobRun{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &jobs))
	require.Len(t, jobs, 1)

	if t.Failed() {
		t.FailNow()
	}

	ctx.job = &jobs[0]
}

func testRegisterWorker(t *testing.T, api *API, db gorpmapper.SqlExecutorWithTx, router *Router, ctx *testRunWorkflowCtx) {
	g, err := group.LoadByID(context.TODO(), api.mustDB(), ctx.user.Groups[0].ID)
	if err != nil {
		t.Fatalf("Error getting group: %+v", err)
	}
	model := LoadOrCreateWorkerModel(t, api, db, g.ID, "Test1")
	var jobID int64
	if ctx.job != nil {
		jobID = ctx.job.ID
	}
	w, workerJWT := RegisterWorker(t, api, db, g.ID, model.Name, jobID, jobID == 0)
	ctx.workerToken = workerJWT
	ctx.worker = w
	ctx.model = model
}

func testRegisterHatchery(t *testing.T, api *API, db gorpmapper.SqlExecutorWithTx, router *Router, ctx *testRunWorkflowCtx) {
	h, _, _, jwt := assets.InsertHatchery(t, db, ctx.user.Groups[0])
	ctx.hatchery = h
	ctx.hatcheryToken = jwt
}

func TestGetWorkflowJobQueueHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	// delete all existing workers
	workers, err := worker.LoadAll(context.TODO(), db)
	test.NoError(t, err)
	for _, w := range workers {
		worker.Delete(db, w.ID)
	}

	// remove all jobs in queue
	filterClean := workflow.NewQueueFilter()
	nrj, _ := workflow.LoadNodeJobRunQueue(context.TODO(), db, api.Cache, filterClean)
	for _, j := range nrj {
		_ = workflow.DeleteNodeJobRuns(db, j.WorkflowNodeRunID)
	}

	_, jwt := assets.InsertAdminUser(t, db)
	t.Log("checkin as a user")

	ctx := testRunWorkflow(t, api, router)
	testGetWorkflowJobAsRegularUser(t, api, router, jwt, &ctx)
	assert.NotNil(t, ctx.job)

	t.Logf("checkin as a worker jobId:%d", ctx.job.ID)

	testGetWorkflowJobAsWorker(t, api, db, router, &ctx)
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
	require.Equal(t, 200, rec.Code)

	var n struct {
		Num int `json:"num"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &n))
	require.Equal(t, 1, n.Num)

	// Update workflow run number

	//Prepare request
	uri = router.GetRoute("POST", api.postWorkflowRunNumHandler, vars)
	test.NotEmpty(t, uri)

	n.Num = 10
	req = assets.NewAuthentifiedRequest(t, ctx.user, ctx.password, "POST", uri, n)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	uri = router.GetRoute("GET", api.getWorkflowRunNumHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, ctx.password, "GET", uri, nil)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &n))
	require.Equal(t, 10, n.Num)
}

func Test_postTakeWorkflowJobHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	ctx := testRunWorkflow(t, api, router)
	testGetWorkflowJobAsWorker(t, api, db, router, &ctx)
	require.NotNil(t, ctx.job)

	//Prepare request
	vars := map[string]string{
		"key":              ctx.project.Key,
		"permWorkflowName": ctx.workflow.Name,
		"id":               fmt.Sprintf("%d", ctx.job.ID),
	}

	// Prepare VCS Mock
	mockVCSSservice, _ := assets.InitCDNService(t, db)
	defer func() {
		_ = services.Delete(db, mockVCSSservice) // nolint
	}()

	//Register the worker
	testRegisterWorker(t, api, db, router, &ctx)

	uri := router.GetRoute("POST", api.postTakeWorkflowJobHandler, vars)
	require.NotEmpty(t, uri)

	//This will check the needWorker() auth
	req := assets.NewJWTAuthentifiedRequest(t, ctx.password, "POST", uri, nil)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 403, rec.Code)

	//This call must work
	req = assets.NewJWTAuthentifiedRequest(t, ctx.workerToken, "POST", uri, nil)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	pbji := &sdk.WorkflowNodeJobRunData{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), pbji))

	assert.Equal(t, "cdn.net:4545", pbji.GelfServiceAddr)

	run, err := workflow.LoadNodeJobRun(context.TODO(), api.mustDB(), api.Cache, ctx.job.ID)
	require.NoError(t, err)
	assert.Equal(t, "Building", run.Status)
	assert.Equal(t, ctx.model.Name, run.Model)
	assert.Equal(t, ctx.worker.Name, run.WorkerName)
	assert.NotEmpty(t, run.HatcheryName)
}

func Test_postTakeWorkflowInvalidJobHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	s, _ := assets.InitCDNService(t, db)
	defer func() {
		_ = services.Delete(db, s)
	}()

	ctx := testRunWorkflow(t, api, router)
	testGetWorkflowJobAsWorker(t, api, db, router, &ctx)
	require.NotNil(t, ctx.job)

	//Prepare request
	vars := map[string]string{
		"key":              ctx.project.Key,
		"permWorkflowName": ctx.workflow.Name,
		"id":               fmt.Sprintf("%d", ctx.job.ID+1), // invalid job
	}

	//Register the worker
	testRegisterWorker(t, api, db, router, &ctx)

	uri := router.GetRoute("POST", api.postTakeWorkflowJobHandler, vars)
	require.NotEmpty(t, uri)

	//this call must failed, we try to take a jobID not reserved at worker's registration
	req := assets.NewJWTAuthentifiedRequest(t, ctx.workerToken, "POST", uri, nil)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 403, rec.Code)

	//This must be ok, take the jobID reserved
	vars2 := map[string]string{
		"key":              ctx.project.Key,
		"permWorkflowName": ctx.workflow.Name,
		"id":               fmt.Sprintf("%d", ctx.job.ID),
	}
	uri2 := router.GetRoute("POST", api.postTakeWorkflowJobHandler, vars2)
	require.NotEmpty(t, uri2)
	req2 := assets.NewJWTAuthentifiedRequest(t, ctx.workerToken, "POST", uri2, nil)
	rec2 := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec2, req2)
	require.Equal(t, 200, rec2.Code)
}

func Test_postBookWorkflowJobHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	ctx := testRunWorkflow(t, api, router)
	testGetWorkflowJobAsHatchery(t, api, db, router, &ctx)
	assert.NotNil(t, ctx.job)

	//Register the hatchery
	testRegisterHatchery(t, api, db, router, &ctx)

	//TakeBook
	uri := router.GetRoute("POST", api.postBookWorkflowJobHandler, map[string]string{
		"permJobID": fmt.Sprintf("%d", ctx.job.ID),
	})
	test.NotEmpty(t, uri)

	req := assets.NewJWTAuthentifiedRequest(t, ctx.hatcheryToken, "POST", uri, nil)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)
}

func Test_postWorkflowJobResultHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	s, _ := assets.InitCDNService(t, db)
	defer func() {
		_ = services.Delete(db, s)
	}()

	ctx := testRunWorkflow(t, api, router)
	testGetWorkflowJobAsWorker(t, api, db, router, &ctx)
	assert.NotNil(t, ctx.job)

	//Register the worker
	testRegisterWorker(t, api, db, router, &ctx)

	//Take
	uri := router.GetRoute("POST", api.postTakeWorkflowJobHandler, map[string]string{
		"id": fmt.Sprintf("%d", ctx.job.ID),
	})
	test.NotEmpty(t, uri)

	req := assets.NewJWTAuthentifiedRequest(t, ctx.workerToken, "POST", uri, nil)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	//Send result
	res := sdk.Result{
		Duration:   "10",
		Status:     sdk.StatusSuccess,
		RemoteTime: time.Now(),
		BuildID:    ctx.job.ID,
		NewVariables: []sdk.Variable{
			{
				Name:  "cds.build.newVar",
				Value: "newVal",
			},
		},
	}

	uri = router.GetRoute("POST", api.postWorkflowJobResultHandler, map[string]string{
		"permJobID": fmt.Sprintf("%d", ctx.job.ID),
	})
	test.NotEmpty(t, uri)

	req = assets.NewJWTAuthentifiedRequest(t, ctx.workerToken, "POST", uri, res)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 204, rec.Code)

	uri = router.GetRoute("GET", api.getWorkflowRunHandler, map[string]string{
		"key":              ctx.project.Key,
		"permWorkflowName": ctx.workflow.Name,
		"number":           fmt.Sprintf("%d", ctx.run.Number),
	})
	req = assets.NewJWTAuthentifiedRequest(t, ctx.password, "GET", uri+"?withDetails=true", res)

	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	btes := rec.Body.Bytes()
	require.NoError(t, json.Unmarshal(btes, ctx.run))
	assert.Contains(t, ctx.run.RootRun().BuildParameters, sdk.Parameter{Name: "cds.build.newVar", Type: sdk.StringParameter, Value: "newVal"})

	vars := map[string]string{
		"key":              ctx.project.Key,
		"permWorkflowName": ctx.workflow.Name,
		"number":           fmt.Sprintf("%d", ctx.run.Number),
		"nodeRunID":        fmt.Sprintf("%d", ctx.run.RootRun().ID),
	}
	uri = router.GetRoute("GET", api.getWorkflowNodeRunHandler, vars)
	req = assets.NewJWTAuthentifiedRequest(t, ctx.password, "GET", uri, res)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)
	btes = rec.Body.Bytes()
	var rootRun sdk.WorkflowNodeRun
	require.NoError(t, json.Unmarshal(btes, &rootRun))

	assert.Contains(t, rootRun.Stages[0].RunJobs[0].Parameters, sdk.Parameter{Name: "cds.build.newVar", Type: sdk.StringParameter, Value: "newVal"})
	assert.Contains(t, rootRun.BuildParameters, sdk.Parameter{Name: "cds.build.newVar", Type: sdk.StringParameter, Value: "newVal"})
}

func Test_postWorkflowJobTestsResultsHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	s, _ := assets.InitCDNService(t, db)
	defer func() {
		_ = services.Delete(db, s)
	}()

	ctx := testRunWorkflow(t, api, router)
	testGetWorkflowJobAsWorker(t, api, db, router, &ctx)
	assert.NotNil(t, ctx.job)

	// Register the worker
	testRegisterWorker(t, api, db, router, &ctx)
	// Register the hatchery
	testRegisterHatchery(t, api, db, router, &ctx)

	// Send spawninfo
	info := []sdk.SpawnInfo{}
	uri := fmt.Sprintf("%s/queue/workflows/%d/spawn/infos", router.Prefix, ctx.job.ID)
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, ctx.hatcheryToken, "POST", uri, info)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 204, rec.Code)

	//spawn
	uri = router.GetRoute("POST", api.postTakeWorkflowJobHandler, map[string]string{
		"key":              ctx.project.Key,
		"permWorkflowName": ctx.workflow.Name,
		"id":               fmt.Sprintf("%d", ctx.job.ID),
	})
	test.NotEmpty(t, uri)

	req = assets.NewJWTAuthentifiedRequest(t, ctx.workerToken, "POST", uri, nil)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

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
								Message: "Error occurred",
							},
						},
					},
				},
			},
		},
	}
	uri = router.GetRoute("POST", api.postWorkflowJobTestsResultsHandler, map[string]string{
		"permJobID": fmt.Sprintf("%d", ctx.job.ID),
	})
	test.NotEmpty(t, uri)

	req = assets.NewJWTAuthentifiedRequest(t, ctx.workerToken, "POST", uri, tests)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 204, rec.Code)

	step := sdk.StepStatus{
		Status:    sdk.StatusSuccess,
		StepOrder: 0,
	}

	uri = router.GetRoute("POST", api.postWorkflowJobStepStatusHandler, map[string]string{
		"permJobID": fmt.Sprintf("%d", ctx.job.ID),
	})
	test.NotEmpty(t, uri)

	req = assets.NewJWTAuthentifiedRequest(t, ctx.workerToken, "POST", uri, step)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 204, rec.Code)

	wNodeJobRun, errJ := workflow.LoadNodeJobRun(context.TODO(), api.mustDB(), api.Cache, ctx.job.ID)
	require.NoError(t, errJ)
	nodeRun, errN := workflow.LoadNodeRunByID(api.mustDB(), wNodeJobRun.WorkflowNodeRunID, workflow.LoadRunOptions{WithArtifacts: true, WithTests: true})
	require.NoError(t, errN)

	assert.NotNil(t, nodeRun.Tests)
	require.Equal(t, 2, nodeRun.Tests.Total)
}

func Test_postWorkflowJobArtifactHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	s, _ := assets.InitCDNService(t, db)
	defer func() {
		_ = services.Delete(db, s)
	}()

	ctx := testRunWorkflow(t, api, router)
	testGetWorkflowJobAsWorker(t, api, db, router, &ctx)

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

	storage, errO := objectstore.Init(context.Background(), cfg)
	require.NoError(t, errO)
	api.SharedStorage = storage

	//Prepare request
	vars := map[string]string{
		"key":              ctx.project.Key,
		"permWorkflowName": ctx.workflow.Name,
		"id":               fmt.Sprintf("%d", ctx.job.ID),
	}

	//Register the worker
	testRegisterWorker(t, api, db, router, &ctx)

	//Take
	uri := router.GetRoute("POST", api.postTakeWorkflowJobHandler, vars)
	test.NotEmpty(t, uri)

	req := assets.NewJWTAuthentifiedRequest(t, ctx.workerToken, "POST", uri, nil)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	vars = map[string]string{
		"ref":             base64.RawURLEncoding.EncodeToString([]byte("latest")),
		"integrationName": sdk.DefaultStorageIntegrationName,
		"permProjectKey":  ctx.project.Key,
	}

	uri = router.GetRoute("POST", api.postWorkflowJobArtifactHandler, vars)
	test.NotEmpty(t, uri)

	myartifact, errF := os.Create(path.Join(os.TempDir(), "myartifact"))
	defer os.RemoveAll(path.Join(os.TempDir(), "myartifact"))
	require.NoError(t, errF)
	_, errW := myartifact.Write([]byte("Hi, I am foo"))
	require.NoError(t, errW)

	errClose := myartifact.Close()
	require.NoError(t, errClose)

	params := map[string]string{}
	params["size"] = "12"
	params["perm"] = "7"
	params["md5sum"] = "123"
	params["sha512sum"] = "1234"
	params["nodeJobRunID"] = fmt.Sprintf("%d", ctx.job.ID)
	req = assets.NewJWTAuthentifiedMultipartRequest(t, ctx.workerToken, "POST", uri, path.Join(os.TempDir(), "myartifact"), "myartifact", params)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 204, rec.Code)

	time.Sleep(1 * time.Second)

	wNodeJobRun, errJ := workflow.LoadNodeJobRun(context.TODO(), api.mustDB(), api.Cache, ctx.job.ID)
	require.NoError(t, errJ)

	updatedNodeRun, errN2 := workflow.LoadNodeRunByID(api.mustDB(), wNodeJobRun.WorkflowNodeRunID, workflow.LoadRunOptions{WithArtifacts: true})
	require.NoError(t, errN2)

	assert.NotNil(t, updatedNodeRun.Artifacts)
	require.Equal(t, 1, len(updatedNodeRun.Artifacts))

	//Prepare request
	vars = map[string]string{
		"key":              ctx.project.Key,
		"permWorkflowName": ctx.workflow.Name,
		"number":           fmt.Sprintf("%d", updatedNodeRun.Number),
	}
	uri = router.GetRoute("GET", api.getWorkflowRunArtifactsHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, ctx.password, "GET", uri, nil)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	var arts []sdk.WorkflowNodeRunArtifact
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &arts))
	require.Equal(t, 1, len(arts))
	require.Equal(t, "myartifact", arts[0].Name)

	// Download artifact
	//Prepare request
	vars = map[string]string{
		"key":              ctx.project.Key,
		"permWorkflowName": ctx.workflow.Name,
		"artifactId":       fmt.Sprintf("%d", arts[0].ID),
	}
	uri = router.GetRoute("GET", api.getDownloadArtifactHandler, vars)
	test.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, ctx.password, "GET", uri, nil)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)

	resp := rec.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	require.Equal(t, 200, rec.Code)
	require.Equal(t, "Hi, I am foo", string(body))

	// check if file is stored locally
	containerPath := path.Join(os.TempDir(), "store", fmt.Sprintf("%d-%d-%v", ctx.run.ID, wNodeJobRun.WorkflowNodeRunID, arts[0].Ref))

	artifactPath := path.Join(containerPath, "myartifact")
	exists := fileExists(artifactPath)
	assert.Equal(t, true, exists)

	// then purge run to delete artifact
	require.NoError(t, purge.DeleteArtifacts(router.Background, db, api.SharedStorage, ctx.run.ID))

	// check if file is deleted
	exists = fileExists(artifactPath)
	assert.Equal(t, false, exists)

	if _, err := os.Stat(containerPath); !os.IsNotExist(err) {
		t.FailNow()
	}

}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func Test_postWorkflowJobStaticFilesHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	s, _ := assets.InitCDNService(t, db)
	defer func() {
		_ = services.Delete(db, s)
	}()

	ctx := testRunWorkflow(t, api, router)
	testGetWorkflowJobAsWorker(t, api, db, router, &ctx)
	require.NotNil(t, ctx.job)

	// Init store
	cfg := objectstore.Config{
		Kind: objectstore.Filesystem,
		Options: objectstore.ConfigOptions{
			Filesystem: objectstore.ConfigOptionsFilesystem{
				Basedir: path.Join(os.TempDir(), "store"),
			},
		},
	}

	storage, errO := objectstore.Init(context.Background(), cfg)
	require.NoError(t, errO)
	api.SharedStorage = storage

	//Prepare request
	vars := map[string]string{
		"key":              ctx.project.Key,
		"permWorkflowName": ctx.workflow.Name,
		"id":               fmt.Sprintf("%d", ctx.job.ID),
	}

	//Register the worker
	testRegisterWorker(t, api, db, router, &ctx)

	//Take
	uri := router.GetRoute("POST", api.postTakeWorkflowJobHandler, vars)
	test.NotEmpty(t, uri)

	req := assets.NewJWTAuthentifiedRequest(t, ctx.workerToken, "POST", uri, nil)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	vars = map[string]string{
		"name":            url.PathEscape("mywebsite"),
		"integrationName": sdk.DefaultStorageIntegrationName,
		"permProjectKey":  ctx.project.Key,
	}

	uri = router.GetRoute("POST", api.postWorkflowJobStaticFilesHandler, vars)
	test.NotEmpty(t, uri)

	mystaticfile, errF := os.Create(path.Join(os.TempDir(), "mystaticfile"))
	defer os.RemoveAll(path.Join(os.TempDir(), "mystaticfile"))
	require.NoError(t, errF)
	_, errW := mystaticfile.Write([]byte("<html>Hi, I am foo</html>"))
	require.NoError(t, errW)

	errClose := mystaticfile.Close()
	require.NoError(t, errClose)

	params := map[string]string{
		"entrypoint":   "index.html",
		"nodeJobRunID": fmt.Sprintf("%d", ctx.job.ID),
	}
	req = assets.NewJWTAuthentifiedMultipartRequest(t, ctx.workerToken, "POST", uri, path.Join(os.TempDir(), "mystaticfile"), "mystaticfile", params)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNotImplemented, rec.Code)
}

func TestWorkerPrivateKey(t *testing.T) {
	api, db, router := newTestAPI(t)

	// Create user
	u, pass := assets.InsertAdminUser(t, db)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	// Create project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	// add group
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
		GroupID:            proj.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
	}))
	u.Groups = append(u.Groups, proj.ProjectGroups[0].Group)

	// Create pipeline
	pip := &sdk.Pipeline{
		ProjectID: proj.ID,
		Name:      sdk.RandomString(10),
	}
	assert.NoError(t, pipeline.InsertPipeline(db, pip))

	s := sdk.Stage{
		PipelineID: pip.ID,
		Name:       "foo",
		Enabled:    true,
	}

	assert.NoError(t, pipeline.InsertStage(db, &s))

	// get script action
	script := assets.GetBuiltinOrPluginActionByName(t, db, sdk.ScriptAction)

	j := sdk.Job{
		Enabled:         true,
		PipelineStageID: s.ID,
		Action: sdk.Action{
			Name: "script",
			Actions: []sdk.Action{
				assets.NewAction(script.ID, sdk.Parameter{Name: "script", Value: "echo lol"}),
			},
		},
	}
	assert.NoError(t, pipeline.InsertJob(db, &j, s.ID, pip))

	var errPip error
	pip, errPip = pipeline.LoadPipelineByID(context.TODO(), db, pip.ID, true)
	assert.NoError(t, errPip)

	// Create application
	app := sdk.Application{
		ProjectID: proj.ID,
		Name:      sdk.RandomString(10),
	}
	assert.NoError(t, application.Insert(db, proj.ID, &app))

	// Create workflow
	w := sdk.Workflow{
		Name:       sdk.RandomString(10),
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "node1",
				Ref:  "node1",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    pip.ID,
					ApplicationID: app.ID,
				},
			},
		},
	}

	p, err := project.Load(context.TODO(), db, proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithApplications)
	assert.NoError(t, err)
	assert.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *p, &w))

	workflowDeepPipeline, err := workflow.LoadByID(context.TODO(), db, api.Cache, *p, w.ID, workflow.LoadOptions{DeepPipeline: true})
	assert.NoError(t, err)

	wrDB, errwr := workflow.CreateRun(api.mustDB(), workflowDeepPipeline, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	assert.NoError(t, errwr)
	wrDB.Workflow = *workflowDeepPipeline

	_, errmr := workflow.StartWorkflowRun(context.Background(), db, api.Cache, *p, wrDB,
		&sdk.WorkflowRunPostHandlerOption{
			Manual: &sdk.WorkflowNodeRunManual{Username: u.Username},
		},
		*consumer, nil)
	assert.NoError(t, errmr)

	ctx := testRunWorkflowCtx{
		user:     u,
		password: pass,
		project:  proj,
		workflow: &w,
		run:      wrDB,
	}
	testRegisterWorker(t, api, db, router, &ctx)
	ctx.worker.JobRunID = &wrDB.WorkflowNodeRuns[w.WorkflowData.Node.ID][0].Stages[0].RunJobs[0].ID
	assert.NoError(t, worker.SetToBuilding(context.TODO(), db, ctx.worker.ID, *ctx.worker.JobRunID, []byte("mysecret")))

	wkFromDB, err := worker.LoadWorkerByNameWithDecryptKey(context.TODO(), db, ctx.worker.Name)
	require.NoError(t, err)
	require.Equal(t, "mysecret", string(wkFromDB.PrivateKey))
}

func TestPostVulnerabilityReportHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	// Create user
	u, pass := assets.InsertAdminUser(t, db)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	// Create project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	// add group
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
		GroupID:            proj.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
	}))
	u.Groups = append(u.Groups, proj.ProjectGroups[0].Group)

	// Create pipeline
	pip := &sdk.Pipeline{
		ProjectID: proj.ID,
		Name:      sdk.RandomString(10),
	}
	assert.NoError(t, pipeline.InsertPipeline(db, pip))

	s := sdk.Stage{
		PipelineID: pip.ID,
		Name:       "foo",
		Enabled:    true,
	}

	assert.NoError(t, pipeline.InsertStage(db, &s))

	// get script action
	script := assets.GetBuiltinOrPluginActionByName(t, db, sdk.ScriptAction)

	j := sdk.Job{
		Enabled:         true,
		PipelineStageID: s.ID,
		Action: sdk.Action{
			Name: "script",
			Actions: []sdk.Action{
				assets.NewAction(script.ID, sdk.Parameter{Name: "script", Value: "echo lol"}),
			},
		},
	}
	assert.NoError(t, pipeline.InsertJob(db, &j, s.ID, pip))

	var errPip error
	pip, errPip = pipeline.LoadPipelineByID(context.TODO(), db, pip.ID, true)
	assert.NoError(t, errPip)

	// Create application
	app := sdk.Application{
		ProjectID: proj.ID,
		Name:      sdk.RandomString(10),
	}
	require.NoError(t, application.Insert(db, proj.ID, &app))

	// Create workflow
	w := sdk.Workflow{
		Name:       sdk.RandomString(10),
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "node1",
				Ref:  "node1",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    pip.ID,
					ApplicationID: app.ID,
				},
			},
		},
	}

	p, err := project.Load(context.TODO(), db, proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithApplications)
	assert.NoError(t, err)
	assert.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *p, &w))

	workflowDeepPipeline, err := workflow.LoadByID(context.TODO(), db, api.Cache, *p, w.ID, workflow.LoadOptions{DeepPipeline: true})
	assert.NoError(t, err)

	wrDB, errwr := workflow.CreateRun(api.mustDB(), workflowDeepPipeline, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	assert.NoError(t, errwr)
	wrDB.Workflow = *workflowDeepPipeline

	_, errmr := workflow.StartWorkflowRun(context.Background(), db, api.Cache, *p, wrDB,
		&sdk.WorkflowRunPostHandlerOption{
			Manual: &sdk.WorkflowNodeRunManual{Username: u.Username},
		},
		*consumer, nil)
	assert.NoError(t, errmr)

	log.Debug("%+v", wrDB.WorkflowNodeRuns)

	// Call post coverage report handler
	// Prepare request
	vars := map[string]string{
		"permJobID": fmt.Sprintf("%d", wrDB.WorkflowNodeRuns[w.WorkflowData.Node.ID][0].Stages[0].RunJobs[0].ID),
	}

	ctx := testRunWorkflowCtx{
		user:     u,
		password: pass,
		project:  proj,
		workflow: &w,
		run:      wrDB,
	}
	testRegisterWorker(t, api, db, router, &ctx)
	ctx.worker.JobRunID = &wrDB.WorkflowNodeRuns[w.WorkflowData.Node.ID][0].Stages[0].RunJobs[0].ID
	assert.NoError(t, worker.SetToBuilding(context.TODO(), db, ctx.worker.ID, *ctx.worker.JobRunID, nil))

	request := sdk.VulnerabilityWorkerReport{
		Vulnerabilities: []sdk.Vulnerability{
			{
				Version:     "1.0.0",
				Title:       "lodash",
				Severity:    "high",
				Origin:      "parsejson>lodash",
				Link:        "",
				FixIn:       "",
				Description: "",
				CVE:         "",
				Component:   "",
				Ignored:     false,
			},
		},
	}

	uri := router.GetRoute("POST", api.postVulnerabilityReportHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, ctx.workerToken, "POST", uri, request)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 204, rec.Code)
}

func TestInsertNewCodeCoverageReport(t *testing.T) {
	api, db, router := newTestAPI(t)

	// Create user
	u, pass := assets.InsertAdminUser(t, db)

	// Create project
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	// add group
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
		GroupID:            proj.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
	}))
	u.Groups = append(u.Groups, proj.ProjectGroups[0].Group)

	// Add repo manager
	proj.VCSServers = make([]sdk.ProjectVCSServerLink, 0, 1)
	proj.VCSServers = append(proj.VCSServers)

	vcsServer := sdk.ProjectVCSServerLink{
		ProjectID: proj.ID,
		Name:      "repoManServ",
	}
	vcsServer.Set("token", "foo")
	vcsServer.Set("secret", "bar")
	assert.NoError(t, repositoriesmanager.InsertProjectVCSServerLink(context.TODO(), db, &vcsServer))

	// Create pipeline
	pip := &sdk.Pipeline{
		ProjectID: proj.ID,
		Name:      sdk.RandomString(10),
	}
	assert.NoError(t, pipeline.InsertPipeline(db, pip))

	s := sdk.Stage{
		PipelineID: pip.ID,
		Name:       "foo",
		Enabled:    true,
	}

	assert.NoError(t, pipeline.InsertStage(db, &s))

	// get script action
	script := assets.GetBuiltinOrPluginActionByName(t, db, sdk.ScriptAction)

	j := sdk.Job{
		Enabled:         true,
		PipelineStageID: s.ID,
		Action: sdk.Action{
			Name: "script",
			Actions: []sdk.Action{
				assets.NewAction(script.ID, sdk.Parameter{Name: "script", Value: "echo lol"}),
			},
		},
	}
	assert.NoError(t, pipeline.InsertJob(db, &j, s.ID, pip))

	var errPip error
	pip, errPip = pipeline.LoadPipelineByID(context.TODO(), db, pip.ID, true)
	assert.NoError(t, errPip)

	// Create application
	app := sdk.Application{
		ProjectID:          proj.ID,
		Name:               sdk.RandomString(10),
		RepositoryFullname: "foo/bar",
		VCSServer:          "repoManServ",
	}
	require.NoError(t, application.Insert(db, proj.ID, &app))
	require.NoError(t, repositoriesmanager.InsertForApplication(db, &app))

	// Create workflow
	w := sdk.Workflow{
		Name:       sdk.RandomString(10),
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "node1",
				Ref:  "node1",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID:    pip.ID,
					ApplicationID: app.ID,
				},
			},
		},
	}

	p, err := project.Load(context.TODO(), db, proj.Key, project.LoadOptions.WithPipelines, project.LoadOptions.WithApplications)
	require.NoError(t, err)
	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *p, &w))

	allSrv, err := services.LoadAll(context.TODO(), db)
	for _, s := range allSrv {
		if err := services.Delete(db, &s); err != nil {
			t.Fatalf("unable to delete service: %v", err)
		}
	}

	a, _ := assets.InsertService(t, db, "TestInsertNewCodeCoverageReport", sdk.TypeVCS)

	defer func() {
		_ = services.Delete(db, a)
	}()

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

	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	// Create previous run on default branch
	wrDB, errwr := workflow.CreateRun(api.mustDB(), &w, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	assert.NoError(t, errwr)

	workflowWithDeepPipeline, err := workflow.LoadByID(context.TODO(), db, api.Cache, *proj, w.ID, workflow.LoadOptions{DeepPipeline: true})
	assert.NoError(t, err)

	wrDB.Workflow = *workflowWithDeepPipeline
	_, errmr := workflow.StartWorkflowRun(context.Background(), db, api.Cache, *p, wrDB, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.Username,
			Payload: map[string]string{
				"git.branch": "master",
			},
		},
	}, *consumer, nil)

	assert.NoError(t, errmr)

	// Create previous run on a branch
	wrCB, errwr2 := workflow.CreateRun(api.mustDB(), &w, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	assert.NoError(t, errwr2)
	wrCB.Workflow = w
	_, errmr = workflow.StartWorkflowRun(context.Background(), db, api.Cache, *p, wrCB, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.Username,
			Payload: map[string]string{
				"git.branch": "my-branch",
			},
		},
	}, *consumer, nil)
	assert.NoError(t, errmr)

	// Add a coverage report on default branch node run
	coverateReportDefaultBranch := sdk.WorkflowNodeRunCoverage{
		WorkflowID:        w.ID,
		WorkflowRunID:     wrDB.ID,
		WorkflowNodeRunID: wrDB.WorkflowNodeRuns[w.WorkflowData.Node.ID][0].ID,
		ApplicationID:     app.ID,
		Num:               wrDB.Number,
		Branch:            wrDB.WorkflowNodeRuns[w.WorkflowData.Node.ID][0].VCSBranch,
		Repository:        wrDB.WorkflowNodeRuns[w.WorkflowData.Node.ID][0].VCSRepository,
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
		WorkflowNodeRunID: wrCB.WorkflowNodeRuns[w.WorkflowData.Node.ID][0].ID,
		ApplicationID:     app.ID,
		Num:               wrCB.Number,
		Branch:            wrCB.WorkflowNodeRuns[w.WorkflowData.Node.ID][0].VCSBranch,
		Repository:        wrCB.WorkflowNodeRuns[w.WorkflowData.Node.ID][0].VCSRepository,
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
	wrToTest, errwr3 := workflow.CreateRun(api.mustDB(), &w, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	assert.NoError(t, errwr3)
	wrToTest.Workflow = *workflowWithDeepPipeline

	_, errT := workflow.StartWorkflowRun(context.Background(), db, api.Cache, *p, wrToTest, &sdk.WorkflowRunPostHandlerOption{
		Manual: &sdk.WorkflowNodeRunManual{
			Username: u.Username,
			Payload: map[string]string{
				"git.branch": "my-branch",
			},
		},
	}, *consumer, nil)
	assert.NoError(t, errT)

	wrr, err := workflow.LoadRunByID(db, wrToTest.ID, workflow.LoadRunOptions{})
	assert.NoError(t, err)

	log.Warning(context.Background(), "%s", wrr.Status)
	// Call post coverage report handler
	// Prepare request
	vars := map[string]string{
		"permJobID": fmt.Sprintf("%d", wrr.WorkflowNodeRuns[w.WorkflowData.Node.ID][0].Stages[0].RunJobs[0].ID),
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
	testRegisterWorker(t, api, db, router, &ctx)
	ctx.worker.JobRunID = &wrr.WorkflowNodeRuns[w.WorkflowData.Node.ID][0].Stages[0].RunJobs[0].ID
	assert.NoError(t, worker.SetToBuilding(context.TODO(), db, ctx.worker.ID, *ctx.worker.JobRunID, nil))

	uri := router.GetRoute("POST", api.postWorkflowJobCoverageResultsHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, ctx.workerToken, "POST", uri, request)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 204, rec.Code)

	covDB, errL := workflow.LoadCoverageReport(db, wrToTest.WorkflowNodeRuns[w.WorkflowData.Node.ID][0].ID)
	assert.NoError(t, errL)

	require.Equal(t, coverateReportDefaultBranch.Report.CoveredBranches, covDB.Trend.DefaultBranch.CoveredBranches)
}

func Test_postWorkflowJobSetVersionHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	s, _ := assets.InitCDNService(t, db)
	defer func() {
		_ = services.Delete(db, s)
	}()

	ctx := testRunWorkflow(t, api, router)
	testGetWorkflowJobAsWorker(t, api, db, router, &ctx)
	require.NotNil(t, ctx.job)

	// Register the worker
	testRegisterWorker(t, api, db, router, &ctx)

	// Take the job
	uri := router.GetRoute("POST", api.postTakeWorkflowJobHandler, map[string]string{
		"id": fmt.Sprintf("%d", ctx.job.ID),
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, ctx.workerToken, "POST", uri, nil)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	// Check that version is not set
	run, err := workflow.LoadRun(context.TODO(), db, ctx.project.Key, ctx.workflow.Name, ctx.run.Number, workflow.LoadRunOptions{})
	require.NoError(t, err)
	require.Empty(t, "", run.Version)
	nodeRun, err := workflow.LoadNodeRunByID(db, ctx.job.WorkflowNodeRunID, workflow.LoadRunOptions{})
	require.NoError(t, err)
	require.Equal(t, "1", sdk.ParameterValue(nodeRun.BuildParameters, "cds.version"))

	// Set version from worker
	uri = router.GetRoute("POST", api.postWorkflowJobSetVersionHandler, map[string]string{
		"permJobID": fmt.Sprintf("%d", ctx.job.ID),
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, ctx.workerToken, "POST", uri, sdk.WorkflowRunVersion{
		Value: "1.2.3",
	})
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 204, rec.Code)

	run, err = workflow.LoadRun(context.TODO(), db, ctx.project.Key, ctx.workflow.Name, ctx.run.Number, workflow.LoadRunOptions{})
	require.NoError(t, err)
	require.NotNil(t, run.Version)
	require.Equal(t, "1.2.3", *run.Version)
	nodeRun, err = workflow.LoadNodeRunByID(db, ctx.job.WorkflowNodeRunID, workflow.LoadRunOptions{})
	require.NoError(t, err)
	require.Equal(t, "1.2.3", sdk.ParameterValue(nodeRun.BuildParameters, "cds.version"))

	uri = router.GetRoute("POST", api.postWorkflowJobSetVersionHandler, map[string]string{
		"permJobID": fmt.Sprintf("%d", ctx.job.ID),
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, ctx.workerToken, "POST", uri, sdk.WorkflowRunVersion{
		Value: "3.2.1",
	})
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 403, rec.Code)
}
