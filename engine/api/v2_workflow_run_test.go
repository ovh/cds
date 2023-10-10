package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/yaml"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestGetWorkflowRunInfoV2Handler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, pwd := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		RepositoryID: repo.ID,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   0,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.StatusBuilding,
		UserID:       admin.ID,
		Username:     admin.Username,
		Event:        sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {},
				"job2": {
					Needs: []string{"job1"},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	infos := sdk.V2WorkflowRunInfo{
		IssuedAt:      time.Now(),
		Level:         sdk.WorkflowRunInfoLevelInfo,
		Message:       "Coucou",
		WorkflowRunID: wr.ID,
	}
	require.NoError(t, workflow_v2.InsertRunInfo(context.TODO(), db, &infos))

	vars := map[string]string{
		"projectKey":           proj.Key,
		"vcsIdentifier":        vcsServer.ID,
		"repositoryIdentifier": repo.ID,
		"workflow":             wr.WorkflowName,
		"runNumber":            strconv.FormatInt(wr.RunNumber, 10),
	}
	uri := api.Router.GetRouteV2("GET", api.getWorkflowRunInfoV2Handler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, "GET", uri, nil)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	infoDB, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)

	require.Equal(t, 1, len(infoDB))
	require.Equal(t, infos.ID, infoDB[0].ID)
}

func TestGetWorkflowRunJobHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, pwd := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		RepositoryID: repo.ID,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   0,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.StatusBuilding,
		UserID:       admin.ID,
		Username:     admin.Username,
		Event:        sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {},
				"job2": {
					Needs: []string{"job1"},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	wrj := sdk.V2WorkflowRunJob{
		Job:           sdk.V2Job{},
		WorkflowRunID: wr.ID,
		Outputs:       sdk.JobResultOutput{},
		UserID:        admin.ID,
		Username:      admin.Username,
		ProjectKey:    wr.ProjectKey,
		JobID:         sdk.RandomString(10),
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj))

	vars := map[string]string{
		"projectKey":           proj.Key,
		"vcsIdentifier":        vcsServer.ID,
		"repositoryIdentifier": repo.ID,
		"workflow":             wr.WorkflowName,
		"runNumber":            strconv.FormatInt(wr.RunNumber, 10),
		"jobName":              wrj.JobID,
	}
	uri := api.Router.GetRouteV2("GET", api.getWorkflowRunJobHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, "GET", uri, nil)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	var rj sdk.V2WorkflowRunJob
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rj))

	require.Equal(t, wrj.ID, rj.ID)
}

func TestGetWorkflowRunJobInfoHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, pwd := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		RepositoryID: repo.ID,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   0,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.StatusBuilding,
		UserID:       admin.ID,
		Username:     admin.Username,
		Event:        sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {},
				"job2": {
					Needs: []string{"job1"},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	wrj := sdk.V2WorkflowRunJob{
		Job:           sdk.V2Job{},
		WorkflowRunID: wr.ID,
		Outputs:       sdk.JobResultOutput{},
		UserID:        admin.ID,
		Username:      admin.Username,
		ProjectKey:    wr.ProjectKey,
		JobID:         sdk.RandomString(10),
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj))

	infos := sdk.V2WorkflowRunJobInfo{
		IssuedAt:         time.Now(),
		Level:            sdk.WorkflowRunInfoLevelInfo,
		Message:          "Coucou",
		WorkflowRunJobID: wrj.ID,
		WorkflowRunID:    wrj.WorkflowRunID,
	}
	require.NoError(t, workflow_v2.InsertRunJobInfo(context.TODO(), db, &infos))

	vars := map[string]string{
		"projectKey":           proj.Key,
		"vcsIdentifier":        vcsServer.ID,
		"repositoryIdentifier": repo.ID,
		"workflow":             wr.WorkflowName,
		"runNumber":            strconv.FormatInt(wr.RunNumber, 10),
		"jobName":              wrj.JobID,
	}
	uri := api.Router.GetRouteV2("GET", api.getWorkflowRunJobInfosHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, "GET", uri, nil)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	infoDB, err := workflow_v2.LoadRunJobInfosByRunJobID(context.TODO(), db, wrj.ID)
	require.NoError(t, err)

	require.Equal(t, 1, len(infoDB))
	require.Equal(t, infos.ID, infoDB[0].ID)
}

func TestPostJobRunStepHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	db.Exec("DELETE FROM rbac")
	db.Exec("DELETE FROM region")

	admin, _ := assets.InsertAdminUser(t, db)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		RepositoryID: repo.ID,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   0,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.StatusBuilding,
		UserID:       admin.ID,
		Username:     admin.Username,
		Event:        sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {},
				"job2": {
					Needs: []string{"job1"},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	hatch := sdk.Hatchery{
		ModelType: "docker",
		Name:      sdk.RandomString(10),
	}
	require.NoError(t, hatchery.Insert(ctx, db, &hatch))

	reg := sdk.Region{Name: "default"}
	require.NoError(t, region.Insert(ctx, db, &reg))

	rbacYaml := `name: perm-default
hatcheries:
- role: %s
  region: default
  hatchery: %s
`
	rbacYaml = fmt.Sprintf(rbacYaml, sdk.HatcheryRoleSpawn, hatch.Name)
	var r sdk.RBAC
	require.NoError(t, yaml.Unmarshal([]byte(rbacYaml), &r))
	r.Hatcheries[0].RegionID = reg.ID
	r.Hatcheries[0].HatcheryID = hatch.ID
	require.NoError(t, rbac.Insert(context.TODO(), db, &r))

	hatchConsumer, err := authentication.NewConsumerHatchery(ctx, db, hatch)
	require.NoError(t, err)

	wrj := sdk.V2WorkflowRunJob{
		Job:           sdk.V2Job{},
		WorkflowRunID: wr.ID,
		Outputs:       sdk.JobResultOutput{},
		UserID:        admin.ID,
		Username:      admin.Username,
		ProjectKey:    wr.ProjectKey,
		JobID:         sdk.RandomString(10),
		Region:        reg.Name,
		HatcheryName:  hatch.Name,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj))

	workerName := sdk.RandomString(10)
	_, jwtWorker := assets.InsertWorker(t, ctx, db, hatchConsumer, hatch, workerName, wrj)

	steps := sdk.JobStepsStatus{}
	steps["job1"] = sdk.JobStepStatus{
		Outcome:    sdk.StatusFail,
		Conclusion: sdk.StatusSuccess,
	}

	uri := api.Router.GetRouteV2("POST", api.postJobRunStepHandler, map[string]string{"regionName": "default", "runJobID": wrj.ID})
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtWorker, "POST", uri, steps)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 204, w.Code)

	runJobDB, err := workflow_v2.LoadRunJobByID(ctx, db, wrj.ID)
	require.NoError(t, err)
	require.Equal(t, 1, len(runJobDB.StepsStatus))
	require.Equal(t, sdk.StatusSuccess, runJobDB.StepsStatus["job1"].Conclusion)

}

func TestGetWorkflowRunJobLogsLinksV2Handler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	db.Exec("DELETE FROM service")

	s := sdk.Service{CanonicalService: sdk.CanonicalService{Type: "cdn", Config: map[string]interface{}{
		"public_http": "http://lolcat.io",
	}}}
	require.NoError(t, services.Insert(context.TODO(), db, &s))

	admin, pwd := assets.InsertAdminUser(t, db)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		RepositoryID: repo.ID,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   0,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.StatusBuilding,
		UserID:       admin.ID,
		Username:     admin.Username,
		Event:        sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {},
				"job2": {
					Needs: []string{"job1"},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	wrj := sdk.V2WorkflowRunJob{
		Job:           sdk.V2Job{},
		WorkflowRunID: wr.ID,
		Outputs:       sdk.JobResultOutput{},
		UserID:        admin.ID,
		Username:      admin.Username,
		ProjectKey:    wr.ProjectKey,
		JobID:         sdk.RandomString(10),
		StepsStatus: sdk.JobStepsStatus{
			"step1": {
				Outcome:    sdk.StatusSuccess,
				Conclusion: sdk.StatusSuccess,
			},
		},
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj))

	vars := map[string]string{
		"projectKey":           proj.Key,
		"vcsIdentifier":        vcsServer.ID,
		"repositoryIdentifier": repo.ID,
		"workflow":             wr.WorkflowName,
		"runNumber":            fmt.Sprintf("%d", wr.RunNumber),
		"jobName":              wrj.JobID,
	}

	// Then Get the region
	uriGet := api.Router.GetRouteV2("GET", api.getWorkflowRunJobLogsLinksV2Handler, vars)
	test.NotEmpty(t, uriGet)
	reqGet := assets.NewAuthentifiedRequest(t, admin, pwd, "GET", uriGet, nil)
	wGet := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wGet, reqGet)
	require.Equal(t, 200, wGet.Code)

	var links sdk.CDNLogLinks
	require.NoError(t, json.Unmarshal(wGet.Body.Bytes(), &links))
	t.Logf("%+v", links)

}

