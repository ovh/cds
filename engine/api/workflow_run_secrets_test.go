package api

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func Test_cleanSecretsSnapshotForRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	api, db, _ := newTestAPI(t)

	u, _ := assets.InsertAdminUser(t, db)
	consumer, _ := authentication.LoadUserConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadUserConsumerOptions.WithAuthentifiedUser)
	projectKey := sdk.RandomString(10)
	p := assets.InsertTestProject(t, db, api.Cache, projectKey, projectKey)

	require.NoError(t, project.InsertVariable(db, p.ID, &sdk.ProjectVariable{
		Type:  sdk.SecretVariable,
		Name:  "my-secret",
		Value: "my-value",
	}, u))

	w := assets.InsertTestWorkflow(t, db, api.Cache, p, sdk.RandomString(10))
	wr, err := workflow.CreateRun(db.DbMap, w, sdk.WorkflowRunPostHandlerOption{
		Hook: &sdk.WorkflowNodeRunHookEvent{},
	})
	require.NoError(t, err)
	api.initWorkflowRun(ctx, p.Key, w, wr, sdk.WorkflowRunPostHandlerOption{
		Manual:         &sdk.WorkflowNodeRunManual{},
		AuthConsumerID: consumer.ID,
	})

	runIDs, err := workflow.LoadRunsIDsCreatedBefore(ctx, db, time.Now(), 100)
	require.NoError(t, err)
	require.Contains(t, runIDs, wr.ID)

	runIDs, err = workflow.LoadRunsIDsCreatedBefore(ctx, db, wr.Start, 100)
	require.NoError(t, err)
	require.NotContains(t, runIDs, wr.ID)

	count, err := workflow.CountRunSecretsByWorkflowRunID(ctx, db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)

	require.NoError(t, api.cleanWorkflowRunSecretsForRun(ctx, db.DbMap, wr.ID))

	result, err := workflow.LoadRunByID(ctx, db, wr.ID, workflow.LoadRunOptions{})
	require.NoError(t, err)
	require.True(t, result.ReadOnly)

	count, err = workflow.CountRunSecretsByWorkflowRunID(ctx, db, wr.ID)
	require.NoError(t, err)
	require.Equal(t, int64(0), count)
}
