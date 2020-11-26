package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/sdk/cdsclient"

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
	api, db, tsURL := newTestServer(t)

	event.OverridePubSubKey("events_pubsub_test")
	require.NoError(t, event.Initialize(context.Background(), api.mustDB(), api.Cache))
	require.NoError(t, api.initWebsocket("events_pubsub_test"))

	u, pass := assets.InsertAdminUser(t, db)
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	_, jws, err := builtin.NewConsumer(context.TODO(), db, sdk.RandomString(10), sdk.RandomString(10), localConsumer, u.GetGroupIDs(),
		sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeProject))

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
	require.NoError(t, workflow.UpdateMaxRunsByID(db, w.ID, 10))
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

	event.Initialize(context.TODO(), api.mustDB(), api.Cache)

	chanMessageReceived := make(chan sdk.WebsocketEvent)
	chanMessageToSend := make(chan []sdk.WebsocketFilter)
	chanErrorReceived := make(chan error)
	client := cdsclient.New(cdsclient.Config{
		Host:                              tsURL,
		User:                              u.Username,
		InsecureSkipVerifyTLS:             true,
		BuitinConsumerAuthenticationToken: jws,
	})
	contextWS, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go client.WebsocketEventsListen(contextWS, sdk.NewGoRoutines(), chanMessageToSend, chanMessageReceived, chanErrorReceived)

	// Subscribe to workflow retention
	chanMessageToSend <- []sdk.WebsocketFilter{{
		Type:         sdk.WebsocketFilterTypeDryRunRetentionWorkflow,
		ProjectKey:   proj.Key,
		WorkflowName: w1.Name,
	}}

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

	var result sdk.PurgeDryRunResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))

	require.Equal(t, int64(2), result.NbRunsToAnalize)

	run1DB, err := workflow.LoadRunByID(api.mustDB(), run2.ID, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
	require.NoError(t, err)
	require.False(t, run1DB.ToDelete)

	timeout := time.NewTimer(5 * time.Second)
	select {
	case <-timeout.C:
		t.Fatal("test timeout")
	case err := <-chanErrorReceived:
		t.Fatal(err)
	case evt := <-chanMessageReceived:
		require.Equal(t, "OK", evt.Status)
		var eventRun sdk.EventRetentionWorkflowDryRun
		require.NoError(t, json.Unmarshal(evt.Event.Payload, &eventRun))
		require.Len(t, eventRun.Runs, 1)
		require.Equal(t, eventRun.Runs[0].ID, run1.ID)
	}
}