func TestGetWorkflowRunJobsV2Handler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, pwd := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		RepositoryID: repo.ID,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   0,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.StatusBuilding,
		UserID:       admin.ID,
		Username:     admin.Username,
		Event:        sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {},
				"job2": {
					Needs: []string{"job1"},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	wrj := sdk.V2WorkflowRunJob{
		Job:           sdk.V2Job{},
		WorkflowRunID: wr.ID,
		Outputs:       sdk.JobResultOutput{},
		UserID:        admin.ID,
		Username:      admin.Username,
		ProjectKey:    wr.ProjectKey,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj))

	vars := map[string]string{
		"projectKey":           proj.Key,
		"vcsIdentifier":        vcsServer.ID,
		"repositoryIdentifier": repo.ID,
		"workflow":             wr.WorkflowName,
		"runNumber":            fmt.Sprintf("%d", wr.RunNumber),
	}
	// Then Get the region
	uriGet := api.Router.GetRouteV2("GET", api.getWorkflowRunJobsV2Handler, vars)
	test.NotEmpty(t, uriGet)
	reqGet := assets.NewAuthentifiedRequest(t, admin, pwd, "GET", uriGet, nil)
	wGet := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wGet, reqGet)
	require.Equal(t, 200, wGet.Code)

	var jobs []sdk.V2WorkflowRunJob
	require.NoError(t, json.Unmarshal(wGet.Body.Bytes(), &jobs))
	require.Equal(t, 1, len(jobs))
	require.Equal(t, wrj.ID, jobs[0].ID)
}

func TestPostStopWorkflowRunHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, pwd := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		RepositoryID: repo.ID,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   0,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.StatusBuilding,
		UserID:       admin.ID,
		Username:     admin.Username,
		Event:        sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {},
				"job2": {
					Needs: []string{"job1"},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	wrj := sdk.V2WorkflowRunJob{
		Job:           sdk.V2Job{},
		WorkflowRunID: wr.ID,
		Outputs:       sdk.JobResultOutput{},
		UserID:        admin.ID,
		Username:      admin.Username,
		ProjectKey:    wr.ProjectKey,
		Status:        sdk.StatusBuilding,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj))

	vars := map[string]string{
		"projectKey":           proj.Key,
		"vcsIdentifier":        vcsServer.ID,
		"repositoryIdentifier": repo.ID,
		"workflow":             wr.WorkflowName,
		"runNumber":            fmt.Sprintf("%d", wr.RunNumber),
	}
	// Then Get the region
	uri := api.Router.GetRouteV2("POST", api.postStopWorkflowRunHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, "POST", uri, nil)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 204, w.Code)

	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.StatusStopped, wrDB.Status)

	rjDB, err := workflow_v2.LoadRunJobByID(context.TODO(), db, wrj.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.StatusStopped, rjDB.Status)

}

func TestPostStopJobHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, pwd := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		RepositoryID: repo.ID,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   0,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.StatusBuilding,
		UserID:       admin.ID,
		Username:     admin.Username,
		Event:        sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {},
				"job2": {
					Needs: []string{"job1"},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	wrj := sdk.V2WorkflowRunJob{
		JobID:         sdk.RandomString(10),
		Job:           sdk.V2Job{},
		WorkflowRunID: wr.ID,
		Outputs:       sdk.JobResultOutput{},
		UserID:        admin.ID,
		Username:      admin.Username,
		ProjectKey:    wr.ProjectKey,
		Status:        sdk.StatusBuilding,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj))

	vars := map[string]string{
		"projectKey":           proj.Key,
		"vcsIdentifier":        vcsServer.ID,
		"repositoryIdentifier": repo.ID,
		"workflow":             wr.WorkflowName,
		"runNumber":            fmt.Sprintf("%d", wr.RunNumber),
		"jobName":              wrj.JobID,
	}
	// Then Get the region
	uri := api.Router.GetRouteV2("POST", api.postStopJobHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, "POST", uri, nil)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 204, w.Code)

	// Workflow must be re-enqueued
	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.StatusBuilding, wrDB.Status)

	rjDB, err := workflow_v2.LoadRunJobByID(context.TODO(), db, wrj.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.StatusStopped, rjDB.Status)

}

func TestPostWorkflowRunHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, pwd := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	e := sdk.Entity{
		Name:                sdk.RandomString(10),
		Type:                sdk.EntityTypeWorkflow,
		ProjectKey:          proj.Key,
		Branch:              "master",
		Commit:              "123456",
		ProjectRepositoryID: repo.ID,
		Data: `name: MyFirstWorkflow
jobs:
  myFirstJob:
    name: This is my first job
    worker_model: buildpack-deps-buster
    region: default
    steps:
      - run: |-
          echo "It is my first step"`,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &e))

	vars := map[string]string{
		"projectKey":           proj.Key,
		"vcsIdentifier":        vcsServer.ID,
		"repositoryIdentifier": repo.ID,
		"workflow":             e.Name,
	}

	uri := api.Router.GetRouteV2("POST", api.postWorkflowRunV2Handler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, "POST", uri+"?branch=master", map[string]interface{}{
		"branch": "main",
	})
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 201, w.Code)

	var wr sdk.V2WorkflowRun
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &wr))
	require.Equal(t, sdk.StatusCrafting, wr.Status)
}
