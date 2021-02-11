package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/featureflipping"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
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

	wrDB, err := workflow.LoadRunByID(db, wrCreate.ID, workflow.LoadRunOptions{})
	require.NoError(t, err)

	artiData := sdk.WorkflowRunResultArtifact{
		Name:       "myarti",
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
	api.Cache.SetWithTTL(workflow.GetArtifactResultKey(wrCreate.ID, artiData.Name), true, 60)
	require.NoError(t, workflow.AddResult(db.DbMap, api.Cache, &result))

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
		"key":              proj.Key,
		"permWorkflowName": w.Name,
		"number":           fmt.Sprintf("%d", wrDB.Number),
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

func Test_workflowRunResultsAdd(t *testing.T) {
	featureflipping.Init(gorpmapping.Mapper)
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)

	feat := &sdk.Feature{
		Name: sdk.FeatureCDNArtifact,
		Rule: "return true",
	}
	require.NoError(t, featureflipping.Insert(gorpmapping.Mapper, db, feat))
	t.Cleanup(func() {
		_ = featureflipping.Delete(db, feat.ID)
	})

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

	wrDB, err := workflow.LoadRunByID(db, wrCreate.ID, workflow.LoadRunOptions{})
	require.NoError(t, err)

	nr := wrDB.WorkflowNodeRuns[w.WorkflowData.Node.ID][0]
	nr.Status = sdk.StatusBuilding
	require.NoError(t, workflow.UpdateNodeRun(db, &nr))

	nrj := nr.Stages[0].RunJobs[0]
	nrj.Status = sdk.StatusBuilding
	workflow.UpdateNodeJobRun(context.Background(), db, &nrj)

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
		"number":           fmt.Sprintf("%d", wrCreate.Number),
	}

	artiData := sdk.WorkflowRunResultArtifact{
		Size:       1,
		MD5:        "AA",
		CDNRefHash: "AA",
		Name:       "myartifact",
		Perm:       0777,
	}
	bts, err := json.Marshal(artiData)
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
	require.NoError(t, api.Cache.SetWithTTL(workflow.GetArtifactResultKey(wrCreate.ID, artiData.Name), true, 60))

	//Do the request
	reqOK := assets.NewJWTAuthentifiedRequest(t, jwtCDN, "POST", uri, addResultRequest)
	recOK := httptest.NewRecorder()
	router.Mux.ServeHTTP(recOK, reqOK)
	assert.Equal(t, 204, recOK.Code)

	b, err := api.Cache.Exist(workflow.GetArtifactResultKey(wrCreate.ID, artiData.Name))
	require.NoError(t, err)
	require.False(t, b)

}

func Test_workflowRunArtifactCheckUpload(t *testing.T) {
	featureflipping.Init(gorpmapping.Mapper)
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)

	u, _ := assets.InsertAdminUser(t, db)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	feat := &sdk.Feature{
		Name: sdk.FeatureCDNArtifact,
		Rule: "return true",
	}
	require.NoError(t, featureflipping.Insert(gorpmapping.Mapper, db, feat))
	t.Cleanup(func() {
		_ = featureflipping.Delete(db, feat.ID)
	})

	w := assets.InsertTestWorkflow(t, db, api.Cache, proj, sdk.RandomString(10))

	wrCreate, err := workflow.CreateRun(api.mustDB(), w, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	assert.NoError(t, err)

	require.NoError(t, api.workflowRunCraft(context.TODO(), wrCreate.ID))

	wrDB, err := workflow.LoadRunByID(db, wrCreate.ID, workflow.LoadRunOptions{})
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
		"key":              proj.Key,
		"permWorkflowName": w.Name,
		"number":           fmt.Sprintf("%d", wrCreate.Number),
		"nodeID":           fmt.Sprintf("%d", wrCreate.Workflow.WorkflowData.Node.ID),
	}
	checkRequest := sdk.CDNArtifactAPIRef{
		ArtifactName: "myArtifact",
		RunID:        wrCreate.ID,
		RunNodeID:    nr.ID,
		RunJobID:     nrj.ID,
		WorkflowID:   w.ID,
		WorkflowName: w.Name,
		ProjectKey:   key,
	}

	uri := router.GetRoute("POST", api.workflowRunArtifactCheckUploadHandler, vars)
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtCDN, "POST", uri, checkRequest)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 204, rec.Code)
}
