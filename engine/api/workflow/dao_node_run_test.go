package workflow_test

import (
	"context"
	"database/sql"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestLoadNodeRunIDs(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	u, _ := assets.InsertAdminUser(t, db)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	w := sdk.Workflow{
		Name:       "mywork",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		WorkflowData: sdk.WorkflowData{
			Node: sdk.Node{
				Name: "node1",
				Type: sdk.NodeTypeFork,
			},
		},
	}

	test.NoError(t, workflow.Insert(context.TODO(), db, cache, *proj, &w))

	wr1, err := workflow.CreateRun(db.DbMap, &w, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	require.NoError(t, err)

	wr2, err := workflow.CreateRun(db.DbMap, &w, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	require.NoError(t, err)

	nr1 := &workflow.NodeRun{
		WorkflowRunID:  wr1.ID,
		WorkflowID:     sql.NullInt64{Int64: w.ID, Valid: true},
		WorkflowNodeID: w.WorkflowData.Node.ID,
		Status:         sdk.StatusSuccess,
		Start:          time.Now(),
	}
	nr2 := &workflow.NodeRun{

		WorkflowRunID:  wr2.ID,
		WorkflowID:     sql.NullInt64{Int64: w.ID, Valid: true},
		WorkflowNodeID: w.WorkflowData.Node.ID,
		Status:         sdk.StatusSuccess,
		Start:          time.Now(),
	}

	require.NoError(t, db.Insert(nr1))
	require.NoError(t, db.Insert(nr2))

	require.NoError(t, workflow.AppendLog(db, 1, nr1.ID, 0, "mylog", 10000000))

	ids, err := workflow.LoadNodeRunIDsWithLogs(db, []int64{w.ID}, []string{sdk.StatusFail, sdk.StatusStopped, sdk.StatusSuccess})
	require.NoError(t, err)

	require.Equal(t, 1, len(ids))
	require.Equal(t, nr1.ID, ids[0].NodeRunID)
}
