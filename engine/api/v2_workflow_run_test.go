package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

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

func TestSearchAllWorkflow(t *testing.T) {
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
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		RunEvent: sdk.V2WorkflowRunEvent{},
		Contexts: sdk.WorkflowRunContext{
			Git: sdk.GitContext{
				Server:     "github",
				Repository: "ovh/cds",
				Ref:        "refs/heads/master",
				Sha:        "123456",
			},
		},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	uri := api.Router.GetRouteV2("GET", api.getWorkflowRunsSearchAllProjectV2Handler, map[string]string{})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, "GET", uri+"?commit=123456&offset=0&limit=1", nil)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	var runs []sdk.V2WorkflowRun
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &runs))
	require.Len(t, runs, 1)
	require.Equal(t, wr.ID, runs[0].ID)

}

func TestRunManualJob_WrongGateReviewer(t *testing.T) {
	api, db, _ := newTestAPI(t)

	lambda, pwd := assets.InsertLambdaUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wName := sdk.RandomString(10)
	assets.InsertRBAcWorkflow(t, db, sdk.WorkflowRoleTrigger, proj.Key, "github/"+repo.Name+"/"+wName, *lambda)

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: wName,
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   0,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.V2WorkflowRunStatusSuccess,
		Initiator: &sdk.V2Initiator{
			UserID: lambda.ID,
			User:   lambda.Initiator(),
		},
		Contexts: sdk.WorkflowRunContext{
			CDS: sdk.CDSContext{
				WorkflowVCSServer:  vcsServer.Name,
				WorkflowRepository: repo.Name,
			},
			Git: sdk.GitContext{
				Server:     vcsServer.Name,
				Repository: repo.Name,
			},
		},
		RunEvent: sdk.V2WorkflowRunEvent{},
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
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job1",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob1))

	wrjJob2 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusSkipped,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job2",
		Job:           wr.WorkflowData.Workflow.Jobs["job2"],
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob2))

	wrjJob3 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job3",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob3))

	wrjJob4 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job4",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob4))

	wrjJob5 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusFail,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job5",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob5))

	wrjJob6 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusFail,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job6",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob6))

	wrjJob7 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job7",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
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

	uri := api.Router.GetRouteV2(http.MethodPost, api.postRunJobHandler, map[string]string{
		"projectKey":    proj.Key,
		"workflowRunID": wr.ID,
		"jobIdentifier": "job2",
	})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, lambda, pwd, http.MethodPost, uri, map[string]interface{}{
		"approve": true,
	})
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 403, w.Code)

	body := w.Body.String()
	require.Contains(t, body, "Conditions are not satisfied")
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
		Status:       sdk.V2WorkflowRunStatusSuccess,
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		RunEvent: sdk.V2WorkflowRunEvent{},
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
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job1",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob1))

	wrjJob2 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusSkipped,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job2",
		Job:           wr.WorkflowData.Workflow.Jobs["job2"],
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob2))

	wrjJob3 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job3",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob3))

	wrjJob4 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job4",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob4))

	wrjJob5 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusFail,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job5",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob5))

	wrjJob6 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusFail,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job6",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob6))

	wrjJob7 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job7",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
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

	uri := api.Router.GetRouteV2(http.MethodPost, api.postRunJobHandler, map[string]string{
		"projectKey":    proj.Key,
		"workflowRunID": wr.ID,
		"jobIdentifier": "job2",
	})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, http.MethodPost, uri, map[string]interface{}{
		"approve": false,
	})
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 403, w.Code)
}

