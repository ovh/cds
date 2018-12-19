package migrate_test

import (
	"testing"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/migrate"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func TestCheckMigrations_WithError(t *testing.T) {
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
	mig2 := sdk.Migration{
		Name:      "secondOne",
		Release:   "0.37.0",
		Mandatory: true,
	}
	mig3 := sdk.Migration{
		Name:      "thirdOne",
		Release:   "snapshot",
		Mandatory: true,
	}

	migrate.Add(mig1)
	migrate.Add(mig2)
	migrate.Add(mig3)
	defer migrate.CleanMigrationsList()

	sdk.VERSION = "0.39.0"

	test.NotNil(t, migrate.CheckMigrations(db))
}
func TestCheckMigrations_WithoutError(t *testing.T) {
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
	mig2 := sdk.Migration{
		Name:      "secondOne",
		Release:   "0.37.0",
		Mandatory: true,
	}
	test.NoError(t, migrate.Insert(db, &mig2))
	defer func() {
		_ = migrate.Delete(db, &mig2)
	}()
	mig3 := sdk.Migration{
		Name:      "thirdOne",
		Release:   "snapshot",
		Mandatory: true,
	}
	test.NoError(t, migrate.Insert(db, &mig3))
	defer func() {
		_ = migrate.Delete(db, &mig3)
	}()

	migrate.Add(mig1)
	migrate.Add(mig2)
	migrate.Add(mig3)
	defer migrate.CleanMigrationsList()

	sdk.VERSION = "0.39.0"

	test.NoError(t, migrate.CheckMigrations(db))

	sdk.VERSION = "0.36.0"
	test.NoError(t, migrate.CheckMigrations(db))
}
