package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/featureflipping"
	"github.com/ovh/cds/sdk"
)

func Test_getWorkflowNodeRunJobStepLogHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	u, pass, proj, w1, lastRun, jobRun := initGetWorkflowNodeRunJobTest(t, api, db)

	uri := router.GetRoute("GET", api.getWorkflowNodeRunJobStepLogHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
		"nodeRunID":        fmt.Sprintf("%d", lastRun.WorkflowNodeRuns[w1.WorkflowData.Node.ID][0].ID),
		"runJobID":         fmt.Sprintf("%d", jobRun.ID),
		"stepOrder":        "0",
	})
	require.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)

	var stepState sdk.BuildState
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &stepState))
	require.Equal(t, 200, rec.Code)
	require.Equal(t, "123456789012345... truncated\n", stepState.StepLogs.Val)
	require.Equal(t, sdk.StatusBuilding, stepState.Status)
}

func Test_getWorkflowNodeRunJobServiceLogHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	u, pass, proj, w1, lastRun, jobRun := initGetWorkflowNodeRunJobTest(t, api, db)

	uri := router.GetRoute("GET", api.getWorkflowNodeRunJobServiceLogHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
		"nodeRunID":        fmt.Sprintf("%d", lastRun.WorkflowNodeRuns[w1.WorkflowData.Node.ID][0].ID),
		"runJobID":         fmt.Sprintf("%d", jobRun.ID),
		"serviceName":      "postgres",
	})
	require.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)

	var log sdk.ServiceLog
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &log))
	require.Equal(t, 200, rec.Code)
	require.Equal(t, "098765432109876... truncated\n", log.Val)
}

func Test_getWorkflowNodeRunJobLinkHandler(t *testing.T) {
	featureflipping.Init(gorpmapping.Mapper)

	api, db, router := newTestAPI(t)

	all, err := featureflipping.LoadAll(context.TODO(), gorpmapping.Mapper, db)
	require.NoError(t, err)
	for _, f := range all {
		require.NoError(t, featureflipping.Delete(db, f.ID))
	}

	u, pass, proj, w1, lastRun, jobRun := initGetWorkflowNodeRunJobTest(t, api, db)

	require.NoError(t, featureflipping.Insert(gorpmapping.Mapper, db, &sdk.Feature{
		Name: "cdn-job-logs",
		Rule: fmt.Sprintf("return project_key == \"%s\"", proj.Key),
	}))

	mockCDNService, _ := assets.InitCDNService(t, db)
	t.Cleanup(func() { _ = services.Delete(db, mockCDNService) })

	uri := router.GetRoute("GET", api.getWorkflowNodeRunJobStepLinkHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
		"nodeRunID":        fmt.Sprintf("%d", lastRun.WorkflowNodeRuns[w1.WorkflowData.Node.ID][0].ID),
		"runJobID":         fmt.Sprintf("%d", jobRun.ID),
		"stepOrder":        "0",
	})
	require.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "GET", uri, nil)
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)

	var link sdk.CDNLogLink
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &link))
	require.Equal(t, "http://cdn.net:8080", link.CDNURL)
	require.Equal(t, sdk.CDNTypeItemStepLog, link.ItemType)
	require.NotEmpty(t, link.APIRef)
}
