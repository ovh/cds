package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/sdk/cdsclient"

	"github.com/rockbears/log"
	"github.com/sguiheux/go-coverage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

type testRunWorkflowCtx struct {
	project       *sdk.Project
	workflow      *sdk.Workflow
	run           *sdk.WorkflowRun
	job           *sdk.WorkflowNodeJobRun
	model         *sdk.Model
	user          *sdk.AuthentifiedUser
	userToken     string
	worker        *sdk.Worker
	workerToken   string
	hatchery      *sdk.Service
	hatcheryToken string
}

type testRunWorkflowOptions func(*testing.T, gorpmapper.SqlExecutorWithTx, *sdk.Pipeline, *sdk.Application)

func testRunWorkflow(t *testing.T, api *API, router *Router, optsF ...testRunWorkflowOptions) testRunWorkflowCtx {
	db, err := api.mustDB().Begin()
	require.NoError(t, err)

	key := "proj-" + sdk.RandomString(10)
	u, jwtLambda := assets.InsertLambdaUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)
	require.NoError(t, group.InsertLinkGroupUser(context.TODO(), db, &group.LinkGroupUser{
		GroupID:            proj.ProjectGroups[0].Group.ID,
		AuthentifiedUserID: u.ID,
		Admin:              true,
	}))
	u.Groups = append(u.Groups, proj.ProjectGroups[0].Group)

	proj.Keys = []sdk.ProjectKey{
		{
			Type: sdk.KeyTypeSSH,
			Name: sdk.GenerateProjectDefaultKeyName(proj.Key, sdk.KeyTypeSSH),
		},
		{
			Type: sdk.KeyTypePGP,
			Name: sdk.GenerateProjectDefaultKeyName(proj.Key, sdk.KeyTypePGP),
		},
	}
	for i := range proj.Keys {
		k := &proj.Keys[i]
		k.ProjectID = proj.ID
		newKey, err := keys.GenerateKey(k.Name, k.Type)
		require.NoError(t, err)
		k.Private = newKey.Private
		k.Public = newKey.Public
		k.KeyID = newKey.KeyID
		require.NoError(t, project.InsertKey(db, k))
	}

	vcsServer := sdk.ProjectVCSServerLink{
		ProjectID: proj.ID,
		Name:      "github",
	}
	vcsServer.Set("token", "foo")
	vcsServer.Set("secret", "bar")
	require.NoError(t, repositoriesmanager.InsertProjectVCSServerLink(context.TODO(), db, &vcsServer))

	require.NoError(t, db.Commit())

	res := testRunWorkflowForProject(t, api, router, proj, jwtLambda, optsF...)
	res.user = u
	res.userToken = jwtLambda

	return res
}

