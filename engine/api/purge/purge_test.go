package purge

import (
	"context"
	"os"
	"path"
	"testing"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/golang/mock/gomock"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/services/mock_services"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/test"
)

func Test_deleteWorkflowRunsHistory(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)

	// Init store
	cfg := objectstore.Config{
		Kind: objectstore.Filesystem,
		Options: objectstore.ConfigOptions{
			Filesystem: objectstore.ConfigOptionsFilesystem{
				Basedir: path.Join(os.TempDir(), "store"),
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	servicesClients := mock_services.NewMockClient(ctrl)
	services.NewClient = func(_ gorp.SqlExecutor, _ []sdk.Service) services.Client {
		return servicesClients
	}
	defer func() {
		services.NewClient = services.NewDefaultClient
	}()

	servicesClients.EXPECT().
		DoJSONRequest(gomock.Any(), "POST", "/bulk/item/delete", gomock.Any(), gomock.Any(), gomock.Any()).MaxTimes(1)

	sharedStorage, errO := objectstore.Init(context.Background(), cfg)
	test.NoError(t, errO)

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
	cdnClient := services.NewClient(db, srvs)

	err = deleteRunHistory(context.Background(), db.DbMap, wr.ID, cdnClient, sharedStorage, nil)
	require.NoError(t, err)

	_, err = workflow.LoadRunByID(db, wr.ID, workflow.LoadRunOptions{})
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
	keep, err := applyRetentionPolicyOnRun(context.TODO(), db.DbMap, wf, run1, nil, sdk.Application{}, nil, MarkAsDeleteOptions{DryRun: true})
	require.NoError(t, err)
	require.False(t, keep)

	run2 := sdk.WorkflowRunSummary{
		LastModified: now.Add(-47 * time.Hour),
	}
	keep, err = applyRetentionPolicyOnRun(context.TODO(), db.DbMap, wf, run2, nil, sdk.Application{}, nil, MarkAsDeleteOptions{DryRun: true})
	require.NoError(t, err)
	require.True(t, keep)
}

func Test_applyRetentionPolicyOnRunWithError(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	defaultRunRetentionPolicy = "" // check empty rule
	keep, err := applyRetentionPolicyOnRun(context.TODO(), db.DbMap, sdk.Workflow{}, sdk.WorkflowRunSummary{}, nil, sdk.Application{}, nil, MarkAsDeleteOptions{DryRun: true})
	require.Error(t, err)
	require.True(t, keep)

	defaultRunRetentionPolicy = "unknown == 'true'" // check no return
	keep, err = applyRetentionPolicyOnRun(context.TODO(), db.DbMap, sdk.Workflow{}, sdk.WorkflowRunSummary{}, nil, sdk.Application{}, nil, MarkAsDeleteOptions{DryRun: true})
	require.Error(t, err)
	require.True(t, keep)

	defaultRunRetentionPolicy = "return unknown == 'true'" // check unknown variable
	keep, err = applyRetentionPolicyOnRun(context.TODO(), db.DbMap, sdk.Workflow{}, sdk.WorkflowRunSummary{}, nil, sdk.Application{}, nil, MarkAsDeleteOptions{DryRun: true})
	require.Error(t, err)
	require.True(t, keep)

	defaultRunRetentionPolicy = "return nil" // check return nil
	keep, err = applyRetentionPolicyOnRun(context.TODO(), db.DbMap, sdk.Workflow{}, sdk.WorkflowRunSummary{}, nil, sdk.Application{}, nil, MarkAsDeleteOptions{DryRun: true})
	require.Error(t, err)
	require.True(t, keep)

	defaultRunRetentionPolicy = "return git_branch_exist == 'false'"
	keep, err = applyRetentionPolicyOnRun(context.TODO(), db.DbMap, sdk.Workflow{}, sdk.WorkflowRunSummary{}, nil, sdk.Application{}, nil, MarkAsDeleteOptions{DryRun: true})
	require.NoError(t, err)
	require.True(t, keep)

	defaultRunRetentionPolicy = "return git_branch_exist == 'true'"
	keep, err = applyRetentionPolicyOnRun(context.TODO(), db.DbMap, sdk.Workflow{}, sdk.WorkflowRunSummary{}, nil, sdk.Application{}, nil, MarkAsDeleteOptions{DryRun: true})
	require.NoError(t, err)
	require.False(t, keep)
}
