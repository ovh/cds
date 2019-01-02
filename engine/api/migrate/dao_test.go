package migrate_test

import (
	"testing"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/migrate"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func TestInsert(t *testing.T) {
	db, _, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()

	mig1 := sdk.Migration{
		Name:      "firstOne",
		Release:   "0.35.0",
		Mandatory: true,
	}
	test.NoError(t, migrate.Insert(db, &mig1))
	defer func() {
		_ = migrate.Delete(db, &mig1)
	}()
	test.Equal(t, uint64(0), mig1.Major)
	test.Equal(t, uint64(35), mig1.Minor)
	test.Equal(t, uint64(0), mig1.Patch)

	mig2 := sdk.Migration{
		Name:      "thirdOne",
		Release:   "snapshot",
		Mandatory: true,
	}
	test.NoError(t, migrate.Insert(db, &mig2))
	defer func() {
		_ = migrate.Delete(db, &mig2)
	}()
	test.Equal(t, uint64(0), mig2.Major)
	test.Equal(t, uint64(0), mig2.Minor)
	test.Equal(t, uint64(0), mig2.Patch)

	mig3 := sdk.Migration{
		Name:      "fourthOne",
		Release:   "1.39.3",
		Mandatory: true,
	}
	test.NoError(t, migrate.Insert(db, &mig3))
	defer func() {
		_ = migrate.Delete(db, &mig3)
	}()
	test.Equal(t, uint64(1), mig3.Major)
	test.Equal(t, uint64(39), mig3.Minor)
	test.Equal(t, uint64(3), mig3.Patch)
}