func testRunWorkflowForProject(t *testing.T, api *API, router *Router, proj *sdk.Project, jwtLambda string, optsF ...testRunWorkflowOptions) testRunWorkflowCtx {
	db, err := api.mustDB().Begin()
	require.NoError(t, err)

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
	require.NoError(t, application.Insert(db, *proj, app))

	for _, opt := range optsF {
		opt(t, db, &pip, app)
	}

	k := &sdk.ApplicationKey{
		Name:          "my-app-key",
		Type:          "pgp",
		ApplicationID: app.ID,
	}

	pgpK, err := keys.GeneratePGPKeyPair(k.Name)
	require.NoError(t, err)

	k.Public = pgpK.Public
	k.Private = pgpK.Private
	k.KeyID = pgpK.KeyID

	require.NoError(t, application.InsertKey(db, k))

	//Insert Application
	env := &sdk.Environment{
		Name:      "env-" + sdk.RandomString(10),
		ProjectID: proj.ID,
	}
	require.NoError(t, environment.InsertEnvironment(db, env))

	envk := &sdk.EnvironmentKey{
		Name:          "my-env-key",
		Type:          "pgp",
		EnvironmentID: env.ID,
	}

	kpgp, err := keys.GeneratePGPKeyPair(envk.Name)
	require.NoError(t, err)

	envk.Public = kpgp.Public
	envk.Private = kpgp.Private
	envk.KeyID = kpgp.KeyID

	require.NoError(t, environment.InsertKey(db, envk))

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

	log.Debug(context.TODO(), "workflow %d groups: %+v", w1.ID, w1.Groups)
	require.NoError(t, db.Commit())

	//Prepare request
	uri := router.GetRoute("POST", api.postWorkflowRunHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
	})
	require.NotEmpty(t, uri)

	opts := &sdk.WorkflowRunPostHandlerOption{}
	req := assets.NewJWTAuthentifiedRequest(t, jwtLambda, "POST", uri, opts)

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

	require.NoError(t, api.workflowRunCraft(context.TODO(), wr.ID))

	// Wait building status
	cpt := 0
	for {
		varsGet := map[string]string{
			"key":              proj.Key,
			"permWorkflowName": w1.Name,
			"number":           fmt.Sprintf("%d", wr.Number),
		}
		uriGet := router.GetRoute("GET", api.getWorkflowRunHandler, varsGet)
		require.NotEmpty(t, uriGet)
		reqGet := assets.NewJWTAuthentifiedRequest(t, jwtLambda, "GET", uriGet, nil)

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

	require.Len(t, wr.WorkflowNodeRuns, 1)
	var nodeRunID int64
	for _, nodeRun := range wr.WorkflowNodeRuns {
		nodeRunID = nodeRun[0].ID
		break
	}

	jobs, err := workflow.LoadNodeJobRunQueue(context.TODO(), api.mustDB(), api.Cache, workflow.NewQueueFilter())
	require.NoError(t, err)

	var job *sdk.WorkflowNodeJobRun
	for i := range jobs {
		if jobs[i].WorkflowNodeRunID == nodeRunID {
			job = &jobs[i]
			break
		}
	}
	require.NotNil(t, job)

	return testRunWorkflowCtx{
		project:  proj,
		workflow: w1,
		run:      wr,
		job:      job,
	}
}

func testCountGetWorkflowJobAsRegularUser(t *testing.T, api *API, router *Router, ctx testRunWorkflowCtx) {
	uri := router.GetRoute("GET", api.countWorkflowJobQueueHandler, nil)
	require.NotEmpty(t, uri)

	req := assets.NewJWTAuthentifiedRequest(t, ctx.userToken, "GET", uri, nil)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	count := sdk.WorkflowNodeJobRunCount{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &count))
	require.True(t, count.Count > 0)
}

func testGetWorkflowJobAsRegularUser(t *testing.T, api *API, router *Router, ctx testRunWorkflowCtx) {
	uri := router.GetRoute("GET", api.getWorkflowJobQueueHandler, nil)
	test.NotEmpty(t, uri)

	req := assets.NewJWTAuthentifiedRequest(t, ctx.userToken, "GET", uri, nil)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	jobs := []sdk.WorkflowNodeJobRun{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &jobs))
	require.True(t, len(jobs) >= 1)
}

func testGetWorkflowJobAsWorker(t *testing.T, api *API, db gorpmapper.SqlExecutorWithTx, router *Router, ctx testRunWorkflowCtx) {
	uri := router.GetRoute("GET", api.getWorkflowJobQueueHandler, nil)
	require.NotEmpty(t, uri)

	req := assets.NewJWTAuthentifiedRequest(t, ctx.workerToken, "GET", uri, nil)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	jobs := []sdk.WorkflowNodeJobRun{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &jobs))
	require.Len(t, jobs, 1)
}

func testGetWorkflowJobAsHatchery(t *testing.T, api *API, db gorpmapper.SqlExecutorWithTx, router *Router, ctx testRunWorkflowCtx) {
	uri := router.GetRoute("GET", api.getWorkflowJobQueueHandler, nil)
	require.NotEmpty(t, uri)

	req := assets.NewJWTAuthentifiedRequest(t, ctx.hatcheryToken, "GET", uri, nil)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	jobs := []sdk.WorkflowNodeJobRun{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &jobs))
	require.Len(t, jobs, 1)
}