func TestRunManualSkippedJob(t *testing.T) {
	api, db, _ := newTestAPI(t)

	_, err := db.Exec("delete from region where name = 'build-test'")
	require.NoError(t, err)

	reg := sdk.Region{Name: "build-test"}
	require.NoError(t, region.Insert(context.TODO(), db, &reg))

	// admin, pwd := assets.InsertAdminUser(t, db)
	lambda, pwd := assets.InsertLambdaUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wName := sdk.RandomString(10)
	assets.InsertRBAcWorkflow(t, db, sdk.WorkflowRoleTrigger, proj.Key, "github/"+repo.Name+"/"+wName, *lambda)

	wr := sdk.V2WorkflowRun{
		ProjectKey:   proj.Key,
		VCSServerID:  vcsServer.ID,
		VCSServer:    vcsServer.Name,
		RepositoryID: repo.ID,
		Repository:   repo.Name,
		WorkflowName: wName,
		WorkflowSha:  "123",
		WorkflowRef:  "master",
		RunAttempt:   0,
		RunNumber:    1,
		Started:      time.Now(),
		LastModified: time.Now(),
		Status:       sdk.V2WorkflowRunStatusSuccess,
		Initiator: &sdk.V2Initiator{
			UserID: lambda.ID,
			User:   lambda.Initiator(),
		},
		Contexts: sdk.WorkflowRunContext{
			CDS: sdk.CDSContext{
				WorkflowVCSServer:  vcsServer.Name,
				WorkflowRepository: repo.Name,
			},
			Git: sdk.GitContext{
				Server:     vcsServer.Name,
				Repository: repo.Name,
			},
		},
		RunEvent: sdk.V2WorkflowRunEvent{},
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
					Steps: []sdk.ActionStep{{
						Run: "echo hello",
					}},
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
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job1",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob1))

	wrjJob2 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusSkipped,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job2",
		Job:           wr.WorkflowData.Workflow.Jobs["job2"],
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob2))

	wrjJob3 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job3",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob3))

	wrjJob4 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job4",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob4))

	wrjJob5 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusFail,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job5",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob5))

	wrjJob6 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusFail,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job6",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob6))

	wrjJob7 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job7",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
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

	uri := api.Router.GetRouteV2(http.MethodPost, api.postRunJobHandler, map[string]string{
		"projectKey":    proj.Key,
		"workflowRunID": wr.ID,
		"jobIdentifier": "job2",
	})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, lambda, pwd, http.MethodPost, uri, map[string]interface{}{
		"approve": true,
	})
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	// trigger jobs
	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID:         lambda.ID,
			User:           lambda.Initiator(),
			IsAdminWithMFA: true,
		},
	}))

	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.V2WorkflowRunStatusBuilding, wrDB.Status)
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
	require.Equal(t, 2, len(rJob2.GateInputs))
	v, has := rJob2.GateInputs["approve"]
	require.True(t, has)
	require.Equal(t, "true", fmt.Sprintf("%v", v))
	v, has = rJob2.GateInputs["manual"]
	require.True(t, has)
	require.Equal(t, "true", fmt.Sprintf("%v", v))
	require.True(t, rJob2.Initiator.IsAdminWithMFA)
}

func TestRunManualSuccessJob(t *testing.T) {
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
		Status:       sdk.V2WorkflowRunStatusSuccess,
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		RunEvent: sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	wrjJob1 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job1",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob1))

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

	uri := api.Router.GetRouteV2(http.MethodPost, api.postRunJobHandler, map[string]string{
		"projectKey":    proj.Key,
		"workflowRunID": wr.ID,
		"jobIdentifier": "job1",
	})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, http.MethodPost, uri, map[string]interface{}{
		"approve": true,
	})
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	// trigger jobs
	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID:         admin.ID,
			User:           admin.Initiator(),
			IsAdminWithMFA: true,
		},
	}))

	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.V2WorkflowRunStatusBuilding, wrDB.Status)
	require.Equal(t, int64(2), wrDB.RunAttempt)

	runjobs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wrDB.ID, wrDB.RunAttempt)
	require.NoError(t, err)
	require.Equal(t, 1, len(runjobs))

	mapJob := make(map[string]sdk.V2WorkflowRunJob)
	for _, rj := range runjobs {
		mapJob[rj.JobID] = rj
	}

	rJob1, has := mapJob["job1"]
	require.True(t, has)
	require.Equal(t, int64(2), rJob1.RunAttempt)
}

func TestRunManualFailedJob(t *testing.T) {
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
		Status:       sdk.V2WorkflowRunStatusSuccess,
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		RunEvent: sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	wrjJob1 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusFail,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job1",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob1))

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

	uri := api.Router.GetRouteV2(http.MethodPost, api.postRunJobHandler, map[string]string{
		"projectKey":    proj.Key,
		"workflowRunID": wr.ID,
		"jobIdentifier": "job1",
	})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, http.MethodPost, uri, map[string]interface{}{
		"approve": true,
	})
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	// trigger jobs
	require.NoError(t, api.workflowRunV2Trigger(context.TODO(), sdk.V2WorkflowRunEnqueue{
		RunID: wr.ID,
		Initiator: sdk.V2Initiator{
			UserID:         admin.ID,
			User:           admin.Initiator(),
			IsAdminWithMFA: true,
		},
	}))

	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.V2WorkflowRunStatusBuilding, wrDB.Status)
	require.Equal(t, int64(2), wrDB.RunAttempt)

	runjobs, err := workflow_v2.LoadRunJobsByRunID(context.TODO(), db, wrDB.ID, wrDB.RunAttempt)
	require.NoError(t, err)
	require.Equal(t, 1, len(runjobs))

	mapJob := make(map[string]sdk.V2WorkflowRunJob)
	for _, rj := range runjobs {
		mapJob[rj.JobID] = rj
	}

	rJob1, has := mapJob["job1"]
	require.True(t, has)
	require.Equal(t, int64(2), rJob1.RunAttempt)
}

