package migrate

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// GetAll returns the migration for given name.
func GetAll(db gorp.SqlExecutor) ([]sdk.Migration, error) {
	var migs []sdk.Migration

	if _, err := db.Select(&migs, "SELECT cds_migration.* FROM cds_migration"); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "Cannot get migrations")
	}

	return migs, nil
}

// GetByName returns the migration for given name.
func GetByName(db gorp.SqlExecutor, name string) (*sdk.Migration, error) {
	var mig sdk.Migration

	if err := db.SelectOne(&mig, "SELECT * FROM cds_migration WHERE name = $1", name); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "Cannot get migration")
	}

	return &mig, nil
}

// Insert migration in database.
func Insert(db gorp.SqlExecutor, mig *sdk.Migration) error {
	mig.Created = time.Now()
	return sdk.WrapError(gorpmapping.Insert(db, mig), "Unable to insert migration %s", mig.Name)
}

// Update migration in database.
func Update(db gorp.SqlExecutor, mig *sdk.Migration) error {
	return sdk.WrapError(gorpmapping.Update(db, mig), "Unable to update migration %s", mig.Name)
}

// Delete migration in database.
func Delete(db gorp.SqlExecutor, mig *sdk.Migration) error {
	return sdk.WrapError(gorpmapping.Delete(db, mig), "Unable to delete migration %s", mig.Name)
}

// UpdateStatus update the status of a migration given its id
func UpdateStatus(db gorp.SqlExecutor, id int64, status string) error {
	_, err := db.Exec("UPDATE cds_migration SET status = $1 WHERE id = $2", status, id)
	return err
}

// CheckMigrations checks if all mandatory migrations are done
func CheckMigrations(db gorp.SqlExecutor) error {
	var migs []sdk.Migration
	_, err := db.Select(&migs, "SELECT cds_migration.* FROM cds_migration WHERE release <> $1 AND status <> $2 AND status <> $3", sdk.VersionCurrent().Version, sdk.MigrationStatusDone, sdk.MigrationStatusCanceled)
	if err != nil && err != sql.ErrNoRows {
		return sdk.WrapError(err, "Cannot load migrations to check")
	}

	migrationsLength := len(migs)
	if migs != nil && migrationsLength > 0 {
		var migrationsList string
		for i, mig := range migs {
			migrationsList += mig.Name
			if i != migrationsLength-1 {
				migrationsList += ", "
			}
		}
		return fmt.Errorf("There are some mandatory migrations which aren't done. Please check each changelog of CDS. Maybe you have skipped a release migration. (List of missing migrations: [%s])", migrationsList)
	}

	return nil
}