func testRegisterWorker(t *testing.T, api *API, db gorpmapper.SqlExecutorWithTx, router *Router, ctx *testRunWorkflowCtx) {
	g, err := group.LoadByID(context.TODO(), api.mustDB(), ctx.user.Groups[0].ID)
	require.NoError(t, err)
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
	require.NoError(t, err)
	for _, w := range workers {
		require.NoError(t, worker.Delete(db, w.ID))
	}

	// remove all jobs in queue
	filterClean := workflow.NewQueueFilter()
	nrj, err := workflow.LoadNodeJobRunQueue(context.TODO(), db, api.Cache, filterClean)
	require.NoError(t, err)
	for _, j := range nrj {
		require.NoError(t, workflow.DeleteNodeJobRuns(db, j.WorkflowNodeRunID))
	}

	ctx := testRunWorkflow(t, api, router)

	testGetWorkflowJobAsRegularUser(t, api, router, ctx)
	testCountGetWorkflowJobAsRegularUser(t, api, router, ctx)

	testRegisterHatchery(t, api, db, router, &ctx)
	testGetWorkflowJobAsHatchery(t, api, db, router, ctx)

	testRegisterWorker(t, api, db, router, &ctx)
	testGetWorkflowJobAsWorker(t, api, db, router, ctx)

	// Get workflow run number

	uri := router.GetRoute("GET", api.getWorkflowRunNumHandler, map[string]string{
		"key":              ctx.project.Key,
		"permWorkflowName": ctx.workflow.Name,
	})
	require.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, ctx.userToken, "GET", uri, nil)
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
	uri = router.GetRoute("POST", api.postWorkflowRunNumHandler, map[string]string{
		"key":              ctx.project.Key,
		"permWorkflowName": ctx.workflow.Name,
	})
	require.NotEmpty(t, uri)

	n.Num = 10
	req = assets.NewJWTAuthentifiedRequest(t, ctx.userToken, "POST", uri, n)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	uri = router.GetRoute("GET", api.getWorkflowRunNumHandler, map[string]string{
		"key":              ctx.project.Key,
		"permWorkflowName": ctx.workflow.Name,
	})
	require.NotEmpty(t, uri)
	req = assets.NewJWTAuthentifiedRequest(t, ctx.userToken, "GET", uri, nil)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &n))
	require.Equal(t, 10, n.Num)
}

