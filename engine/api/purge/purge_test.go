package purge

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func Test_deleteWorkflowRunsHistory(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	sharedStorage := &objectstore.FilesystemStore{ProjectIntegration: sdk.ProjectIntegration{Name: sdk.DefaultStorageIntegrationName}, Basedir: path.Join(os.TempDir(), "store")}

	err := deleteWorkflowRunsHistory(context.Background(), db, cache, sharedStorage, nil)
	test.NoError(t, err)

	// test on delete artifact from storage is done on Test_postWorkflowJobArtifactHandler
}
