package workflow_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func createRunNodeRunAndJob(t *testing.T, db gorpmapper.SqlExecutorWithTx, store cache.Store) (sdk.Project, sdk.Workflow, sdk.WorkflowRun, workflow.NodeRun, sdk.WorkflowNodeJobRun) {
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, store, key, key)

	wk := sdk.Workflow{
		Name:       "coucou",
		ProjectKey: key,
		ProjectID:  proj.ID,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "coucou",
			},
		},
	}
	require.NoError(t, workflow.Insert(context.TODO(), db, store, *proj, &wk))

	wr := workflow.Run{
		WorkflowID: wk.ID,
		Workflow:   wk,
		ProjectID:  proj.ID,
	}
	require.NoError(t, db.Insert(&wr))

	nodeRun := workflow.NodeRun{
		WorkflowRunID:  wr.ID,
		WorkflowID:     sql.NullInt64{Int64: wk.ID},
		WorkflowNodeID: wk.WorkflowData.Node.ID,
	}
	require.NoError(t, db.Insert(&nodeRun))
	jobRun := sdk.WorkflowNodeJobRun{
		ID:                1,
		ProjectID:         proj.ID,
		WorkflowNodeRunID: nodeRun.ID,
		Status:            sdk.StatusBuilding,
	}
	dbj := new(workflow.JobRun)
	require.NoError(t, dbj.ToJobRun(&jobRun))
	require.NoError(t, db.Insert(dbj))
	jobRun.ID = dbj.ID

	workflowRun, err := workflow.LoadRunByID(context.Background(), db, wr.ID, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
	require.NoError(t, err)
	return *proj, wk, *workflowRun, nodeRun, jobRun
}

func TestCanUploadArtifactTerminatedJob(t *testing.T) {
	ctx := context.Background()
	db, store := test.SetupPG(t)

	_, _, workflowRun, nodeRun, jobRun := createRunNodeRunAndJob(t, db, store)

	artifactRef := sdk.WorkflowRunResultCheck{
		RunJobID:  jobRun.ID,
		RunNodeID: nodeRun.ID,
		RunID:     workflowRun.ID,
		Name:      "myartifact",
	}

	jobRun.Status = sdk.StatusSuccess
	require.NoError(t, workflow.UpdateNodeJobRun(ctx, db, &jobRun))

	_, err := workflow.CanUploadRunResult(ctx, db.DbMap, store, workflowRun, artifactRef)
	require.True(t, sdk.ErrorIs(err, sdk.ErrInvalidData))
	require.Contains(t, err.Error(), "unable to upload artifact on a terminated job")
}

func TestCanUploadArtifactWrongNodeRun(t *testing.T) {
	ctx := context.Background()
	db, store := test.SetupPG(t)

	_, _, workflowRun, nodeRun, jobRun := createRunNodeRunAndJob(t, db, store)

	artifactRef := sdk.WorkflowRunResultCheck{
		RunJobID:  jobRun.ID,
		RunNodeID: nodeRun.ID + 1,
		RunID:     workflowRun.ID,
		Name:      "myartifact",
	}

	_, err := workflow.CanUploadRunResult(ctx, db.DbMap, store, workflowRun, artifactRef)
	require.True(t, sdk.ErrorIs(err, sdk.ErrNotFound))
	require.Contains(t, err.Error(), "unable to find node run")
}

func TestCanUploadArtifactAlreadyExist(t *testing.T) {
	ctx := context.Background()
	db, store := test.SetupPG(t)

	_, _, workflowRun, nodeRun, jobRun := createRunNodeRunAndJob(t, db, store)

	artifactRef := sdk.WorkflowRunResultCheck{
		RunJobID:   jobRun.ID,
		RunNodeID:  nodeRun.ID,
		RunID:      workflowRun.ID,
		Name:       "myartifact",
		ResultType: sdk.WorkflowRunResultTypeArtifact,
	}

	result := sdk.WorkflowRunResult{
		ID:                sdk.UUID(),
		Created:           time.Now(),
		WorkflowNodeRunID: nodeRun.ID,
		WorkflowRunID:     workflowRun.ID,
		SubNum:            0,
		WorkflowRunJobID:  jobRun.ID + 1,
		Type:              sdk.WorkflowRunResultTypeArtifact,
	}
	artiData := sdk.WorkflowRunResultArtifact{
		Name:       "myartifact",
		CDNRefHash: "123",
		MD5:        "123",
		Size:       1,
		Perm:       0777,
	}
	bts, err := json.Marshal(artiData)
	require.NoError(t, err)
	result.DataRaw = bts

	cacheKey := workflow.GetRunResultKey(result.WorkflowRunID, sdk.WorkflowRunResultTypeArtifact, artiData.Name)
	require.NoError(t, store.SetWithTTL(cacheKey, true, 60))
	require.NoError(t, workflow.AddResult(ctx, db.DbMap, store, &workflowRun, &result))
	b, err := store.Exist(cacheKey)
	require.NoError(t, err)
	require.False(t, b)

	_, err = workflow.CanUploadRunResult(ctx, db.DbMap, store, workflowRun, artifactRef)
	require.True(t, sdk.ErrorIs(err, sdk.ErrConflictData))
	require.Contains(t, err.Error(), "artifact myartifact has already been uploaded")
}