func TestGetWorkflowJobQueueHandler_WithRegions(t *testing.T) {
	api, db, router := newTestAPI(t)

	// Delete all existing workers
	workers, err := worker.LoadAll(context.TODO(), db)
	require.NoError(t, err)
	for _, w := range workers {
		_ = worker.Delete(db, w.ID)
	}

	// Remove all jobs in queue
	filterClean := workflow.NewQueueFilter()
	nrj, _ := workflow.LoadNodeJobRunQueue(context.TODO(), db, api.Cache, filterClean)
	for _, j := range nrj {
		_ = workflow.DeleteNodeJobRuns(db, j.WorkflowNodeRunID)
	}

	res := testRunWorkflow(t, api, router, func(tt *testing.T, tx gorpmapper.SqlExecutorWithTx, pip *sdk.Pipeline, app *sdk.Application) {
		script := assets.GetBuiltinOrPluginActionByName(t, tx, sdk.ScriptAction)

		j2 := &sdk.Job{
			Enabled: true,
			Action: sdk.Action{
				Enabled: true,
				Actions: []sdk.Action{
					assets.NewAction(script.ID, sdk.Parameter{Name: "script", Value: "echo j2"}),
				},
				Requirements: []sdk.Requirement{{
					Name:  "region",
					Type:  sdk.RegionRequirement,
					Value: "test1",
				}},
			},
		}
		require.NoError(tt, pipeline.InsertJob(tx, j2, pip.Stages[0].ID, pip))
		pip.Stages[0].Jobs = append(pip.Stages[0].Jobs, *j2)

		j3 := &sdk.Job{
			Enabled: true,
			Action: sdk.Action{
				Enabled: true,
				Actions: []sdk.Action{
					assets.NewAction(script.ID, sdk.Parameter{Name: "script", Value: "echo j3"}),
				},
				Requirements: []sdk.Requirement{{
					Name:  "region",
					Type:  sdk.RegionRequirement,
					Value: "test2",
				}},
			},
		}
		require.NoError(tt, pipeline.InsertJob(tx, j3, pip.Stages[0].ID, pip))
		pip.Stages[0].Jobs = append(pip.Stages[0].Jobs, *j3)
	})

	test := func(jwt string) func(t *testing.T) {
		return func(t *testing.T) {
			uri := router.GetRoute("GET", api.getWorkflowJobQueueHandler, nil)
			test.NotEmpty(t, uri)
			req := assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil)
			rec := httptest.NewRecorder()
			router.Mux.ServeHTTP(rec, req)
			require.Equal(t, 200, rec.Code)
			jobs := []sdk.WorkflowNodeJobRun{}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &jobs))
			require.Len(t, jobs, 3)
			require.Nil(t, jobs[0].Region)
			require.NotNil(t, jobs[1].Region)
			require.Equal(t, "test1", *jobs[1].Region)
			require.NotNil(t, jobs[2].Region)
			require.Equal(t, "test2", *jobs[2].Region)

			uri = router.GetRoute("GET", api.getWorkflowJobQueueHandler, nil)
			test.NotEmpty(t, uri)
			req = assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil)
			cdsclient.Region("test1", "")(req)
			rec = httptest.NewRecorder()
			router.Mux.ServeHTTP(rec, req)
			require.Equal(t, 200, rec.Code)
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &jobs))
			require.Len(t, jobs, 2)
			require.Nil(t, jobs[0].Region)
			require.NotNil(t, jobs[1].Region)
			require.Equal(t, "test1", *jobs[1].Region)

			uri = router.GetRoute("GET", api.getWorkflowJobQueueHandler, nil)
			test.NotEmpty(t, uri)
			req = assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil)
			cdsclient.Region("test3")(req)
			rec = httptest.NewRecorder()
			router.Mux.ServeHTTP(rec, req)
			require.Equal(t, 200, rec.Code)
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &jobs))
			require.Len(t, jobs, 0)
		}
	}

	_, jwtAdmin := assets.InsertAdminUser(t, db)
	jwtUser := res.userToken
	t.Run("test as admin", test(jwtAdmin))
	t.Run("test as lambda user", test(jwtUser))
}

func Test_postTakeWorkflowJobHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	ctx := testRunWorkflow(t, api, router)

	// Prepare VCS Mock
	mockVCSSservice, _, _ := assets.InitCDNService(t, db)
	defer func() {
		_ = services.Delete(db, mockVCSSservice) // nolint
	}()

	//Register the worker
	testRegisterWorker(t, api, db, router, &ctx)

	uri := router.GetRoute("POST", api.postTakeWorkflowJobHandler, map[string]string{
		"permJobID": fmt.Sprintf("%d", ctx.job.ID),
	})
	require.NotEmpty(t, uri)

	//This will check the needWorker() auth
	req := assets.NewJWTAuthentifiedRequest(t, ctx.userToken, "POST", uri, nil)
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
	assert.Equal(t, true, pbji.GelfServiceAddrEnableTLS)
	require.Len(t, pbji.Secrets, 5)
	var foundDefaultSSHKey, foundDefaultPGPKey bool
	for _, s := range pbji.Secrets {
		if s.Name == "cds.key.proj-ssh-"+strings.ToLower(pbji.ProjectKey)+".priv" {
			foundDefaultSSHKey = true
		}
		if s.Name == "cds.key.proj-pgp-"+strings.ToLower(pbji.ProjectKey)+".priv" {
			foundDefaultPGPKey = true
		}
	}
	require.True(t, foundDefaultSSHKey)
	require.True(t, foundDefaultPGPKey)

	run, err := workflow.LoadNodeJobRun(context.TODO(), api.mustDB(), api.Cache, ctx.job.ID)
	require.NoError(t, err)
	assert.Equal(t, "Building", run.Status)
	assert.Equal(t, ctx.model.Name, run.Model)
	assert.Equal(t, ctx.worker.Name, run.WorkerName)
	assert.NotEmpty(t, run.HatcheryName)
}

