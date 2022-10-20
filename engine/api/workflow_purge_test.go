package api

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/authentication/builtin"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/engine/api/integration"
	"github.com/ovh/cds/engine/api/integration/artifact_manager"
	"github.com/ovh/cds/engine/api/integration/artifact_manager/mock_artifact_manager"
	"github.com/ovh/cds/engine/api/purge"
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
	require.NoError(t, event.Initialize(context.Background(), api.mustDB(), api.Cache, nil))
	require.NoError(t, api.initWebsocket("events_pubsub_test"))

	u, pass := assets.InsertAdminUser(t, db)
	localConsumer, err := authentication.LoadConsumerByTypeAndUserID(context.TODO(), api.mustDB(), sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)
	require.NoError(t, err)

	consumerOptions := builtin.NewConsumerOptions{
		Name:     sdk.RandomString(10),
		GroupIDs: u.GetGroupIDs(),
		Scopes:   sdk.NewAuthConsumerScopeDetails(sdk.AuthConsumerScopeProject),
	}
	_, jws, err := builtin.NewConsumer(context.TODO(), db, consumerOptions, localConsumer)
	require.NoError(t, err)

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

	event.Initialize(context.TODO(), api.mustDB(), api.Cache, nil)

	chanMessageReceived := make(chan sdk.WebsocketEvent)
	chanMessageToSend := make(chan []sdk.WebsocketFilter)
	chanErrorReceived := make(chan error)
	client := cdsclient.New(cdsclient.Config{
		Host:                               tsURL,
		User:                               u.Username,
		InsecureSkipVerifyTLS:              true,
		BuiltinConsumerAuthenticationToken: jws,
	})
	contextWS, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go client.WebsocketEventsListen(contextWS, sdk.NewGoRoutines(context.TODO()), chanMessageToSend, chanMessageReceived, chanErrorReceived)

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

	run1DB, err := workflow.LoadRunByID(context.Background(), api.mustDB(), run2.ID, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
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

func Test_Purge_DeleteArtifactsFromRepositoryManager(t *testing.T) {
	ctx := context.Background()
	api, db, _ := newTestAPI(t)

	// Create user
	u, _ := assets.InsertAdminUser(t, db)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	p := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	w := assets.InsertTestWorkflow(t, db, api.Cache, p, sdk.RandomString(10))

	integrationModel, err := integration.LoadModelByName(context.TODO(), db, sdk.ArtifactoryIntegration.Name)
	require.NoError(t, err)

	integrationConfig := integrationModel.DefaultConfig.Clone()
	integrationConfig.SetValue("url", "https://artifactory.lolcat.local/")
	integrationConfig.SetValue("platform", "artifactory")
	integrationConfig.SetValue("token.name", "my-token")
	integrationConfig.SetValue("token", "abcdef")
	integrationConfig.SetValue("release.token", "abcdef")
	integrationConfig.SetValue("project.key", "my-project-key")
	integrationConfig.SetValue("cds.repository", "my-repository")
	integrationConfig.SetValue("promotion.maturity.low", "snapshot")
	integrationConfig.SetValue("promotion.maturity.high", "release")

	projectIntegration := sdk.ProjectIntegration{
		ProjectID:          p.ID,
		IntegrationModelID: integrationModel.ID,
		Config:             integrationConfig,
		Model:              integrationModel,
	}

	require.NoError(t, integration.InsertIntegration(db, &projectIntegration))

	p.Integrations = append(p.Integrations, projectIntegration)

	w.Integrations = append(w.Integrations,
		sdk.WorkflowProjectIntegration{
			WorkflowID:           w.ID,
			ProjectIntegrationID: projectIntegration.ID,
			Config:               integrationModel.AdditionalDefaultConfig.Clone(),
			ProjectIntegration:   projectIntegration,
		},
	)

	workflow.Update(ctx, db, api.Cache, *p, w, workflow.UpdateOptions{
		DisableHookManagement: true,
	})

	w, err = workflow.Load(ctx, db, api.Cache, *p, w.Name, workflow.LoadOptions{
		DeepPipeline:          true,
		WithAsCodeUpdateEvent: true,
		WithIcon:              true,
		WithIntegrations:      true,
		WithTemplate:          true,
	})
	require.NoError(t, err)

	wr, err := workflow.CreateRun(db.DbMap, w, sdk.WorkflowRunPostHandlerOption{
		Hook: &sdk.WorkflowNodeRunHookEvent{},
	})
	require.NoError(t, err)

	opts := sdk.WorkflowRunPostHandlerOption{
		Manual:         &sdk.WorkflowNodeRunManual{},
		AuthConsumerID: consumer.ID,
	}

	api.initWorkflowRun(ctx, p.Key, w, wr, opts)

	wr, err = workflow.LoadRunByID(context.Background(), db.DbMap, wr.ID, workflow.LoadRunOptions{
		WithTests:           true,
		WithVulnerabilities: true,
	})
	require.NoError(t, err)

	data := sdk.WorkflowRunResultArtifactManager{
		WorkflowRunResultArtifactCommon: sdk.WorkflowRunResultArtifactCommon{
			Name: "foo",
		},
		Path:     "path/to/foo",
		RepoName: "repository",
	}
	rawData, _ := json.Marshal(data)

	ctrl := gomock.NewController(t)
	t.Cleanup(func() {
		ctrl.Finish()
	})

	mockArtifactory := mock_artifact_manager.NewMockArtifactManager(ctrl)

	mockArtifactory.EXPECT().GetFileInfo("repository", "path/to/foo").Return(
		sdk.FileInfo{
			Type:      "generic",
			Checksums: &sdk.FileInfoChecksum{},
		},
		nil)

	mockArtifactory.EXPECT().SetProperties("repository-snapshot", "path/to/foo", gomock.Any(), gomock.Any()).Return(nil)

	artifact_manager.DefaultClientFactory = func(_, _, _ string) (artifact_manager.ArtifactManager, error) {
		return mockArtifactory, nil
	}

	api.Cache.SetWithTTL(workflow.GetRunResultKey(wr.ID, sdk.WorkflowRunResultTypeArtifactManager, data.Name), true, 60)
	require.NoError(t,
		workflow.AddResult(ctx, db.DbMap, api.Cache, wr,
			&sdk.WorkflowRunResult{
				Type:              sdk.WorkflowRunResultTypeArtifactManager,
				Created:           time.Now(),
				WorkflowRunID:     wr.ID,
				WorkflowNodeRunID: wr.RootRun().ID,
				WorkflowRunJobID:  wr.RootRun().Stages[0].RunJobs[0].ID,
				DataRaw:           rawData,
			},
		),
	)

	require.NoError(t, purge.DeleteArtifactsFromRepositoryManager(ctx, db, wr))
}
