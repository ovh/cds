package purge

import (
	"context"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"gopkg.in/h2non/gock.v1"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
)

func Test_deleteWorkflowRunsHistory(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/bulk/item/delete", gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(1)

	p := assets.InsertTestProject(t, db, cache, sdk.RandomString(10), sdk.RandomString(10))
	w := assets.InsertTestWorkflow(t, db, cache, p, sdk.RandomString(10))

	wr, err := workflow.CreateRun(db.DbMap, w, sdk.WorkflowRunPostHandlerOption{
		Hook: &sdk.WorkflowNodeRunHookEvent{},
	})
	require.NoError(t, err)

	wr.ToDelete = true
	require.NoError(t, workflow.UpdateWorkflowRun(context.TODO(), db, wr))

	srvs, err := services.LoadAllByType(context.TODO(), db, sdk.TypeCDN)
	require.NoError(t, err)
	cdnClient := services.NewClient(srvs)

	err = deleteRunHistory(context.Background(), db.DbMap, wr.ID, cdnClient, nil)
	require.NoError(t, err)

	_, err = workflow.LoadRunByID(context.Background(), db, wr.ID, workflow.LoadRunOptions{})
	require.NotNil(t, err)
	require.True(t, sdk.ErrorIs(err, sdk.ErrNotFound))
	require.True(t, gock.IsDone())
}

func Test_applyRetentionPolicyOnRun(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)
	wf := sdk.Workflow{
		RetentionPolicy: "return run_days_before < 2",
	}
	now := time.Now()
	run1 := sdk.WorkflowRunSummary{
		LastModified: now.Add(-49 * time.Hour),
	}
	keep, err := applyRetentionPolicyOnRun(context.TODO(), db.DbMap, wf, run1, map[string]string{}, nil, sdk.Application{}, nil, MarkAsDeleteOptions{DryRun: true})
	require.NoError(t, err)
	require.False(t, keep)

	run2 := sdk.WorkflowRunSummary{
		LastModified: now.Add(-47 * time.Hour),
	}
	keep, err = applyRetentionPolicyOnRun(context.TODO(), db.DbMap, wf, run2, map[string]string{}, nil, sdk.Application{}, nil, MarkAsDeleteOptions{DryRun: true})
	require.NoError(t, err)
	require.True(t, keep)
}

func Test_applyRetentionPolicyOnRunWithError(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	// check empty rule
	keep, err := applyRetentionPolicyOnRun(context.TODO(), db.DbMap, sdk.Workflow{
		RetentionPolicy: "",
	}, sdk.WorkflowRunSummary{}, map[string]string{}, nil, sdk.Application{}, nil, MarkAsDeleteOptions{DryRun: true})
	require.Error(t, err)
	require.True(t, keep)

	// check no return
	keep, err = applyRetentionPolicyOnRun(context.TODO(), db.DbMap, sdk.Workflow{
		RetentionPolicy: "unknown == 'true'",
	}, sdk.WorkflowRunSummary{}, map[string]string{}, nil, sdk.Application{}, nil, MarkAsDeleteOptions{DryRun: true})
	require.Error(t, err)
	require.True(t, keep)

	// check unknown variable
	keep, err = applyRetentionPolicyOnRun(context.TODO(), db.DbMap, sdk.Workflow{
		RetentionPolicy: "return unknown == 'true'",
	}, sdk.WorkflowRunSummary{}, map[string]string{}, nil, sdk.Application{}, nil, MarkAsDeleteOptions{DryRun: true})
	require.Error(t, err)
	require.True(t, keep)

	// check return nil
	keep, err = applyRetentionPolicyOnRun(context.TODO(), db.DbMap, sdk.Workflow{
		RetentionPolicy: "return nil",
	}, sdk.WorkflowRunSummary{}, map[string]string{}, nil, sdk.Application{}, nil, MarkAsDeleteOptions{DryRun: true})
	require.Error(t, err)
	require.True(t, keep)

	keep, err = applyRetentionPolicyOnRun(context.TODO(), db.DbMap, sdk.Workflow{
		RetentionPolicy: "return run_status == 'Success'",
	}, sdk.WorkflowRunSummary{
		Status: sdk.StatusSuccess,
	}, map[string]string{}, nil, sdk.Application{}, nil, MarkAsDeleteOptions{DryRun: true})
	require.NoError(t, err)
	require.True(t, keep)

	keep, err = applyRetentionPolicyOnRun(context.TODO(), db.DbMap, sdk.Workflow{
		RetentionPolicy: "return run_status ~= 'Success'",
	}, sdk.WorkflowRunSummary{
		Status: sdk.StatusSuccess,
	}, map[string]string{}, nil, sdk.Application{}, nil, MarkAsDeleteOptions{DryRun: true})
	require.NoError(t, err)
	require.False(t, keep)
}