func Test_postTakeWorkflowJobWithFilteredSecretHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	api.Config.Secrets.SkipProjectSecretsOnRegion = []string{"test"}

	ctx := testRunWorkflow(t, api, router, func(tt *testing.T, tx gorpmapper.SqlExecutorWithTx, pip *sdk.Pipeline, app *sdk.Application) {
		pip.Stages[0].Jobs[0].Action.Requirements = []sdk.Requirement{{
			Name:  "test",
			Type:  sdk.RegionRequirement,
			Value: "test",
		}, {
			Name:  "cds.proj",
			Type:  sdk.SecretRequirement,
			Value: "^cds.key.proj-ssh-.*.priv$",
		}}
		require.NoError(tt, pipeline.UpdateJob(context.TODO(), tx, &pip.Stages[0].Jobs[0]))
	})

	mockVCSSservice, _, _ := assets.InitCDNService(t, db)
	t.Cleanup(func() {
		_ = services.Delete(db, mockVCSSservice) // nolint
	})

	testRegisterWorker(t, api, db, router, &ctx)

	uri := router.GetRoute("POST", api.postTakeWorkflowJobHandler, map[string]string{
		"permJobID": fmt.Sprintf("%d", ctx.job.ID),
	})
	require.NotEmpty(t, uri)

	req := assets.NewJWTAuthentifiedRequest(t, ctx.workerToken, "POST", uri, nil)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	pbji := &sdk.WorkflowNodeJobRunData{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), pbji))

	require.Len(t, pbji.Secrets, 4)
	var foundDefaultSSHKey, foundDefaultPGPKey bool
	for _, s := range pbji.Secrets {
		if s.Name == "cds.key.proj-ssh-"+strings.ToLower(pbji.ProjectKey)+".priv" {
			foundDefaultSSHKey = true
		}
		if s.Name == "cds.key.proj-pgp-"+strings.ToLower(pbji.ProjectKey)+".priv" {
			foundDefaultPGPKey = true
		}
	}
	require.True(t, foundDefaultSSHKey)
	require.False(t, foundDefaultPGPKey)
}

func Test_postTakeWorkflowInvalidJobHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	s, _, _ := assets.InitCDNService(t, db)
	defer func() {
		_ = services.Delete(db, s)
	}()

	ctx := testRunWorkflow(t, api, router)

	//Register the worker
	testRegisterWorker(t, api, db, router, &ctx)

	uri := router.GetRoute("POST", api.postTakeWorkflowJobHandler, map[string]string{
		"permJobID": fmt.Sprintf("%d", ctx.job.ID+1), // invalid job
	})
	require.NotEmpty(t, uri)

	//this call must failed, we try to take a jobID not reserved at worker's registration
	req := assets.NewJWTAuthentifiedRequest(t, ctx.workerToken, "POST", uri, nil)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 403, rec.Code)

	//This must be ok, take the jobID reserved
	uri2 := router.GetRoute("POST", api.postTakeWorkflowJobHandler, map[string]string{
		"permJobID": fmt.Sprintf("%d", ctx.job.ID),
	})
	require.NotEmpty(t, uri2)
	req2 := assets.NewJWTAuthentifiedRequest(t, ctx.workerToken, "POST", uri2, nil)
	rec2 := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec2, req2)
	require.Equal(t, 200, rec2.Code)
}

