package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/golang/mock/gomock"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/entity"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/yaml"
	"github.com/stretchr/testify/require"
)

func TestRunManualJob_WrongGateReviewer(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, pwd := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   0,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.StatusSuccess,
		UserID:       admin.ID,
		Username:     admin.Username,
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Gates: map[string]sdk.V2JobGate{
				"preprod": {
					If: "${{gate.approve}}",
					Inputs: map[string]sdk.V2JobGateInput{
						"approve": {
							Type: "boolean",
						},
					},
					Reviewers: sdk.V2JobGateReviewers{
						Users: []string{"foo.bar"},
					},
				},
			},
			Jobs: map[string]sdk.V2Job{
				"job1": {},
				"job2": {
					Gate:  "preprod",
					Needs: []string{"job1"},
				},
				"job3": {
					Needs: []string{"job1"},
				},
				"job4": {
					Needs: []string{"job2"},
				},
				"job5": {
					Needs: []string{"job3"},
				},
				"job6": {
					Needs: []string{"job4", "job5"},
				},
				"job7": {
					Needs: []string{"job6"},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	wrjJob1 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job1",
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob1))

	wrjJob2 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusSkipped,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job2",
		Job:           wr.WorkflowData.Workflow.Jobs["job2"],
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob2))

	wrjJob3 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job3",
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob3))

	wrjJob4 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job4",
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob4))

	wrjJob5 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusFail,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job5",
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob5))

	wrjJob6 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusFail,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job6",
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob6))

	wrjJob7 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job7",
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob7))

	// Mock Hook
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeHooks)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/item/duplicate", gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(3)

	uri := api.Router.GetRouteV2("PUT", api.putWorkflowRunJobV2Handler, map[string]string{
		"projectKey":    proj.Key,
		"runIdentifier": wr.ID,
		"jobIdentifier": "job2",
	})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, "PUT", uri, map[string]interface{}{
		"approve": true,
	})
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 403, w.Code)
}

func TestRunManualJob_WrongGateCondition(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, pwd := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   0,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.StatusSuccess,
		UserID:       admin.ID,
		Username:     admin.Username,
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Gates: map[string]sdk.V2JobGate{
				"preprod": {
					If: "${{gate.approve}}",
					Inputs: map[string]sdk.V2JobGateInput{
						"approve": {
							Type: "boolean",
						},
					},
				},
			},
			Jobs: map[string]sdk.V2Job{
				"job1": {},
				"job2": {
					Gate:  "preprod",
					Needs: []string{"job1"},
				},
				"job3": {
					Needs: []string{"job1"},
				},
				"job4": {
					Needs: []string{"job2"},
				},
				"job5": {
					Needs: []string{"job3"},
				},
				"job6": {
					Needs: []string{"job4", "job5"},
				},
				"job7": {
					Needs: []string{"job6"},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	wrjJob1 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job1",
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob1))

	wrjJob2 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusSkipped,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job2",
		Job:           wr.WorkflowData.Workflow.Jobs["job2"],
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob2))

	wrjJob3 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job3",
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob3))

	wrjJob4 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job4",
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob4))

	wrjJob5 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusFail,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job5",
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob5))

	wrjJob6 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusFail,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job6",
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob6))

	wrjJob7 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job7",
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob7))

	// Mock Hook
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeHooks)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/item/duplicate", gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(3)

	uri := api.Router.GetRouteV2("PUT", api.putWorkflowRunJobV2Handler, map[string]string{
		"projectKey":    proj.Key,
		"runIdentifier": wr.ID,
		"jobIdentifier": "job2",
	})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, "PUT", uri, map[string]interface{}{
		"approve": false,
	})
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 403, w.Code)
}

