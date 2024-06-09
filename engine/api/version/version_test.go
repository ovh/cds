package version_test

import (
	"testing"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/version"
	"github.com/ovh/cds/sdk"
)

func TestMaxInstall(t *testing.T) {
	db, _ := test.SetupPG(t, bootstrap.InitializeDB)

	sdk.VERSION = "0.37.0"
	test.NoError(t, version.Upsert(db))

	sdk.VERSION = "1.35.1"
	test.NoError(t, version.Upsert(db))

	sdk.VERSION = "1.36.5"
	test.NoError(t, version.Upsert(db))

	major, minor, patch, err := version.MaxVersion(db)
	test.NoError(t, err)

	test.Equal(t, uint64(1), major)
	test.Equal(t, uint64(36), minor)
	test.Equal(t, uint64(5), patch)
}