func TestRunManualSkippedJobWithoutGate(t *testing.T) {
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
		Status:       sdk.V2WorkflowRunStatusSuccess,
		RunEvent:     sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {},
			},
		}},
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	wrjJob1 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusSkipped,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job1",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob1))

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

	uri := api.Router.GetRouteV2(http.MethodPost, api.postRunJobHandler, map[string]string{
		"projectKey":    proj.Key,
		"workflowRunID": wr.ID,
		"jobIdentifier": "job1",
	})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, http.MethodPost, uri, map[string]interface{}{
		"approve": true,
	})
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 403, w.Code)
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
		Status:       sdk.V2WorkflowRunStatusFail,
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		RunEvent: sdk.V2WorkflowRunEvent{},
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
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job1",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob1))

	wrjJob2 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusFail,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job2",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob2))

	wrjJob3 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job3",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob3))

	wrjJob4 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job4",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob4))

	wrjJob5 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusFail,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job5",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob5))

	wrjJob6 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusFail,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job6",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob6))

	wrjJob7 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job7",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
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

	uri := api.Router.GetRouteV2(http.MethodPost, api.postRestartWorkflowRunHandler, map[string]string{
		"projectKey":    proj.Key,
		"workflowRunID": wr.ID,
		"runNumber":     strconv.FormatInt(wr.RunNumber, 10),
	})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, http.MethodPost, uri, nil)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.V2WorkflowRunStatusBuilding, wrDB.Status)
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

func TestPostRestartWorkflowRun_BuildingRun(t *testing.T) {
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
		Status:       sdk.V2WorkflowRunStatusBuilding,
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		RunEvent: sdk.V2WorkflowRunEvent{},
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

	uri := api.Router.GetRouteV2(http.MethodPost, api.postRestartWorkflowRunHandler, map[string]string{
		"projectKey":    proj.Key,
		"workflowRunID": wr.ID,
	})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, http.MethodPost, uri, nil)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 400, w.Code)
}

func TestPostRestartWorkflowRun_NoFailingJob(t *testing.T) {
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
		Status:       sdk.V2WorkflowRunStatusSuccess,
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		RunEvent: sdk.V2WorkflowRunEvent{},
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

	uri := api.Router.GetRouteV2(http.MethodPost, api.postRestartWorkflowRunHandler, map[string]string{
		"projectKey":    proj.Key,
		"workflowRunID": wr.ID,
	})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, http.MethodPost, uri, nil)
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
		Status:       sdk.V2WorkflowRunStatusBuilding,
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		RunEvent: sdk.V2WorkflowRunEvent{},
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
		"workflowRunID": wr.ID,
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
		Status:       sdk.V2WorkflowRunStatusBuilding,
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		RunEvent: sdk.V2WorkflowRunEvent{},
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
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		ProjectKey: wr.ProjectKey,
		RunAttempt: wr.RunAttempt,
		JobID:      sdk.RandomString(10),
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj))

	uri := api.Router.GetRouteV2("GET", api.getWorkflowRunJobHandler, map[string]string{
		"projectKey":    proj.Key,
		"workflowRunID": wr.ID,
		"jobRunID":      wrj.ID,
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
		Status:       sdk.V2WorkflowRunStatusBuilding,
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		RunEvent: sdk.V2WorkflowRunEvent{},
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
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		ProjectKey: wr.ProjectKey,
		RunAttempt: wr.RunAttempt,
		JobID:      sdk.RandomString(10),
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
		"workflowRunID": wr.ID,
		"jobRunID":      wrj.ID,
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
		Status:       sdk.V2WorkflowRunStatusBuilding,
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		RunEvent: sdk.V2WorkflowRunEvent{},
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
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		ProjectKey:   wr.ProjectKey,
		JobID:        sdk.RandomString(10),
		Region:       reg.Name,
		HatcheryName: hatch.Name,
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
	require.Equal(t, sdk.V2WorkflowRunJobStatusSuccess, runJobDB.StepsStatus["job1"].Conclusion)
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
		Status:       sdk.V2WorkflowRunStatusBuilding,
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		RunEvent: sdk.V2WorkflowRunEvent{},
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
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		ProjectKey: wr.ProjectKey,
		RunAttempt: wr.RunAttempt,
		JobID:      sdk.RandomString(10),
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
		"workflowRunID": wr.ID,
		"jobRunID":      wrj.ID,
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
		Status:       sdk.V2WorkflowRunStatusBuilding,
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		RunEvent: sdk.V2WorkflowRunEvent{},
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
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		ProjectKey: wr.ProjectKey,
		RunAttempt: wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj))

	uriGet := api.Router.GetRouteV2("GET", api.getWorkflowRunJobsV2Handler, map[string]string{
		"projectKey":    proj.Key,
		"workflowRunID": wr.ID,
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
		Status:       sdk.V2WorkflowRunStatusBuilding,
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		RunEvent: sdk.V2WorkflowRunEvent{},
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
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		ProjectKey: wr.ProjectKey,
		RunNumber:  wr.RunNumber,
		RunAttempt: wr.RunAttempt,
		Status:     sdk.V2WorkflowRunJobStatusBuilding,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj))

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

	servicesClients.EXPECT().DoJSONRequest(gomock.Any(), "POST", "/v2/workflow/outgoing", gomock.Any(), gomock.Any())

	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	go api.V2WorkflowRunEngineDequeue(ctx)

	uri := api.Router.GetRouteV2("POST", api.postStopWorkflowRunHandler, map[string]string{
		"projectKey":    proj.Key,
		"workflowRunID": wr.ID,
	})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, "POST", uri, nil)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 204, w.Code)

	enqueueRequest := sdk.V2WorkflowRunEnqueue{
		RunID:     wr.ID,
		Initiator: *wr.Initiator,
		Status:    sdk.V2WorkflowRunStatusStopped,
	}
	require.NoError(t, api.workflowRunV2Trigger(context.Background(), enqueueRequest))

	nbRetry := 10
	for i := 0; i < nbRetry; i++ {
		wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
		require.NoError(t, err)
		if wrDB.Status != sdk.V2WorkflowRunStatusStopped {
			if i == nbRetry-1 {
				t.Fail()
			}
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}
	cancel()

	rjDB, err := workflow_v2.LoadRunJobByRunIDAndID(context.TODO(), db, wr.ID, wrj.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.V2WorkflowRunJobStatusStopped, rjDB.Status)
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
		Status:       sdk.V2WorkflowRunStatusBuilding,
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		RunEvent: sdk.V2WorkflowRunEvent{},
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
		Initiator: sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		ProjectKey: wr.ProjectKey,
		Status:     sdk.V2WorkflowRunJobStatusBuilding,
		RunAttempt: wr.RunAttempt,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrj))

	uri := api.Router.GetRouteV2("POST", api.postStopJobHandler, map[string]string{
		"projectKey":    proj.Key,
		"workflowRunID": wr.ID,
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
	require.Equal(t, sdk.V2WorkflowRunStatusBuilding, wrDB.Status)

	rjDB, err := workflow_v2.LoadRunJobByRunIDAndID(context.TODO(), db, wr.ID, wrj.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.V2WorkflowRunJobStatusStopped, rjDB.Status)
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
		Commit:              "HEAD",
		ProjectRepositoryID: repo.ID,
		Head:                true,
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

	// Mock Hook
	s, _ := assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	s2, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		_ = services.Delete(db, s2)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/v2/workflow/manual", gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/"+repo.Name+"/branches/?branch=&default=true", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				b := &sdk.VCSBranch{
					Default:      true,
					DisplayID:    "master",
					ID:           "refs/heads/master",
					LatestCommit: "HEAD",
				}
				*(out.(*sdk.VCSBranch)) = *b
				return nil, 200, nil
			},
		).Times(1)

	vars := map[string]string{
		"projectKey":           proj.Key,
		"vcsIdentifier":        vcsServer.ID,
		"repositoryIdentifier": repo.ID,
		"workflow":             e.Name,
	}

	uri := api.Router.GetRouteV2("POST", api.postWorkflowRunV2Handler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, "POST", uri, map[string]interface{}{
		"branch": "main",
		"sha":    "123456",
	})
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 201, w.Code)
}