func TestRunManualJob(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, err := db.Exec("delete from region where name = 'build-test'")
	require.NoError(t, err)

	reg := sdk.Region{Name: "build-test"}
	require.NoError(t, region.Insert(context.TODO(), db, &reg))

	admin, pwd := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   0,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.StatusSuccess,
		UserID:       admin.ID,
		Username:     admin.Username,
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Gates: map[string]sdk.V2JobGate{
				"preprod": {
					If: "${{gate.approve}}",
					Inputs: map[string]sdk.V2JobGateInput{
						"approve": {
							Type: "boolean",
						},
					},
				},
			},
			Jobs: map[string]sdk.V2Job{
				"job1": {},
				"job2": {
					Gate:   "preprod",
					Region: "build-test",
					Needs:  []string{"job1"},
				},
				"job3": {
					Needs: []string{"job1"},
				},
				"job4": {
					Needs: []string{"job2"},
				},
				"job5": {
					Needs: []string{"job3"},
				},
				"job6": {
					Needs: []string{"job4", "job5"},
				},
				"job7": {
					Needs: []string{"job6"},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	wrjJob1 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job1",
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob1))

	wrjJob2 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusSkipped,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job2",
		Job:           wr.WorkflowData.Workflow.Jobs["job2"],
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob2))

	wrjJob3 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job3",
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob3))

	wrjJob4 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job4",
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob4))

	wrjJob5 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusFail,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job5",
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob5))

	wrjJob6 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusFail,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job6",
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob6))

	wrjJob7 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job7",
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob7))

	// Mock Hook
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeHooks)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/item/duplicate", gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(3)

	uri := api.Router.GetRouteV2("PUT", api.putWorkflowRunJobV2Handler, map[string]string{
		"projectKey":    proj.Key,
		"runIdentifier": wr.ID,
		"jobIdentifier": "job2",
	})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, "PUT", uri, map[string]interface{}{
		"approve": true,
	})
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	// trigger jobs
	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID:  wr.ID,
		UserID: admin.ID,
	}))

	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.StatusBuilding, wrDB.Status)
	require.Equal(t, int64(2), wrDB.RunAttempt)

	runjobs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wrDB.ID, wrDB.RunAttempt)
	require.NoError(t, err)
	require.Equal(t, 4, len(runjobs))

	mapJob := make(map[string]sdk.V2WorkflowRunJob)
	for _, rj := range runjobs {
		mapJob[rj.JobID] = rj
	}

	rJob1, has := mapJob["job1"]
	require.True(t, has)
	require.Equal(t, int64(2), rJob1.RunAttempt)

	rJob3, has := mapJob["job3"]
	require.True(t, has)
	require.Equal(t, int64(2), rJob3.RunAttempt)

	rJob5, has := mapJob["job5"]
	require.True(t, has)
	require.Equal(t, int64(2), rJob5.RunAttempt)

	rJob2, has := mapJob["job2"]
	require.True(t, has)
	require.Equal(t, 1, len(rJob2.GateInputs))
	v, has := rJob2.GateInputs["approve"]
	require.True(t, has)
	require.Equal(t, "true", fmt.Sprintf("%v", v))
}

func TestPutWorkflowRun(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, pwd := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   0,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.StatusFail,
		UserID:       admin.ID,
		Username:     admin.Username,
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {},
				"job2": {
					Needs: []string{"job1"},
				},
				"job3": {
					Needs: []string{"job1"},
				},
				"job4": {
					Needs: []string{"job2"},
				},
				"job5": {
					Needs: []string{"job3"},
				},
				"job6": {
					Needs: []string{"job4", "job5"},
				},
				"job7": {
					Needs: []string{"job6"},
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	wrjJob1 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job1",
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob1))

	wrjJob2 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusFail,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job2",
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob2))

	wrjJob3 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job3",
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob3))

	wrjJob4 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job4",
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob4))

	wrjJob5 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusFail,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job5",
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob5))

	wrjJob6 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusFail,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job6",
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob6))

	wrjJob7 := sdk.V2WorkflowRunJob{
		Status:        sdk.StatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job7",
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob7))

	// Mock Hook
	s, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeHooks)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/item/duplicate", gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(2)

	uri := api.Router.GetRouteV2("PUT", api.putWorkflowRunV2Handler, map[string]string{
		"projectKey":    proj.Key,
		"runIdentifier": wr.ID,
		"runNumber":     strconv.FormatInt(wr.RunNumber, 10),
	})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, "PUT", uri, nil)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.StatusBuilding, wrDB.Status)
	require.Equal(t, int64(2), wrDB.RunAttempt)

	runjobs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wrDB.ID, wrDB.RunAttempt)
	require.NoError(t, err)
	require.Equal(t, 2, len(runjobs))

	mapJob := make(map[string]sdk.V2WorkflowRunJob)
	for _, rj := range runjobs {
		mapJob[rj.JobID] = rj
	}

	rJob1, has := mapJob["job1"]
	require.True(t, has)
	require.Equal(t, int64(2), rJob1.RunAttempt)

	rJob3, has := mapJob["job3"]
	require.True(t, has)
	require.Equal(t, int64(2), rJob3.RunAttempt)
}

func TestPutWorkflowRun_BuildingRun(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, pwd := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
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
		RunEvent:     sdk.V2WorkflowRunEvent{},
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

	vars := map[string]string{
		"projectKey":           proj.Key,
		"vcsIdentifier":        vcsServer.ID,
		"repositoryIdentifier": repo.ID,
		"workflow":             wr.WorkflowName,
		"runNumber":            strconv.FormatInt(wr.RunNumber, 10),
	}
	uri := api.Router.GetRouteV2("PUT", api.putWorkflowRunV2Handler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, "PUT", uri, nil)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 400, w.Code)
}

func TestPutWorkflowRun_NoFailingJob(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, pwd := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   0,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.StatusSuccess,
		UserID:       admin.ID,
		Username:     admin.Username,
		RunEvent:     sdk.V2WorkflowRunEvent{},
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

	vars := map[string]string{
		"projectKey":           proj.Key,
		"vcsIdentifier":        vcsServer.ID,
		"repositoryIdentifier": repo.ID,
		"workflow":             wr.WorkflowName,
		"runNumber":            strconv.FormatInt(wr.RunNumber, 10),
	}
	uri := api.Router.GetRouteV2("PUT", api.putWorkflowRunV2Handler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, "PUT", uri, nil)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 400, w.Code)
}