func Test_postBookWorkflowJobHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	ctx := testRunWorkflow(t, api, router)

	//Register the hatchery
	testRegisterHatchery(t, api, db, router, &ctx)

	//TakeBook
	uri := router.GetRoute("POST", api.postBookWorkflowJobHandler, map[string]string{
		"permJobID": fmt.Sprintf("%d", ctx.job.ID),
	})
	require.NotEmpty(t, uri)

	req := assets.NewJWTAuthentifiedRequest(t, ctx.hatcheryToken, "POST", uri, nil)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)
}

func Test_postWorkflowJobResultHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	s, _, _ := assets.InitCDNService(t, db)
	defer func() {
		_ = services.Delete(db, s)
	}()

	ctx := testRunWorkflow(t, api, router)

	//Register the worker
	testRegisterWorker(t, api, db, router, &ctx)

	//Take
	uri := router.GetRoute("POST", api.postTakeWorkflowJobHandler, map[string]string{
		"permJobID": fmt.Sprintf("%d", ctx.job.ID),
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
	require.NotEmpty(t, uri)

	req = assets.NewJWTAuthentifiedRequest(t, ctx.workerToken, "POST", uri, res)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 204, rec.Code)

	uri = router.GetRoute("GET", api.getWorkflowRunHandler, map[string]string{
		"key":              ctx.project.Key,
		"permWorkflowName": ctx.workflow.Name,
		"number":           fmt.Sprintf("%d", ctx.run.Number),
	})
	req = assets.NewJWTAuthentifiedRequest(t, ctx.userToken, "GET", uri+"?withDetails=true", res)

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
	req = assets.NewJWTAuthentifiedRequest(t, ctx.userToken, "GET", uri, res)
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

	s, _, _ := assets.InitCDNService(t, db)
	defer func() {
		_ = services.Delete(db, s)
	}()

	ctx := testRunWorkflow(t, api, router)

	// Register the worker
	testRegisterWorker(t, api, db, router, &ctx)
	// Register the hatchery
	testRegisterHatchery(t, api, db, router, &ctx)

	// Send spawninfo
	uri := router.GetRoute("POST", api.postSpawnInfosWorkflowJobHandler, map[string]string{
		"permJobID": fmt.Sprintf("%d", ctx.job.ID),
	})
	require.NotEmpty(t, uri)

	req := assets.NewJWTAuthentifiedRequest(t, ctx.hatcheryToken, "POST", uri, []sdk.SpawnInfo{})
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 204, rec.Code)

	//spawn
	uri = router.GetRoute("POST", api.postTakeWorkflowJobHandler, map[string]string{
		"permJobID": fmt.Sprintf("%d", ctx.job.ID),
	})
	require.NotEmpty(t, uri)

	req = assets.NewJWTAuthentifiedRequest(t, ctx.workerToken, "POST", uri, nil)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	//Send test
	tests := sdk.JUnitTestsSuites{
		TestSuites: []sdk.JUnitTestSuite{
			{
				Name: "TestSuite1",
				TestCases: []sdk.JUnitTestCase{
					{
						Name:   "TestCase1",
						Status: "OK",
					},
				},
			},
			{
				Name: "TestSuite2",
				TestCases: []sdk.JUnitTestCase{
					{
						Name:   "TestCase1",
						Status: "KO",
						Failures: []sdk.JUnitTestFailure{
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
	require.NotEmpty(t, uri)

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
	require.NotEmpty(t, uri)

	req = assets.NewJWTAuthentifiedRequest(t, ctx.workerToken, "POST", uri, step)
	rec = httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 204, rec.Code)

	wNodeJobRun, errJ := workflow.LoadNodeJobRun(context.TODO(), api.mustDB(), api.Cache, ctx.job.ID)
	require.NoError(t, errJ)
	nodeRun, errN := workflow.LoadNodeRunByID(context.Background(), api.mustDB(), wNodeJobRun.WorkflowNodeRunID, workflow.LoadRunOptions{WithTests: true})
	require.NoError(t, errN)

	require.NotNil(t, nodeRun.Tests)
	require.Equal(t, 2, nodeRun.Tests.Total)
	require.Equal(t, 1, nodeRun.Tests.TotalKO)
	require.Equal(t, 1, nodeRun.Tests.TotalOK)
}

func TestWorkerPrivateKey(t *testing.T) {
	api, db, router := newTestAPI(t)

	// Create user
	u, jwtAdmin := assets.InsertAdminUser(t, db)
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
	assert.NoError(t, application.Insert(db, *proj, &app))

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
		user:      u,
		userToken: jwtAdmin,
		project:   proj,
		workflow:  &w,
		run:       wrDB,
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
	u, jwtAdmin := assets.InsertAdminUser(t, db)
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
	assert.NoError(t, application.Insert(db, *proj, &app))

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

	log.Debug(context.TODO(), "%+v", wrDB.WorkflowNodeRuns)

	// Call post coverage report handler
	// Prepare request
	vars := map[string]string{
		"permJobID": fmt.Sprintf("%d", wrDB.WorkflowNodeRuns[w.WorkflowData.Node.ID][0].Stages[0].RunJobs[0].ID),
	}

	ctx := testRunWorkflowCtx{
		user:      u,
		userToken: jwtAdmin,
		project:   proj,
		workflow:  &w,
		run:       wrDB,
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
	u, jwtAdmin := assets.InsertAdminUser(t, db)

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
	assert.NoError(t, application.Insert(db, *proj, &app))
	assert.NoError(t, repositoriesmanager.InsertForApplication(db, &app))

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
	require.NoError(t, err)
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
			wri.Body = io.NopCloser(body)

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
			case "/vcs/repoManServ/repos/foo/bar/branches/?branch=&default=true":
				b := sdk.VCSBranch{
					DisplayID: "master",
					Default:   true,
				}
				if err := enc.Encode(b); err != nil {
					return writeError(wri, err)
				}
			case "/vcs/repoManServ/repos/foo/bar/branches/?branch=master&default=false":
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
			case "/vcs/repoManServ/repos/foo/bar/branches/?branch=my-branch&default=false":
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

	wrr, err := workflow.LoadRunByID(context.Background(), db, wrToTest.ID, workflow.LoadRunOptions{})
	assert.NoError(t, err)

	log.Warn(context.Background(), "%s", wrr.Status)
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
		user:      u,
		userToken: jwtAdmin,
		project:   proj,
		workflow:  &w,
		run:       wrr,
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

	s, _, _ := assets.InitCDNService(t, db)
	defer func() {
		_ = services.Delete(db, s)
	}()

	ctx := testRunWorkflow(t, api, router)

	// Register the worker
	testRegisterWorker(t, api, db, router, &ctx)

	// Take the job
	uri := router.GetRoute("POST", api.postTakeWorkflowJobHandler, map[string]string{
		"permJobID": fmt.Sprintf("%d", ctx.job.ID),
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
	nodeRun, err := workflow.LoadNodeRunByID(context.Background(), db, ctx.job.WorkflowNodeRunID, workflow.LoadRunOptions{})
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
	nodeRun, err = workflow.LoadNodeRunByID(context.Background(), db, ctx.job.WorkflowNodeRunID, workflow.LoadRunOptions{})
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

func Test_workflowRunResultsAdd(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)

	cdnServices, _, jwtCDN := assets.InitCDNService(t, db)
	t.Cleanup(func() { _ = services.Delete(db, cdnServices) })

	u, _ := assets.InsertAdminUser(t, db)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	w := assets.InsertTestWorkflow(t, db, api.Cache, proj, sdk.RandomString(10))

	wrCreate, err := workflow.CreateRun(api.mustDB(), w, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	assert.NoError(t, err)

	require.NoError(t, api.workflowRunCraft(context.TODO(), wrCreate.ID))

	wrDB, err := workflow.LoadRunByID(context.Background(), db, wrCreate.ID, workflow.LoadRunOptions{})
	require.NoError(t, err)

	nr := wrDB.WorkflowNodeRuns[w.WorkflowData.Node.ID][0]
	nr.Status = sdk.StatusBuilding
	require.NoError(t, workflow.UpdateNodeRun(db, &nr))

	nrj := nr.Stages[0].RunJobs[0]
	nrj.Status = sdk.StatusBuilding
	workflow.UpdateNodeJobRun(context.Background(), db, &nrj)

	//Prepare request
	vars := map[string]string{
		"permJobID": fmt.Sprintf("%d", nrj.ID),
	}

	artiData := sdk.WorkflowRunResultArtifact{
		Size:       1,
		MD5:        "AA",
		CDNRefHash: "AA",
		Name:       "myartifact",
		Perm:       0777,
	}
	bts, err := json.Marshal(artiData)
	require.NoError(t, err)
	addResultRequest := sdk.WorkflowRunResult{
		WorkflowRunID:     wrCreate.ID,
		WorkflowNodeRunID: nr.ID,
		WorkflowRunJobID:  nrj.ID,
		Type:              sdk.WorkflowRunResultTypeArtifact,
		DataRaw:           bts,
	}

	uri := router.GetRoute("POST", api.postWorkflowRunResultsHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtCDN, "POST", uri, addResultRequest)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)

	// Can't work because check has not be done
	assert.Equal(t, 403, rec.Code)

	// add check
	require.NoError(t, api.Cache.SetWithTTL(workflow.GetRunResultKey(wrCreate.ID, sdk.WorkflowRunResultTypeArtifact, artiData.Name), true, 60))

	//Do the request
	reqOK := assets.NewJWTAuthentifiedRequest(t, jwtCDN, "POST", uri, addResultRequest)
	recOK := httptest.NewRecorder()
	router.Mux.ServeHTTP(recOK, reqOK)
	assert.Equal(t, 204, recOK.Code)

	b, err := api.Cache.Exist(workflow.GetRunResultKey(wrCreate.ID, sdk.WorkflowRunResultTypeArtifact, artiData.Name))
	require.NoError(t, err)
	require.False(t, b)

}

func Test_workflowRunResultCheckUpload(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)

	u, _ := assets.InsertAdminUser(t, db)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	w := assets.InsertTestWorkflow(t, db, api.Cache, proj, sdk.RandomString(10))

	wrCreate, err := workflow.CreateRun(api.mustDB(), w, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	assert.NoError(t, err)

	require.NoError(t, api.workflowRunCraft(context.TODO(), wrCreate.ID))

	wrDB, err := workflow.LoadRunByID(context.Background(), db, wrCreate.ID, workflow.LoadRunOptions{})
	require.NoError(t, err)

	nr := wrDB.WorkflowNodeRuns[w.WorkflowData.Node.ID][0]
	nr.Status = sdk.StatusBuilding
	require.NoError(t, workflow.UpdateNodeRun(db, &nr))

	nrj := nr.Stages[0].RunJobs[0]
	nrj.Status = sdk.StatusBuilding
	workflow.UpdateNodeJobRun(context.Background(), db, &nrj)

	cdnServices, _, jwtCDN := assets.InitCDNService(t, db)
	t.Cleanup(func() { _ = services.Delete(db, cdnServices) })

	//Prepare request
	vars := map[string]string{
		"permJobID": fmt.Sprintf("%d", nrj.ID),
	}
	checkRequest := sdk.CDNRunResultAPIRef{
		ArtifactName: "myArtifact",
		RunID:        wrCreate.ID,
		RunNodeID:    nr.ID,
		RunJobID:     nrj.ID,
		WorkflowID:   w.ID,
		WorkflowName: w.Name,
		ProjectKey:   key,
	}

	uri := router.GetRoute("POST", api.workflowRunResultCheckUploadHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtCDN, "POST", uri, checkRequest)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 204, rec.Code)
}
