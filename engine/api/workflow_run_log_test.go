package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/featureflipping"
	"github.com/ovh/cds/sdk"
)

func Test_getWorkflowNodeRunJobStepDeprecatedHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	u, pass, proj, w1, lastRun, jobRun := initGetWorkflowNodeRunJobTest(t, api, db)

	uri := router.GetRoute("GET", api.getWorkflowNodeRunJobStepDeprecatedHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
		"number":           fmt.Sprintf("%d", lastRun.Number),
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

func Test_getWorkflowNodeRunJobServiceLogsDeprecatedHandler(t *testing.T) {
	api, db, router := newTestAPI(t)

	u, pass, proj, w1, lastRun, jobRun := initGetWorkflowNodeRunJobTest(t, api, db)

	uri := router.GetRoute("GET", api.getWorkflowNodeRunJobServiceLogDeprecatedHandler, map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
		"number":           fmt.Sprintf("%d", lastRun.Number),
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

func Test_getWorkflowNodeRunJobLogHandler(t *testing.T) {
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

	//This is a mock for the cdn service
	services.HTTPClient = mock(
		func(r *http.Request) (*http.Response, error) {
			t.Logf("[MOCK] %s %v", r.Method, r.URL)
			body := new(bytes.Buffer)
			w := new(http.Response)
			enc := json.NewEncoder(body)
			w.Body = ioutil.NopCloser(body)

			if strings.HasPrefix(r.URL.String(), "/item/step-log/") {
				var res interface{}
				if err := enc.Encode(res); err != nil {
					return writeError(w, err)
				}
				return w, nil
			}

			return writeError(w, sdk.NewError(sdk.ErrServiceUnavailable,
				fmt.Errorf("route %s must not be called", r.URL.String()),
			))
		},
	)

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
	require.Equal(t, 200, rec.Code)

	var access sdk.CDNLogAccess
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &access))
	require.True(t, access.Exists)
	require.Equal(t, "http://cdn.net:8080", access.CDNURL)
	require.NotEmpty(t, access.DownloadPath)
	require.NotEmpty(t, access.Token)
}