func TestGetWorkflowRunInfoV2Handler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, pwd := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
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
		RunEvent:     sdk.V2WorkflowRunEvent{},
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

	uri := api.Router.GetRouteV2("GET", api.getWorkflowRunInfoV2Handler, map[string]string{
		"projectKey":    proj.Key,
		"runIdentifier": wr.ID,
		"runNumber":     strconv.FormatInt(wr.RunNumber, 10),
	})
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
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   1,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.StatusBuilding,
		UserID:       admin.ID,
		Username:     admin.Username,
		RunEvent:     sdk.V2WorkflowRunEvent{},
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
		UserID:        admin.ID,
		Username:      admin.Username,
		ProjectKey:    wr.ProjectKey,
		RunAttempt:    wr.RunAttempt,
		JobID:         sdk.RandomString(10),
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj))

	uri := api.Router.GetRouteV2("GET", api.getWorkflowRunJobHandler, map[string]string{
		"projectKey":    proj.Key,
		"runIdentifier": wr.ID,
		"jobIdentifier": wrj.JobID,
	})
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
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   1,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.StatusBuilding,
		UserID:       admin.ID,
		Username:     admin.Username,
		RunEvent:     sdk.V2WorkflowRunEvent{},
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
		UserID:        admin.ID,
		Username:      admin.Username,
		ProjectKey:    wr.ProjectKey,
		RunAttempt:    wr.RunAttempt,
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

	uri := api.Router.GetRouteV2("GET", api.getWorkflowRunJobInfosHandler, map[string]string{
		"projectKey":    proj.Key,
		"runIdentifier": wr.ID,
		"jobIdentifier": wrj.JobID,
	})
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
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
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
		RunEvent:     sdk.V2WorkflowRunEvent{},
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

	runJobDB, err := workflow_v2.LoadRunJobByRunIDAndID(ctx, db, wr.ID, wrj.ID)
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
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   1,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.StatusBuilding,
		UserID:       admin.ID,
		Username:     admin.Username,
		RunEvent:     sdk.V2WorkflowRunEvent{},
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
		UserID:        admin.ID,
		Username:      admin.Username,
		ProjectKey:    wr.ProjectKey,
		RunAttempt:    wr.RunAttempt,
		JobID:         sdk.RandomString(10),
		StepsStatus: sdk.JobStepsStatus{
			"step1": {
				Outcome:    sdk.StatusSuccess,
				Conclusion: sdk.StatusSuccess,
			},
		},
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj))

	uriGet := api.Router.GetRouteV2("GET", api.getWorkflowRunJobLogsLinksV2Handler, map[string]string{
		"projectKey":    proj.Key,
		"runIdentifier": wr.ID,
		"jobIdentifier": wrj.JobID,
	})
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
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   1,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.StatusBuilding,
		UserID:       admin.ID,
		Username:     admin.Username,
		RunEvent:     sdk.V2WorkflowRunEvent{},
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
		UserID:        admin.ID,
		Username:      admin.Username,
		ProjectKey:    wr.ProjectKey,
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj))

	uriGet := api.Router.GetRouteV2("GET", api.getWorkflowRunJobsV2Handler, map[string]string{
		"projectKey":    proj.Key,
		"runIdentifier": wr.ID,
	})
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
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
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
		RunEvent:     sdk.V2WorkflowRunEvent{},
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
		UserID:        admin.ID,
		Username:      admin.Username,
		ProjectKey:    wr.ProjectKey,
		Status:        sdk.StatusBuilding,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj))

	uri := api.Router.GetRouteV2("POST", api.postStopWorkflowRunHandler, map[string]string{
		"projectKey":    proj.Key,
		"runIdentifier": wr.ID,
	})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, "POST", uri, nil)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 204, w.Code)

	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.StatusStopped, wrDB.Status)

	rjDB, err := workflow_v2.LoadRunJobByRunIDAndID(context.TODO(), db, wrDB.ID, wrj.ID)
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
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: sdk.RandomString(10),
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   1,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.StatusBuilding,
		UserID:       admin.ID,
		Username:     admin.Username,
		RunEvent:     sdk.V2WorkflowRunEvent{},
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
		UserID:        admin.ID,
		Username:      admin.Username,
		ProjectKey:    wr.ProjectKey,
		Status:        sdk.StatusBuilding,
		RunAttempt:    wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj))

	uri := api.Router.GetRouteV2("POST", api.postStopJobHandler, map[string]string{
		"projectKey":    proj.Key,
		"runIdentifier": wr.ID,
		"jobIdentifier": wrj.JobID,
	})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, "POST", uri, nil)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 204, w.Code)

	// Workflow must be re-enqueued
	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.StatusBuilding, wrDB.Status)

	rjDB, err := workflow_v2.LoadRunJobByRunIDAndID(context.TODO(), db, wr.ID, wrj.ID)
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
		Ref:                 "refs/heads/master",
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
