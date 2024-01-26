package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/authentication"
	authhatch "github.com/ovh/cds/engine/api/authentication/hatchery"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/yaml"
	"github.com/stretchr/testify/require"
)

func TestPostHatcheryTakeAndReleaseJobRunHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	db.Exec("DELETE FROM rbac")
	db.Exec("DELETE FROM region")

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, "myrepo")

	wkfName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		Status:       sdk.StatusBuilding,
		ProjectKey:   proj.Key,
		UserID:       admin.ID,
		WorkflowName: wkfName,
		RepositoryID: repo.ID,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		Repository:   repo.Name,
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	jobRun := sdk.V2WorkflowRunJob{
		ProjectKey:    proj.Key,
		UserID:        admin.ID,
		Status:        sdk.StatusWaiting,
		ModelType:     "docker",
		Region:        "default",
		WorkflowRunID: wr.ID,
	}
	require.NoError(t, workflow_v2.InsertRunJob(ctx, db, &jobRun))

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

	consumer, err := authentication.NewConsumerHatchery(ctx, db, hatch)
	require.NoError(t, err)

	session, err := authentication.NewSession(context.TODO(), db, &consumer.AuthConsumer, authhatch.SessionDuration)
	require.NoError(t, err)
	jwt, err := authentication.NewSessionJWT(session, "")
	require.NoError(t, err)

	// Take Job
	vars := map[string]string{"runJobID": jobRun.ID, "regionName": "default"}
	uri := api.Router.GetRouteV2("POST", api.postHatcheryTakeJobRunHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, nil)

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
	var jobRunResponse sdk.V2WorkflowRunJob
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &jobRunResponse))
	require.Equal(t, sdk.StatusScheduling, jobRunResponse.Status)

	// release

	uriRelease := api.Router.GetRouteV2("DELETE", api.deleteHatcheryReleaseJobRunHandler, vars)
	test.NotEmpty(t, uri)
	reqRelease := assets.NewJWTAuthentifiedRequest(t, jwt, "DELETE", uriRelease, nil)
	wRelease := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wRelease, reqRelease)
	require.Equal(t, 200, w.Code)

	jobRunDB, err := workflow_v2.LoadRunJobByID(ctx, db, jobRun.ID)
	require.NoError(t, err)
	require.Equal(t, jobRunDB.Status, sdk.StatusWaiting)
	require.Equal(t, jobRunDB.HatcheryName, "")

}

func TestPostJobResultHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	db.Exec("DELETE FROM rbac")
	db.Exec("DELETE FROM region")

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, "myrepo")

	reg := sdk.Region{Name: "default"}
	require.NoError(t, region.Insert(ctx, db, &reg))

	wkfName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		Status:       sdk.StatusBuilding,
		ProjectKey:   proj.Key,
		UserID:       admin.ID,
		WorkflowName: wkfName,
		RepositoryID: repo.ID,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		Repository:   repo.Name,
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	hatch := sdk.Hatchery{
		ModelType: "docker",
		Name:      sdk.RandomString(10),
	}
	require.NoError(t, hatchery.Insert(ctx, db, &hatch))

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

	jobRun := sdk.V2WorkflowRunJob{
		ProjectKey:    proj.Key,
		UserID:        admin.ID,
		Status:        sdk.StatusScheduling,
		ModelType:     "docker",
		Region:        "default",
		WorkflowRunID: wr.ID,
		HatcheryName:  hatch.Name,
	}
	require.NoError(t, workflow_v2.InsertRunJob(ctx, db, &jobRun))

	consumer, err := authentication.NewConsumerHatchery(ctx, db, hatch)
	require.NoError(t, err)

	session, err := authentication.NewSession(context.TODO(), db, &consumer.AuthConsumer, authhatch.SessionDuration)
	require.NoError(t, err)
	jwt, err := authentication.NewSessionJWT(session, "")
	require.NoError(t, err)

	// Take Job
	jobResult := sdk.V2WorkflowRunJobResult{
		Status: sdk.StatusFail,
		Error:  "unable to craft job",
	}
	vars := map[string]string{"runJobID": jobRun.ID, "regionName": "default"}
	uri := api.Router.GetRouteV2("POST", api.postJobResultHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, jobResult)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 204, w.Code)

	jobRunDB, err := workflow_v2.LoadRunJobByID(ctx, db, jobRun.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.StatusFail, jobRunDB.Status)
}

func TestGetJobsQueuedHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	db.Exec("DELETE FROM rbac")
	db.Exec("DELETE FROM region")
	db.Exec("DELETE FROM v2_workflow_run_job")

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, "myrepo")

	wkfName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		Status:       sdk.StatusBuilding,
		ProjectKey:   proj.Key,
		UserID:       admin.ID,
		WorkflowName: wkfName,
		RepositoryID: repo.ID,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		Repository:   repo.Name,
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	jobRun := sdk.V2WorkflowRunJob{
		ProjectKey:    proj.Key,
		UserID:        admin.ID,
		Status:        sdk.StatusWaiting,
		JobID:         "job1",
		ModelType:     "docker",
		Region:        "default",
		WorkflowRunID: wr.ID,
	}
	require.NoError(t, workflow_v2.InsertRunJob(ctx, db, &jobRun))

	jobRun2 := sdk.V2WorkflowRunJob{
		ProjectKey:    proj.Key,
		UserID:        admin.ID,
		Status:        sdk.StatusWaiting,
		JobID:         "job2",
		ModelType:     "docker",
		Region:        "default2",
		WorkflowRunID: wr.ID,
	}
	require.NoError(t, workflow_v2.InsertRunJob(ctx, db, &jobRun2))

	jobRun3 := sdk.V2WorkflowRunJob{
		ProjectKey:    proj.Key,
		UserID:        admin.ID,
		Status:        sdk.StatusWaiting,
		JobID:         "job3",
		ModelType:     "openstack",
		Region:        "default",
		WorkflowRunID: wr.ID,
	}
	require.NoError(t, workflow_v2.InsertRunJob(ctx, db, &jobRun3))

	hatch := sdk.Hatchery{
		ModelType: "docker",
		Name:      sdk.RandomString(10),
	}
	require.NoError(t, hatchery.Insert(ctx, db, &hatch))

	reg := sdk.Region{Name: "default"}
	require.NoError(t, region.Insert(ctx, db, &reg))

	reg2 := sdk.Region{Name: "default2"}
	require.NoError(t, region.Insert(ctx, db, &reg2))

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

	consumer, err := authentication.NewConsumerHatchery(ctx, db, hatch)
	require.NoError(t, err)

	session, err := authentication.NewSession(context.TODO(), db, &consumer.AuthConsumer, authhatch.SessionDuration)
	require.NoError(t, err)
	jwt, err := authentication.NewSessionJWT(session, "")
	require.NoError(t, err)

	// Take Job
	uri := api.Router.GetRouteV2("GET", api.getJobsQueuedHandler, map[string]string{"regionName": "default"})
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil)

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
	var jobRunResponse []sdk.V2WorkflowRunJob
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &jobRunResponse))
	require.Equal(t, 1, len(jobRunResponse))
	require.Equal(t, "job1", jobRunResponse[0].JobID)
}

func TestGetJobHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	db.Exec("DELETE FROM rbac")
	db.Exec("DELETE FROM region")
	db.Exec("DELETE FROM v2_workflow_run_job")
	db.Exec("DELETE FROM hatchery")

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, "myrepo")

	wkfName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		Status:       sdk.StatusBuilding,
		ProjectKey:   proj.Key,
		UserID:       admin.ID,
		WorkflowName: wkfName,
		RepositoryID: repo.ID,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		Repository:   repo.Name,
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	jobRun := sdk.V2WorkflowRunJob{
		ProjectKey:    proj.Key,
		UserID:        admin.ID,
		Status:        sdk.StatusWaiting,
		JobID:         "job1",
		ModelType:     "docker",
		Region:        "default",
		WorkflowRunID: wr.ID,
	}
	require.NoError(t, workflow_v2.InsertRunJob(ctx, db, &jobRun))

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

	consumer, err := authentication.NewConsumerHatchery(ctx, db, hatch)
	require.NoError(t, err)

	session, err := authentication.NewSession(context.TODO(), db, &consumer.AuthConsumer, authhatch.SessionDuration)
	require.NoError(t, err)
	jwt, err := authentication.NewSessionJWT(session, "")
	require.NoError(t, err)

	// Take Job
	vars := map[string]string{
		"runJobID":   jobRun.ID,
		"regionName": "default",
	}
	uri := api.Router.GetRouteV2("GET", api.getJobRunHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil)

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
	var jobRunResponse sdk.V2WorkflowRunJob
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &jobRunResponse))
	require.Equal(t, "job1", jobRunResponse.JobID)
}

func TestPostJobRunInfoHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	db.Exec("DELETE FROM rbac")
	db.Exec("DELETE FROM region")
	db.Exec("DELETE FROM v2_workflow_run_job")
	db.Exec("DELETE FROM hatchery")

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, "myrepo")

	wkfName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		Status:       sdk.StatusBuilding,
		ProjectKey:   proj.Key,
		UserID:       admin.ID,
		WorkflowName: wkfName,
		RepositoryID: repo.ID,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		Repository:   repo.Name,
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	jobRun := sdk.V2WorkflowRunJob{
		ProjectKey:    proj.Key,
		UserID:        admin.ID,
		Status:        sdk.StatusWaiting,
		JobID:         "job1",
		ModelType:     "docker",
		Region:        "default",
		WorkflowRunID: wr.ID,
	}
	require.NoError(t, workflow_v2.InsertRunJob(ctx, db, &jobRun))

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

	consumer, err := authentication.NewConsumerHatchery(ctx, db, hatch)
	require.NoError(t, err)

	workerName := "worker" + sdk.RandomString(10)
	_, jwtWorker := assets.InsertWorker(t, ctx, db, consumer, hatch, workerName, jobRun)

	infoToSend := sdk.V2SendJobRunInfo{
		Time:    time.Now(),
		Level:   sdk.WorkflowRunInfoLevelInfo,
		Message: "My message",
	}
	vars := map[string]string{
		"runJobID":   jobRun.ID,
		"regionName": "default",
	}
	uri := api.Router.GetRouteV2("POST", api.postJobRunInfoHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtWorker, "POST", uri, infoToSend)

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 204, w.Code)

	// Get run info from db
	info, err := workflow_v2.LoadRunJobInfosByRunJobID(ctx, db, jobRun.ID)
	require.NoError(t, err)
	require.Len(t, info, 1)
	require.Equal(t, infoToSend.Message, info[0].Message)
}
