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

	mockCDNService, _, _ := assets.InitCDNService(t, db)

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
	require.Equal(t, sdk.CDNTypeItemStepLog, link.ItemType)
	require.NotEmpty(t, link.APIRef)
}