func TestCanUploadArtifactAlreadyExistInMoreRecentSubNum(t *testing.T) {
	ctx := context.Background()
	db, store := test.SetupPG(t)

	_, _, workflowRun, nodeRun, jobRun := createRunNodeRunAndJob(t, db, store)

	artifactRef := sdk.WorkflowRunResultCheck{
		RunJobID:   jobRun.ID,
		RunNodeID:  nodeRun.ID,
		RunID:      workflowRun.ID,
		Name:       "myartifact",
		ResultType: sdk.WorkflowRunResultTypeArtifact,
	}

	result := sdk.WorkflowRunResult{
		ID:                sdk.UUID(),
		Created:           time.Now(),
		WorkflowNodeRunID: nodeRun.ID,
		WorkflowRunID:     workflowRun.ID,
		SubNum:            1,
		WorkflowRunJobID:  jobRun.ID + 1,
		Type:              sdk.WorkflowRunResultTypeArtifact,
	}
	artiData := sdk.WorkflowRunResultArtifact{
		Name:       "myartifact",
		CDNRefHash: "123",
		MD5:        "123",
		Size:       1,
		Perm:       0777,
	}
	bts, err := json.Marshal(artiData)
	require.NoError(t, err)
	result.DataRaw = bts

	cacheKey := workflow.GetRunResultKey(result.WorkflowRunID, sdk.WorkflowRunResultTypeArtifact, artiData.Name)
	require.NoError(t, store.SetWithTTL(cacheKey, true, 60))
	require.NoError(t, workflow.AddResult(ctx, db.DbMap, store, &workflowRun, &result))
	b, err := store.Exist(cacheKey)
	require.NoError(t, err)
	require.False(t, b)

	_, err = workflow.CanUploadRunResult(ctx, db.DbMap, store, workflowRun, artifactRef)
	require.True(t, sdk.ErrorIs(err, sdk.ErrConflictData))
	require.Contains(t, err.Error(), "artifact myartifact cannot be uploaded into a previous run")
}

func TestCanUploadArtifactAlreadyExistInAPreviousSubNum(t *testing.T) {
	ctx := context.Background()
	db, store := test.SetupPG(t)

	_, wk, workflowRun, nodeRun, jobRun := createRunNodeRunAndJob(t, db, store)

	nodeRun2 := workflow.NodeRun{
		WorkflowRunID:  workflowRun.ID,
		WorkflowID:     sql.NullInt64{Int64: wk.ID},
		WorkflowNodeID: wk.WorkflowData.Node.ID,
		SubNumber:      1,
	}
	require.NoError(t, db.Insert(&nodeRun2))
	jobRun.WorkflowNodeRunID = nodeRun2.ID
	require.NoError(t, workflow.UpdateNodeJobRun(ctx, db, &jobRun))

	run2, err := workflow.LoadRunByID(context.Background(), db, workflowRun.ID, workflow.LoadRunOptions{DisableDetailledNodeRun: true})
	require.NoError(t, err)
	workflowRun = *run2

	artifactRef := sdk.WorkflowRunResultCheck{
		RunJobID:  jobRun.ID,
		RunNodeID: nodeRun2.ID,
		RunID:     workflowRun.ID,
		Name:      "myartifact",
	}

	result := sdk.WorkflowRunResult{
		ID:                sdk.UUID(),
		Created:           time.Now(),
		WorkflowNodeRunID: nodeRun.ID,
		WorkflowRunID:     workflowRun.ID,
		SubNum:            0,
		WorkflowRunJobID:  jobRun.ID + 1,
		Type:              sdk.WorkflowRunResultTypeArtifact,
	}
	artiData := sdk.WorkflowRunResultArtifact{
		Name:       "myartifact",
		CDNRefHash: "123",
		MD5:        "123",
		Size:       1,
		Perm:       0777,
	}
	bts, err := json.Marshal(artiData)
	require.NoError(t, err)
	result.DataRaw = bts

	cacheKey := workflow.GetRunResultKey(result.WorkflowRunID, sdk.WorkflowRunResultTypeArtifact, artiData.Name)
	require.NoError(t, store.SetWithTTL(cacheKey, true, 60))
	require.NoError(t, workflow.AddResult(ctx, db.DbMap, store, &workflowRun, &result))
	b, err := store.Exist(cacheKey)
	require.NoError(t, err)
	require.False(t, b)

	_, err = workflow.CanUploadRunResult(ctx, db.DbMap, store, workflowRun, artifactRef)
	require.NoError(t, err)
}
