package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func Test_purgeDryRunHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)

	u, pass := assets.InsertAdminUser(t, db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(api.mustDB(), &pip))

	w := sdk.Workflow{
		Name:       sdk.RandomString(10),
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
			},
		},
	}
	test.NoError(t, workflow.RenameNode(context.Background(), db, &w))

	proj, _ = project.Load(context.TODO(), api.mustDB(), proj.Key,
		project.LoadOptions.WithPipelines,
		project.LoadOptions.WithGroups,
	)

	require.NoError(t, workflow.Insert(context.TODO(), db, api.Cache, *proj, &w))
	w1, err := workflow.Load(context.TODO(), api.mustDB(), api.Cache, *proj, w.Name, workflow.LoadOptions{})
	test.NoError(t, err)

	run1, err := workflow.CreateRun(api.mustDB(), w1, sdk.WorkflowRunPostHandlerOption{Hook: &sdk.WorkflowNodeRunHookEvent{}})
	require.NoError(t, err)

	run2, err := workflow.CreateRun(api.mustDB(), w1, sdk.WorkflowRunPostHandlerOption{Hook: &sdk.WorkflowNodeRunHookEvent{}})
	require.NoError(t, err)

	run1.Status = sdk.StatusSuccess
	require.NoError(t, workflow.UpdateWorkflowRunStatus(api.mustDB(), run1))

	run2.Status = sdk.StatusFail
	require.NoError(t, workflow.UpdateWorkflowRunStatus(api.mustDB(), run2))

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w1.Name,
	}
	request := sdk.PurgeDryRunRequest{RetentionPolicy: "return run_status == 'Success'"}
	uri := api.Router.GetRoute("POST", api.postWorkflowRetentionPolicyDryRun, vars)
	test.NotEmpty(t, uri)
	req := assets.NewAuthentifiedRequest(t, u, pass, "POST", uri, request)

	//Do the request
	rec := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 200, rec.Code)

	var result []sdk.PurgeRunToDelete
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))

	require.Len(t, result, 1)
	require.Equal(t, run2.ID, result[0].ID)

	run2DB, err := workflow.LoadRunByID(api.mustDB(), run2.ID, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
	require.NoError(t, err)
	require.False(t, run2DB.ToDelete)

}
