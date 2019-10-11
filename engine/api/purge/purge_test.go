package purge

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/test"
)

func Test_deleteWorkflowRunsHistory(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

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

	err := deleteWorkflowRunsHistory(context.Background(), db, cache, sharedStorage, nil)
	test.NoError(t, err)

	// test on delete artifact from storage is done on Test_postWorkflowJobArtifactHandler
}