func TestRunManualJob_GateArrayInput(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, pwd := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:         proj.Key,
		VCSServerID:        vcsServer.ID,
		VCSServer:          vcsServer.Name,
		RepositoryID:       repo.ID,
		Repository:         repo.Name,
		WorkflowName:       sdk.RandomString(10),
		WorkflowSha:        "123",
		WorkflowRef:        "master",
		RunAttempt:         0,
		RunNumber:          1,
		Started:            time.Now(),
		LastModified:       time.Now(),
		Status:             sdk.V2WorkflowRunStatusSuccess,
		DeprecatedUserID:   admin.ID,
		DeprecatedUsername: admin.Username,
		Initiator:          &sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator()},
		RunEvent:           sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Gates: map[string]sdk.V2JobGate{
				"preprod": {
					If: "${{gate.approve == 'foo'}}",
					Inputs: map[string]sdk.V2JobGateInput{
						"approve": {
							Options: &sdk.V2JobGateOptions{
								Values: []interface{}{"foo", "boo"},
							},
						},
					},
				},
			},
			Jobs: map[string]sdk.V2Job{
				"job1": {
					Gate: "preprod",
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	wrjJob1 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusSkipped,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job1",
		RunAttempt:    wr.RunAttempt,
		Job:           wr.WorkflowData.Workflow.Jobs["job1"],
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob1))

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

	uri := api.Router.GetRouteV2(http.MethodPost, api.postRunJobHandler, map[string]string{
		"projectKey":    proj.Key,
		"workflowRunID": wr.ID,
		"jobIdentifier": "job1",
	})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, http.MethodPost, uri, map[string]interface{}{
		"approve": "foo",
	})
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
}

