package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"sync"
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
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/yaml"
	"github.com/stretchr/testify/require"
)

// runJobEventsRecorder keeps the run job events published through the cache; everything else reaches the store.
type runJobEventsRecorder struct {
	cache.Store
	mu     sync.Mutex
	events []sdk.WorkflowRunJobEvent
}

func (r *runJobEventsRecorder) Enqueue(queueName string, value interface{}) error {
	if e, ok := value.(sdk.WorkflowRunJobEvent); ok {
		r.mu.Lock()
		r.events = append(r.events, e)
		r.mu.Unlock()
		return nil
	}
	return r.Store.Enqueue(queueName, value)
}

// waitFor blocks until an event of the given type is published for the run job; the handlers publish after answering.
func (r *runJobEventsRecorder) waitFor(t *testing.T, runJobID string, eventType sdk.EventType) {
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if r.count(runJobID, eventType) > 0 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("no %s event published for run job %s", eventType, runJobID)
}

func (r *runJobEventsRecorder) count(runJobID string, eventType sdk.EventType) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, e := range r.events {
		if e.RunJobID == runJobID && e.Type == eventType {
			n++
		}
	}
	return n
}

func TestPostHatcheryTakeAndReleaseJobRunHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	api.Config.WorkflowV2.JobSchedulingMaxErrors = 2
	events := &runJobEventsRecorder{Store: api.Cache}
	api.Cache = events

	db.Exec("DELETE FROM rbac")
	db.Exec("DELETE FROM region")

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, "myrepo")

	wkfName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		Status:           sdk.V2WorkflowRunStatusBuilding,
		ProjectKey:       proj.Key,
		DeprecatedUserID: admin.ID,
		WorkflowName:     wkfName,
		RepositoryID:     repo.ID,
		VCSServerID:      vcsServer.ID,
		VCSServer:        vcsServer.Name,
		Repository:       repo.Name,
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	jobRun := sdk.V2WorkflowRunJob{
		ProjectKey:    proj.Key,
		Status:        sdk.V2WorkflowRunJobStatusWaiting,
		ModelType:     "docker",
		ModelOSArch:   "linux/amd64",
		Region:        "default",
		WorkflowRunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
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
	require.Equal(t, sdk.V2WorkflowRunJobStatusScheduling, jobRunResponse.Status)

	// Release: the job goes back to the queue, without the trace of its scheduling
	uriRelease := api.Router.GetRouteV2("DELETE", api.deleteHatcheryReleaseJobRunHandler, vars)
	test.NotEmpty(t, uriRelease)
	reqRelease := assets.NewJWTAuthentifiedRequest(t, jwt, "DELETE", uriRelease, nil)
	wRelease := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wRelease, reqRelease)
	require.Equal(t, 204, wRelease.Code)

	jobRunDB, err := workflow_v2.LoadRunJobByRunIDAndID(ctx, db, wr.ID, jobRun.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.V2WorkflowRunJobStatusWaiting, jobRunDB.Status)
	require.Equal(t, "", jobRunDB.HatcheryName)
	require.Nil(t, jobRunDB.Scheduled)
	require.Nil(t, jobRunDB.Ended)
	events.waitFor(t, jobRun.ID, sdk.EventRunJobEnqueued)

	// Take and release again: the last allowed attempt fails the job for good
	req = assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, nil)
	w = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	reqRelease = assets.NewJWTAuthentifiedRequest(t, jwt, "DELETE", uriRelease, nil)
	wRelease = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wRelease, reqRelease)
	require.Equal(t, 204, wRelease.Code)

	jobRunDB, err = workflow_v2.LoadRunJobByRunIDAndID(ctx, db, wr.ID, jobRun.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.V2WorkflowRunJobStatusFail, jobRunDB.Status)
	require.Equal(t, "", jobRunDB.HatcheryName)
	require.NotNil(t, jobRunDB.Scheduled)
	require.NotNil(t, jobRunDB.Ended)
	events.waitFor(t, jobRun.ID, sdk.EventRunJobEnded)
	require.Equal(t, 1, events.count(jobRun.ID, sdk.EventRunJobEnqueued))
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
		Status:           sdk.V2WorkflowRunStatusBuilding,
		ProjectKey:       proj.Key,
		DeprecatedUserID: admin.ID,
		WorkflowName:     wkfName,
		RepositoryID:     repo.ID,
		VCSServerID:      vcsServer.ID,
		VCSServer:        vcsServer.Name,
		Repository:       repo.Name,
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
		Status:        sdk.V2WorkflowRunJobStatusScheduling,
		ModelType:     "docker",
		ModelOSArch:   "linux/amd64",
		Region:        "default",
		WorkflowRunID: wr.ID,
		HatcheryName:  hatch.Name,
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
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
		Status: sdk.V2WorkflowRunJobStatusFail,
		Error:  "unable to craft job",
	}
	vars := map[string]string{"runJobID": jobRun.ID, "regionName": "default"}
	uri := api.Router.GetRouteV2("POST", api.postJobResultHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "POST", uri, jobResult)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 204, w.Code)

	jobRunDB, err := workflow_v2.LoadRunJobByRunIDAndID(ctx, db, wr.ID, jobRun.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.V2WorkflowRunJobStatusFail, jobRunDB.Status)
}

func TestUserGetJobsQueuedHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	db.Exec("DELETE FROM rbac")
	db.Exec("DELETE FROM region")
	db.Exec("DELETE FROM v2_workflow_run_job")

	admin, _ := assets.InsertAdminUser(t, db)
	lambda, pwd := assets.InsertLambdaUser(t, db)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	proj2 := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, "myrepo")

	wkfName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		Status:           sdk.V2WorkflowRunStatusBuilding,
		ProjectKey:       proj.Key,
		DeprecatedUserID: admin.ID,
		WorkflowName:     wkfName,
		RepositoryID:     repo.ID,
		VCSServerID:      vcsServer.ID,
		VCSServer:        vcsServer.Name,
		Repository:       repo.Name,
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	jobRun := sdk.V2WorkflowRunJob{
		ProjectKey:    proj.Key,
		Status:        sdk.V2WorkflowRunJobStatusWaiting,
		JobID:         "job1",
		ModelType:     "docker",
		ModelOSArch:   "linux/amd64",
		Region:        "default",
		WorkflowRunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
	}
	require.NoError(t, workflow_v2.InsertRunJob(ctx, db, &jobRun))

	jobRun2 := sdk.V2WorkflowRunJob{
		ProjectKey:    proj2.Key,
		Status:        sdk.V2WorkflowRunJobStatusWaiting,
		JobID:         "job2",
		ModelType:     "docker",
		Region:        "default",
		WorkflowRunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
	}
	require.NoError(t, workflow_v2.InsertRunJob(ctx, db, &jobRun2))

	reg := sdk.Region{Name: "default"}
	require.NoError(t, region.Insert(context.TODO(), db, &reg))

	assets.InsertRBAcProject(t, db, sdk.ProjectRoleRead, proj.Key, *lambda)
	assets.InsertRBAcRegion(t, db, "default", "default", sdk.RegionRoleExecute, *lambda)

	// Get jobs
	uri := api.Router.GetRouteV2("GET", api.getJobsQueuedHandler, map[string]string{})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, lambda, pwd, "GET", uri, nil)

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
	var jobRunResponse []sdk.V2WorkflowRunJob
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &jobRunResponse))
	require.Equal(t, 1, len(jobRunResponse))
	require.Equal(t, "job1", jobRunResponse[0].JobID)
}

func TestGetJobsRegionalizedQueuedHandler(t *testing.T) {
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
		Status:           sdk.V2WorkflowRunStatusBuilding,
		ProjectKey:       proj.Key,
		DeprecatedUserID: admin.ID,
		WorkflowName:     wkfName,
		RepositoryID:     repo.ID,
		VCSServerID:      vcsServer.ID,
		VCSServer:        vcsServer.Name,
		Repository:       repo.Name,
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	jobRun := sdk.V2WorkflowRunJob{
		ProjectKey:    proj.Key,
		Status:        sdk.V2WorkflowRunJobStatusWaiting,
		JobID:         "job1",
		ModelType:     "docker",
		ModelOSArch:   "linux/amd64",
		Region:        "default",
		WorkflowRunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
	}
	require.NoError(t, workflow_v2.InsertRunJob(ctx, db, &jobRun))

	jobRun2 := sdk.V2WorkflowRunJob{
		ProjectKey:    proj.Key,
		Status:        sdk.V2WorkflowRunJobStatusWaiting,
		JobID:         "job2",
		ModelType:     "docker",
		Region:        "default2",
		WorkflowRunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
	}
	require.NoError(t, workflow_v2.InsertRunJob(ctx, db, &jobRun2))

	jobRun3 := sdk.V2WorkflowRunJob{
		ProjectKey:    proj.Key,
		Status:        sdk.V2WorkflowRunJobStatusWaiting,
		JobID:         "job3",
		ModelType:     "openstack",
		Region:        "default",
		WorkflowRunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
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
	uri := api.Router.GetRouteV2("GET", api.getJobsQueuedRegionalizedHandler, map[string]string{"regionName": "default"})
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
		Status:           sdk.V2WorkflowRunStatusBuilding,
		ProjectKey:       proj.Key,
		DeprecatedUserID: admin.ID,
		WorkflowName:     wkfName,
		RepositoryID:     repo.ID,
		VCSServerID:      vcsServer.ID,
		VCSServer:        vcsServer.Name,
		Repository:       repo.Name,
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	jobRun := sdk.V2WorkflowRunJob{
		ProjectKey:    proj.Key,
		Status:        sdk.V2WorkflowRunJobStatusWaiting,
		JobID:         "job1",
		ModelType:     "docker",
		ModelOSArch:   "linux/amd64",
		Region:        "default",
		WorkflowRunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
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
	uri := api.Router.GetRouteV2("GET", api.getJobRunQueueInfoHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwt, "GET", uri, nil)

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
	var jobRunResponse sdk.V2QueueJobInfo
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &jobRunResponse))
	require.Equal(t, "job1", jobRunResponse.RunJob.JobID)
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
		Status:           sdk.V2WorkflowRunStatusBuilding,
		ProjectKey:       proj.Key,
		DeprecatedUserID: admin.ID,
		WorkflowName:     wkfName,
		RepositoryID:     repo.ID,
		VCSServerID:      vcsServer.ID,
		VCSServer:        vcsServer.Name,
		Repository:       repo.Name,
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	jobRun := sdk.V2WorkflowRunJob{
		ProjectKey:    proj.Key,
		Status:        sdk.V2WorkflowRunJobStatusWaiting,
		JobID:         "job1",
		ModelType:     "docker",
		ModelOSArch:   "linux/amd64",
		Region:        "default",
		WorkflowRunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
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
