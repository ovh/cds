package worker_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
)

func TestUpdate(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitializeDB)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

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
	require.NoError(t, workflow.Insert(context.TODO(), db, cache, *proj, &wk))

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
	}
	dbj := new(workflow.JobRun)
	require.NoError(t, dbj.ToJobRun(&jobRun))
	require.NoError(t, db.Insert(dbj))
	jobRun.ID = dbj.ID
	t.Cleanup(func() {
		workflow.DeleteNodeJobRun(db, jobRun.ID)
	})

	w := sdk.Worker{
		ID:         sdk.UUID(),
		ConsumerID: sdk.UUID(),
		Status:     sdk.StatusBuilding,
		Name:       sdk.RandomString(10),
	}
	require.NoError(t, worker.Insert(context.TODO(), db, &w))

	w.PrivateKey = []byte("je suis la cle")
	require.NoError(t, worker.SetToBuilding(context.TODO(), db, w.ID, jobRun.ID, w.PrivateKey))

	wDB, err := worker.LoadWorkerByNameWithDecryptKey(context.TODO(), db, w.Name)
	require.NoError(t, err)
	require.Equal(t, sdk.StatusBuilding, wDB.Status)
	require.Equal(t, string(w.PrivateKey), string(wDB.PrivateKey))

	require.NoError(t, worker.SetStatus(context.TODO(), db, w.ID, sdk.StatusWaiting))
	wDB, err = worker.LoadWorkerByNameWithDecryptKey(context.TODO(), db, w.Name)
	require.NoError(t, err)
	require.Nil(t, wDB.JobRunID)
	require.Equal(t, sdk.StatusWaiting, wDB.Status)
	require.Equal(t, string(w.PrivateKey), string(wDB.PrivateKey))

}