func TestRunManualJob_GateArrayWrongValue(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, pwd := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:         proj.Key,
		VCSServerID:        vcsServer.ID,
		VCSServer:          vcsServer.Name,
		RepositoryID:       repo.ID,
		Repository:         repo.Name,
		WorkflowName:       sdk.RandomString(10),
		WorkflowSha:        "123",
		WorkflowRef:        "master",
		RunAttempt:         0,
		RunNumber:          1,
		Started:            time.Now(),
		LastModified:       time.Now(),
		Status:             sdk.V2WorkflowRunStatusSuccess,
		DeprecatedUserID:   admin.ID,
		DeprecatedUsername: admin.Username,
		Initiator:          &sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator()},
		RunEvent:           sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Gates: map[string]sdk.V2JobGate{
				"preprod": {
					If: "${{gate.approve == 'foo'}}",
					Inputs: map[string]sdk.V2JobGateInput{
						"approve": {
							Options: &sdk.V2JobGateOptions{
								Values: []interface{}{"foo", "boo"},
							},
						},
					},
				},
			},
			Jobs: map[string]sdk.V2Job{
				"job1": {
					Gate: "preprod",
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	wrjJob1 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusSkipped,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job1",
		RunAttempt:    wr.RunAttempt,
		Job:           wr.WorkflowData.Workflow.Jobs["job1"],
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob1))

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

	uri := api.Router.GetRouteV2(http.MethodPost, api.postRunJobHandler, map[string]string{
		"projectKey":    proj.Key,
		"workflowRunID": wr.ID,
		"jobIdentifier": "job1",
	})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, http.MethodPost, uri, map[string]interface{}{
		"approve": "toto",
	})
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 400, w.Code)
}

func TestRunManualJob_GateArrayMultipleValues(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, pwd := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:         proj.Key,
		VCSServerID:        vcsServer.ID,
		VCSServer:          vcsServer.Name,
		RepositoryID:       repo.ID,
		Repository:         repo.Name,
		WorkflowName:       sdk.RandomString(10),
		WorkflowSha:        "123",
		WorkflowRef:        "master",
		RunAttempt:         0,
		RunNumber:          1,
		Started:            time.Now(),
		LastModified:       time.Now(),
		Status:             sdk.V2WorkflowRunStatusSuccess,
		DeprecatedUserID:   admin.ID,
		DeprecatedUsername: admin.Username,
		Initiator:          &sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator()},
		RunEvent:           sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Gates: map[string]sdk.V2JobGate{
				"preprod": {
					If: "${{gate.approve[0] == 'foo' && gate.approve[1] == 'bar'}}",
					Inputs: map[string]sdk.V2JobGateInput{
						"approve": {
							Options: &sdk.V2JobGateOptions{
								Multiple: true,
								Values:   []interface{}{"foo", "bar"},
							},
						},
					},
				},
			},
			Jobs: map[string]sdk.V2Job{
				"job1": {
					Gate: "preprod",
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	wrjJob1 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusSkipped,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job1",
		RunAttempt:    wr.RunAttempt,
		Job:           wr.WorkflowData.Workflow.Jobs["job1"],
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob1))

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

	uri := api.Router.GetRouteV2(http.MethodPost, api.postRunJobHandler, map[string]string{
		"projectKey":    proj.Key,
		"workflowRunID": wr.ID,
		"jobIdentifier": "job1",
	})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, http.MethodPost, uri, map[string]interface{}{
		"approve": []interface{}{"foo", "bar"},
	})
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
}

func TestRunManualJob_GateArrayMultipleValuesButWrong(t *testing.T) {
	api, db, _ := newTestAPI(t)

	admin, pwd := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, sdk.RandomString(10))

	wr := sdk.V2WorkflowRun{
		ProjectKey:         proj.Key,
		VCSServerID:        vcsServer.ID,
		VCSServer:          vcsServer.Name,
		RepositoryID:       repo.ID,
		Repository:         repo.Name,
		WorkflowName:       sdk.RandomString(10),
		WorkflowSha:        "123",
		WorkflowRef:        "master",
		RunAttempt:         0,
		RunNumber:          1,
		Started:            time.Now(),
		LastModified:       time.Now(),
		Status:             sdk.V2WorkflowRunStatusSuccess,
		DeprecatedUserID:   admin.ID,
		DeprecatedUsername: admin.Username,
		Initiator:          &sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator()},
		RunEvent:           sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Gates: map[string]sdk.V2JobGate{
				"preprod": {
					If: "${{gate.approve[0] == 'foo' && gate.approve[1] == 'bar'}}",
					Inputs: map[string]sdk.V2JobGateInput{
						"approve": {
							Options: &sdk.V2JobGateOptions{
								Multiple: true,
								Values:   []interface{}{"foo", "bar"},
							},
						},
					},
				},
			},
			Jobs: map[string]sdk.V2Job{
				"job1": {
					Gate: "preprod",
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	wrjJob1 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusSkipped,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job1",
		RunAttempt:    wr.RunAttempt,
		Job:           wr.WorkflowData.Workflow.Jobs["job1"],
		Initiator:     sdk.V2Initiator{UserID: admin.ID, User: admin.Initiator()},
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob1))

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

	uri := api.Router.GetRouteV2(http.MethodPost, api.postRunJobHandler, map[string]string{
		"projectKey":    proj.Key,
		"workflowRunID": wr.ID,
		"jobIdentifier": "job1",
	})
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, admin, pwd, http.MethodPost, uri, map[string]interface{}{
		"approve": []interface{}{"foo", "bar", "boo"},
	})
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 400, w.Code)
}

