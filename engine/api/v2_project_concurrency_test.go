package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func Test_crudConcurrencyOnProjectLambdaUserOK(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertLambdaUser(t, db)

	// Insert rbac
	assets.InsertRBAcProject(t, db, "manage", proj.Key, *user1)
	assets.InsertRBAcProject(t, db, "read", proj.Key, *user1)

	// POST request
	concurequest := sdk.ProjectConcurrency{
		Name:             sdk.RandomString(10),
		Description:      "My concurrency rule",
		Order:            sdk.ConcurrencyOrderNewestFirst,
		Pool:             10,
		CancelInProgress: true,
	}

	vars := map[string]string{
		"projectKey": proj.Key,
	}
	uri := api.Router.GetRouteV2("POST", api.postProjectConcurrencyHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, user1, pass, "POST", uri, nil)

	bts, _ := json.Marshal(concurequest)
	// Here, we insert the vcs server as a CDS user (not administrator)
	req.Body = io.NopCloser(bytes.NewReader(bts))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &concurequest))
	require.Equal(t, concurequest.Name, concurequest.Name)

	// Then, Get concurrency
	vars["concurrencyName"] = concurequest.Name
	uriGetOne := api.Router.GetRouteV2("GET", api.getProjectConcurrencyHandler, vars)
	test.NotEmpty(t, uriGetOne)
	reqGetOne := assets.NewAuthentifiedRequest(t, user1, pass, "GET", uriGetOne, nil)
	wGetOne := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wGetOne, reqGetOne)
	require.Equal(t, 200, wGetOne.Code)
	var concurrency sdk.ProjectConcurrency
	require.NoError(t, json.Unmarshal(wGetOne.Body.Bytes(), &concurrency))
	require.Equal(t, concurequest.Name, concurrency.Name)

	// Then PUT
	uriPut := api.Router.GetRouteV2("PUT", api.putProjectConcurrencyHandler, vars)
	test.NotEmpty(t, uriPut)
	reqPut := assets.NewAuthentifiedRequest(t, user1, pass, "PUT", uriPut, nil)
	concurequest.Pool = 1
	bts, _ = json.Marshal(concurequest)
	// Here, we insert the vcs server as a CDS user (not administrator)
	reqPut.Body = io.NopCloser(bytes.NewReader(bts))
	reqPut.Header.Set("Content-Type", "application/json")

	wPut := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wPut, reqPut)
	require.Equal(t, 200, wPut.Code)

	// Then, List
	uriList := api.Router.GetRouteV2("GET", api.getProjectConcurrenciesHandler, vars)
	test.NotEmpty(t, uriList)
	reqList := assets.NewAuthentifiedRequest(t, user1, pass, "GET", uriList, nil)
	wList := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wList, reqList)
	require.Equal(t, 200, wList.Code)

	var concurrencies []sdk.ProjectConcurrency
	require.NoError(t, json.Unmarshal(wList.Body.Bytes(), &concurrencies))
	require.Len(t, concurrencies, 1)
	require.Equal(t, concurequest.Pool, concurrencies[0].Pool)
	require.Equal(t, concurequest.ID, concurrencies[0].ID)

	// Then Delete
	uriDelete := api.Router.GetRouteV2("DELETE", api.deleteProjectConcurrencyHandler, vars)
	test.NotEmpty(t, uriDelete)
	reqDelete := assets.NewAuthentifiedRequest(t, user1, pass, "DELETE", uriDelete, nil)
	w3 := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w3, reqDelete)
	require.Equal(t, 204, w3.Code)

	uriList = api.Router.GetRouteV2("GET", api.getProjectConcurrenciesHandler, vars)
	test.NotEmpty(t, uriList)
	reqList = assets.NewAuthentifiedRequest(t, user1, pass, "GET", uriList, nil)
	wList = httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(wList, reqList)
	require.Equal(t, 200, wList.Code)

	require.NoError(t, json.Unmarshal(wList.Body.Bytes(), &concurrencies))
	require.Len(t, concurrencies, 0)

}

func Test_getProjectConcurrencyRunsHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	user1, pass := assets.InsertAdminUser(t, db)

	pc := sdk.ProjectConcurrency{
		ProjectKey: proj.Key,
		Name:       sdk.RandomString(10),
	}
	require.NoError(t, project.InsertConcurrency(context.TODO(), db, &pc))

	wrSuccess := sdk.V2WorkflowRun{
		ID:           sdk.UUID(),
		ProjectKey:   proj.Key,
		VCSServerID:  sdk.UUID(),
		RepositoryID: sdk.UUID(),
		WorkflowName: "myworkflow",
		Status:       sdk.V2WorkflowRunStatusSuccess,
		RunNumber:    1,
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:  pc.Name,
				Order: sdk.ConcurrencyOrderOldestFirst,
				Pool:  1,
			},
			Scope: sdk.V2RunConcurrencyScopeProject,
		},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wrSuccess))

	wrBlocked := sdk.V2WorkflowRun{
		ID:           sdk.UUID(),
		ProjectKey:   proj.Key,
		VCSServerID:  sdk.UUID(),
		RepositoryID: sdk.UUID(),
		WorkflowName: "myworkflow",
		Status:       sdk.V2WorkflowRunStatusBlocked,
		RunNumber:    1,
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:  pc.Name,
				Order: sdk.ConcurrencyOrderOldestFirst,
				Pool:  1,
			},
			Scope: sdk.V2RunConcurrencyScopeProject,
		},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wrBlocked))

	wrBuilding := sdk.V2WorkflowRun{
		ID:           sdk.UUID(),
		ProjectKey:   proj.Key,
		VCSServerID:  sdk.UUID(),
		RepositoryID: sdk.UUID(),
		WorkflowName: "myworkflow",
		Status:       sdk.V2WorkflowRunStatusBuilding,
		RunNumber:    1,
		Concurrency: &sdk.V2RunConcurrency{
			WorkflowConcurrency: sdk.WorkflowConcurrency{
				Name:  pc.Name,
				Order: sdk.ConcurrencyOrderOldestFirst,
				Pool:  1,
			},
			Scope: sdk.V2RunConcurrencyScopeProject,
		},
	}
	require.NoError(t, workflow_v2.InsertRun(context.TODO(), db, &wrBuilding))

	rjBlocked := sdk.V2WorkflowRunJob{
		JobID:         "myjobBlocked",
		ID:            sdk.UUID(),
		ProjectKey:    proj.Key,
		WorkflowRunID: wrBuilding.ID,
		RunNumber:     wrBuilding.RunNumber,
		Status:        sdk.V2WorkflowRunJobStatusBlocked,
		Initiator:     sdk.V2Initiator{UserID: user1.ID, User: user1.Initiator()},
		Concurrency:   wrBuilding.Concurrency,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &rjBlocked))

	rjSuccess := sdk.V2WorkflowRunJob{
		JobID:         "myjobSuccess",
		ID:            sdk.UUID(),
		ProjectKey:    proj.Key,
		WorkflowRunID: wrBuilding.ID,
		RunNumber:     wrBuilding.RunNumber + 1,
		Status:        sdk.V2WorkflowRunJobStatusSuccess,
		Initiator:     sdk.V2Initiator{UserID: user1.ID, User: user1.Initiator()},
		Concurrency:   wrBuilding.Concurrency,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &rjSuccess))

	rjScheduling := sdk.V2WorkflowRunJob{
		JobID:         "myjobScheduled",
		ID:            sdk.UUID(),
		ProjectKey:    proj.Key,
		WorkflowRunID: wrBuilding.ID,
		RunNumber:     wrBuilding.RunNumber + 2,
		Status:        sdk.V2WorkflowRunJobStatusScheduling,
		Initiator:     sdk.V2Initiator{UserID: user1.ID, User: user1.Initiator()},
		Concurrency:   wrBuilding.Concurrency,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &rjScheduling))

	rjWaiting := sdk.V2WorkflowRunJob{
		JobID:         "myjobWaiting",
		ID:            sdk.UUID(),
		ProjectKey:    proj.Key,
		WorkflowRunID: wrBuilding.ID,
		RunNumber:     wrBuilding.RunNumber + 3,
		Status:        sdk.V2WorkflowRunJobStatusWaiting,
		Initiator:     sdk.V2Initiator{UserID: user1.ID, User: user1.Initiator()},
		Concurrency:   wrBuilding.Concurrency,
	}
	require.NoError(t, workflow_v2.InsertRunJob(context.TODO(), db, &rjWaiting))

	vars := make(map[string]string)
	vars["projectKey"] = proj.Key
	vars["concurrencyName"] = pc.Name
	uriGetOne := api.Router.GetRouteV2("GET", api.getProjectConcurrencyRunsHandler, vars)
	test.NotEmpty(t, uriGetOne)
	reqGetOne := assets.NewAuthentifiedRequest(t, user1, pass, "GET", uriGetOne, nil)
	wGetOne := httptest.NewRecorder()

	api.Router.Mux.ServeHTTP(wGetOne, reqGetOne)
	require.Equal(t, 200, wGetOne.Code)
	var pcrs []sdk.ProjectConcurrencyRunObject
	require.NoError(t, json.Unmarshal(wGetOne.Body.Bytes(), &pcrs))

	require.Equal(t, 5, len(pcrs))
	t.Logf("%+v", pcrs)
	for _, p := range pcrs {
		require.True(t, p.JobName != rjSuccess.JobID)
		require.True(t, p.WorkflowRunID != wrSuccess.ID)
	}
}
