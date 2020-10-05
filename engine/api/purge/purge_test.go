package purge

import (
	"context"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/test"
)

func Test_deleteWorkflowRunsHistory(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)

	// Init store
	cfg := objectstore.Config{
		Kind: objectstore.Filesystem,
		Options: objectstore.ConfigOptions{
			Filesystem: objectstore.ConfigOptionsFilesystem{
				Basedir: path.Join(os.TempDir(), "store"),
			},
		},
	}

	sharedStorage, errO := objectstore.Init(context.Background(), cfg)
	test.NoError(t, errO)

	err := deleteWorkflowRunsHistory(context.Background(), db.DbMap, sharedStorage, nil)
	test.NoError(t, err)

	// test on delete artifact from storage is done on Test_postWorkflowJobArtifactHandler
}

func Test_applyRetentionPolicyOnRun(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitiliazeDB)
	wf := sdk.Workflow{
		RetentionPolicy: "return run_date_before < 2",
	}
	now := time.Now()
	run1 := sdk.WorkflowRun{
		LastModified: now.Add(-49 * time.Hour),
	}
	keep, err := applyRetentionPolicyOnRun(db.DbMap, wf, run1, nil, MarkAsDeleteOptions{DryRun: true})
	require.NoError(t, err)
	require.False(t, keep)

	run2 := sdk.WorkflowRun{
		LastModified: now.Add(-47 * time.Hour),
	}
	keep, err = applyRetentionPolicyOnRun(db.DbMap, wf, run2, nil, MarkAsDeleteOptions{DryRun: true})
	require.NoError(t, err)
	require.True(t, keep)
}
