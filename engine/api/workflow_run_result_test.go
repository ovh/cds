package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func Test_getWorkflowRunAndNodeRunResults(t *testing.T) {
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)

	u, pass := assets.InsertAdminUser(t, db)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	w := assets.InsertTestWorkflow(t, db, api.Cache, proj, sdk.RandomString(10))

	wrCreate, err := workflow.CreateRun(api.mustDB(), w, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	assert.NoError(t, err)

	require.NoError(t, api.workflowRunCraft(context.TODO(), wrCreate.ID))

	wrDB, err := workflow.LoadRunByID(context.Background(), db, wrCreate.ID, workflow.LoadRunOptions{})
	require.NoError(t, err)

	artiData := sdk.WorkflowRunResultArtifact{
		WorkflowRunResultArtifactCommon: sdk.WorkflowRunResultArtifactCommon{
			Name: "myarti",
		},
		CDNRefHash: "123",
		MD5:        "123",
		Size:       1,
		Perm:       0777,
	}
	bts, err := json.Marshal(&artiData)
	require.NoError(t, err)
	result := sdk.WorkflowRunResult{
		Type:              sdk.WorkflowRunResultTypeArtifact,
		WorkflowNodeRunID: wrDB.WorkflowNodeRuns[w.WorkflowData.Node.ID][0].ID,
		WorkflowRunJobID:  wrDB.WorkflowNodeRuns[w.WorkflowData.Node.ID][0].Stages[0].RunJobs[0].ID,
		WorkflowRunID:     wrDB.ID,
		DataRaw:           bts,
	}
	api.Cache.SetWithTTL(workflow.GetRunResultKey(wrCreate.ID, sdk.WorkflowRunResultTypeArtifact, artiData.Name), true, 60)
	require.NoError(t, workflow.AddResult(context.TODO(), db.DbMap, api.Cache, wrDB, &result))

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
		"number":           fmt.Sprintf("%d", wrDB.Number),
		"nodeRunID":        fmt.Sprintf("%d", wrDB.WorkflowNodeRuns[wrDB.Workflow.WorkflowData.Node.ID][0].ID),
	}

	uri := router.GetRoute("GET", api.getWorkflowNodeRunResultsHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)

	// Can't work because check has not be done
	assert.Equal(t, 200, rec.Code)

	var results []sdk.WorkflowRunResult
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &results))

	require.Equal(t, 1, len(results))

	art, err := results[0].GetArtifact()
	require.NoError(t, err)
	require.Equal(t, art.Name, artiData.Name)

	//Prepare request for RUN
	varsRun := map[string]string{
		"key":                      proj.Key,
		"permWorkflowNameAdvanced": w.Name,
		"number":                   fmt.Sprintf("%d", wrDB.Number),
	}

	uriRun := router.GetRoute("GET", api.getWorkflowRunResultsHandler, varsRun)
	test.NotEmpty(t, uri)
	reqRun := assets.NewAuthentifiedRequest(t, u, pass, "GET", uriRun, nil)

	//Do the request
	recRun := httptest.NewRecorder()
	router.Mux.ServeHTTP(recRun, reqRun)

	// Can't work because check has not be done
	assert.Equal(t, 200, recRun.Code)

	var resultsRun []sdk.WorkflowRunResult
	require.NoError(t, json.Unmarshal(recRun.Body.Bytes(), &resultsRun))

	require.Equal(t, 1, len(resultsRun))

	artRun, err := results[0].GetArtifact()
	require.NoError(t, err)
	require.Equal(t, artRun.Name, artiData.Name)
}