func TestPostWorkflowRunHandler_JobInputs(t *testing.T) {
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
		Commit:              "HEAD",
		ProjectRepositoryID: repo.ID,
		Head:                true,
		Data: `name: MyFirstWorkflow
gates:
  mygate:
    if: gate.zone != ''
    inputs:
      zone:
        type: string
jobs:
  jobNoGate:
    name: This is my first job
    worker_model: buildpack-deps-buster
    region: default
    steps:
      - run: |-
          echo "It is my first step"
  jobWithGate:
    gate: mygate
    name: This is my first job
    worker_model: buildpack-deps-buster
    region: default
    steps:
      - run: |-
          echo "It is my first step"
  jobWithNeeds:
    gate: mygate
    needs: [jobWithGate]
    name: This is my first job
    worker_model: buildpack-deps-buster
    region: default
    steps:
      - run: |-
          echo "It is my first step"`,
	}
	require.NoError(t, entity.Insert(context.TODO(), db, &e))

	// Mock Hook
	s, _ := assets.InsertService(t, db, t.Name()+"_HOOKS", sdk.TypeHooks)
	s2, _ := assets.InsertService(t, db, t.Name()+"_VCS", sdk.TypeVCS)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		_ = services.Delete(db, s)
		_ = services.Delete(db, s2)
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/v2/workflow/manual", gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "GET", "/vcs/github/repos/"+repo.Name+"/branches/?branch=&default=true", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, method, path string, in interface{}, out interface{}, _ interface{}) (http.Header, int, error) {
				b := &sdk.VCSBranch{
					Default:      true,
					DisplayID:    "master",
					ID:           "refs/heads/master",
					LatestCommit: "HEAD",
				}
				*(out.(*sdk.VCSBranch)) = *b
				return nil, 200, nil
			},
		).AnyTimes()

	vars := map[string]string{
		"projectKey":           proj.Key,
		"vcsIdentifier":        vcsServer.ID,
		"repositoryIdentifier": repo.ID,
		"workflow":             e.Name,
	}

	payload := sdk.V2WorkflowRunManualRequest{
		Branch: "main",
		Sha:    "123456",
	}

	tests := []struct {
		name   string
		inputs map[string]sdk.V2WorkflowRunManualRequestJobInput
		err    string
	}{
		{
			name: "Job with no gate",
			inputs: map[string]sdk.V2WorkflowRunManualRequestJobInput{
				"jobNoGate": {
					"zone": "zone1",
				},
			},
			err: `unable to send input to a job without a gate \"jobNoGate\"`,
		},
		{
			name: "Non root job",
			inputs: map[string]sdk.V2WorkflowRunManualRequestJobInput{
				"jobWithNeeds": {
					"zone": "zone1",
				},
			},
			err: `unable to send input to a non root job \"jobWithNeeds\"`,
		},
		{
			name: "Non existing job",
			inputs: map[string]sdk.V2WorkflowRunManualRequestJobInput{
				"unknown": {
					"zone": "zone1",
				},
			},
			err: `job \"unknown\" not found in workflow`,
		},
		{
			name: "wrong inputs",
			inputs: map[string]sdk.V2WorkflowRunManualRequestJobInput{
				"jobWithGate": {
					"zaune": "zone1",
				},
			},
			err: `input \"zaune\" not found in gate \"mygate\" of job \"jobWithGate\"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload.JobInputs = tt.inputs

			uri := api.Router.GetRouteV2("POST", api.postWorkflowRunV2Handler, vars)
			test.NotEmpty(t, uri)
			payload.JobInputs = tt.inputs
			req := assets.NewAuthentifiedRequest(t, admin, pwd, "POST", uri, payload)
			w := httptest.NewRecorder()
			api.Router.Mux.ServeHTTP(w, req)
			require.Equal(t, 400, w.Code)
			require.Contains(t, w.Body.String(), tt.err)
		})
	}
}

func TestPostStartJobWorkflowRunHandler_RunningWorkflow(t *testing.T) {
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
		Status:       sdk.V2WorkflowRunStatusBuilding,
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		RunEvent: sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	uri := api.Router.GetRouteV2(http.MethodPost, api.postStartJobWorkflowRunHandler, map[string]string{
		"projectKey":    proj.Key,
		"workflowRunID": wr.ID,
	})
	test.NotEmpty(t, uri)
	payload := sdk.V2WorkflowRunJobsRequest{
		JobInputs: map[string]sdk.V2WorkflowRunManualRequestJobInput{
			"job1": {},
		},
	}
	req := assets.NewAuthentifiedRequest(t, admin, pwd, http.MethodPost, uri, payload)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 400, w.Code)
	require.Contains(t, w.Body.String(), "unable to start jobs on a running workflow")
}

func TestPostStartJobWorkflowRunHandler_NoJobProvided(t *testing.T) {
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
		Status:       sdk.V2WorkflowRunStatusFail,
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		RunEvent: sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Jobs: map[string]sdk.V2Job{
				"job1": {},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	uri := api.Router.GetRouteV2(http.MethodPost, api.postStartJobWorkflowRunHandler, map[string]string{
		"projectKey":    proj.Key,
		"workflowRunID": wr.ID,
	})
	test.NotEmpty(t, uri)

	payload := sdk.V2WorkflowRunJobsRequest{
		JobInputs: map[string]sdk.V2WorkflowRunManualRequestJobInput{},
	}
	req := assets.NewAuthentifiedRequest(t, admin, pwd, http.MethodPost, uri, payload)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 400, w.Code)
	require.Contains(t, w.Body.String(), "no job provided")
}

func TestPostStartJobWorkflowRunHandler_NoTerminatedRunJobs(t *testing.T) {
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
		Status:       sdk.V2WorkflowRunStatusSuccess,
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		RunEvent: sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Gates: map[string]sdk.V2JobGate{
				"mygate": {
					Inputs: map[string]sdk.V2JobGateInput{
						"environment": {
							Type: "string",
						},
					},
				},
			},
			Jobs: map[string]sdk.V2Job{
				"job1": {
					Gate: "mygate",
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	wrjJob1 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusBuilding,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job1",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob1))

	uri := api.Router.GetRouteV2(http.MethodPost, api.postStartJobWorkflowRunHandler, map[string]string{
		"projectKey":    proj.Key,
		"workflowRunID": wr.ID,
	})
	test.NotEmpty(t, uri)
	payload := sdk.V2WorkflowRunJobsRequest{
		JobInputs: map[string]sdk.V2WorkflowRunManualRequestJobInput{
			"job1": map[string]interface{}{
				"environment": "prod",
			},
		},
	}
	req := assets.NewAuthentifiedRequest(t, admin, pwd, http.MethodPost, uri, payload)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 400, w.Code)
	require.Contains(t, w.Body.String(), "unable to find job that can be restarted")
}

func TestPostStartJobWorkflowRunHandler_Success(t *testing.T) {
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
		Status:       sdk.V2WorkflowRunStatusFail,
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		RunEvent: sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Gates: map[string]sdk.V2JobGate{
				"mygate": {
					Inputs: map[string]sdk.V2JobGateInput{
						"environment": {
							Type: "string",
						},
					},
				},
			},
			Jobs: map[string]sdk.V2Job{
				"job1": {
					Gate: "mygate",
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	wrjJob1 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusSkipped,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job1",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob1))

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

	uri := api.Router.GetRouteV2(http.MethodPost, api.postStartJobWorkflowRunHandler, map[string]string{
		"projectKey":    proj.Key,
		"workflowRunID": wr.ID,
	})
	test.NotEmpty(t, uri)
	payload := sdk.V2WorkflowRunJobsRequest{
		JobInputs: map[string]sdk.V2WorkflowRunManualRequestJobInput{
			"job1": map[string]interface{}{
				"environment": "prod",
			},
		},
	}
	req := assets.NewAuthentifiedRequest(t, admin, pwd, http.MethodPost, uri, payload)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)

	require.Equal(t, int64(2), wrDB.RunAttempt)
	require.Len(t, wrDB.RunJobEvent, 1)
	require.Equal(t, "job1", wrDB.RunJobEvent[0].JobID)
	require.Equal(t, admin.ID, wrDB.RunJobEvent[0].UserID)

	runInfos, err := workflow_v2.LoadRunInfosByRunID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(runInfos), 1)

	var found bool
	for _, info := range runInfos {
		//fmt.Printf("Info: %+v\n", info)
		if info.Message == admin.Fullname+" starts jobs: [job1]" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestPostStartJobWorkflowRunHandler_WithGate(t *testing.T) {
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
		Status:       sdk.V2WorkflowRunStatusFail,
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		RunEvent: sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Gates: map[string]sdk.V2JobGate{
				"mygate": {
					Inputs: map[string]sdk.V2JobGateInput{
						"environment": {
							Type: "string",
						},
					},
				},
			},
			Jobs: map[string]sdk.V2Job{
				"job1": {
					Gate: "mygate",
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	wrjJob1 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusSkipped,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job1",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob1))

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

	uri := api.Router.GetRouteV2(http.MethodPost, api.postStartJobWorkflowRunHandler, map[string]string{
		"projectKey":    proj.Key,
		"workflowRunID": wr.ID,
	})
	test.NotEmpty(t, uri)
	payload := sdk.V2WorkflowRunJobsRequest{
		JobInputs: map[string]sdk.V2WorkflowRunManualRequestJobInput{
			"job1": map[string]interface{}{
				"environment": "production",
			},
		},
	}
	req := assets.NewAuthentifiedRequest(t, admin, pwd, http.MethodPost, uri, payload)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Len(t, wrDB.RunJobEvent, 1)
	require.Equal(t, "job1", wrDB.RunJobEvent[0].JobID)
	require.NotNil(t, wrDB.RunJobEvent[0].Inputs)
	require.Equal(t, "production", wrDB.RunJobEvent[0].Inputs["environment"])
	require.Equal(t, true, wrDB.RunJobEvent[0].Inputs["manual"])
}

func TestPostStartJobWorkflowRunHandler_InvalidGateInput(t *testing.T) {
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
		Status:       sdk.V2WorkflowRunStatusFail,
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		RunEvent: sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Gates: map[string]sdk.V2JobGate{
				"mygate": {
					Inputs: map[string]sdk.V2JobGateInput{
						"environment": {
							Type: "string",
						},
					},
				},
			},
			Jobs: map[string]sdk.V2Job{
				"job1": {
					Gate: "mygate",
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	wrjJob1 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusSkipped,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job1",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob1))

	uri := api.Router.GetRouteV2(http.MethodPost, api.postStartJobWorkflowRunHandler, map[string]string{
		"projectKey":    proj.Key,
		"workflowRunID": wr.ID,
	})
	test.NotEmpty(t, uri)

	payload := sdk.V2WorkflowRunJobsRequest{
		JobInputs: map[string]sdk.V2WorkflowRunManualRequestJobInput{
			"job1": map[string]interface{}{
				"wrong_input": "production",
			},
		},
	}

	req := assets.NewAuthentifiedRequest(t, admin, pwd, http.MethodPost, uri, payload)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 400, w.Code)
	t.Logf("Response: %s", w.Body.String())
	require.Contains(t, w.Body.String(), `\"wrong_input\" not found in gate \"mygate\" of job \"job1\"`)
}

func TestPostStartJobWorkflowRunHandler_MultipleJobs(t *testing.T) {
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
		Status:       sdk.V2WorkflowRunStatusFail,
		Initiator: &sdk.V2Initiator{
			UserID: admin.ID,
			User:   admin.Initiator(),
		},
		RunEvent: sdk.V2WorkflowRunEvent{},
		WorkflowData: sdk.V2WorkflowRunData{Workflow: sdk.V2Workflow{
			Gates: map[string]sdk.V2JobGate{
				"gate1": {
					Inputs: map[string]sdk.V2JobGateInput{
						"environment": {
							Type: "string",
						},
					},
				},
				"gate2": {
					Inputs: map[string]sdk.V2JobGateInput{
						"region": {
							Type: "string",
						},
					},
				},
			},
			Jobs: map[string]sdk.V2Job{
				"job1": {
					Gate: "gate1",
				},
				"job2": {
					Gate: "gate2",
				},
			},
		}},
	}
	require.NoError(t, workflow_v2.InsertRun(context.Background(), db, &wr))

	wrjJob1 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusSkipped,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job1",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob1))

	wrjJob2 := sdk.V2WorkflowRunJob{
		Status:        sdk.V2WorkflowRunJobStatusSkipped,
		WorkflowRunID: wr.ID,
		ProjectKey:    proj.Key,
		JobID:         "job2",
		RunAttempt:    wr.RunAttempt,
		Initiator:     *wr.Initiator,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &wrjJob2))

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

	uri := api.Router.GetRouteV2(http.MethodPost, api.postStartJobWorkflowRunHandler, map[string]string{
		"projectKey":    proj.Key,
		"workflowRunID": wr.ID,
	})
	test.NotEmpty(t, uri)
	payload := sdk.V2WorkflowRunJobsRequest{
		JobInputs: map[string]sdk.V2WorkflowRunManualRequestJobInput{
			"job1": map[string]interface{}{
				"environment": "prod",
			},
			"job2": map[string]interface{}{
				"region": "eu-west",
			},
		},
	}
	req := assets.NewAuthentifiedRequest(t, admin, pwd, http.MethodPost, uri, payload)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	wrDB, err := workflow_v2.LoadRunByID(context.TODO(), db, wr.ID)
	require.NoError(t, err)
	require.Len(t, wrDB.RunJobEvent, 2)

	var findJob1, findjob2 bool
	for _, event := range wrDB.RunJobEvent {
		if event.JobID == "job1" && event.Inputs["environment"] == "prod" {
			findJob1 = true
		}
		if event.JobID == "job2" && event.Inputs["region"] == "eu-west" {
			findjob2 = true
		}
	}
	require.True(t, findJob1)
	require.True(t, findjob2)
}
