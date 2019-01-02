package migrate

import (
	"database/sql"
	"time"

	"github.com/blang/semver"
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
	if mig.Release != "" && mig.Release != "snapshot" {
		v, err := semver.Parse(mig.Release)
		if err != nil {
			return sdk.WrapError(err, "Your migration %s with release %s is not semver compatible", mig.Name, mig.Release)
		}
		mig.Major = v.Major
		mig.Minor = v.Minor
		mig.Patch = v.Patch
	}
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

// SaveAllMigrations save all local migrations marked to "done" into database (in case of a fresh installation)
func SaveAllMigrations(db gorp.SqlExecutor) error {
	for _, migration := range migrations {
		migration.Done = time.Now()
		migration.Status = sdk.MigrationStatusDone
		migration.Progress = "Done because it was a fresh installation"
		if err := Insert(db, &migration); err != nil {
			return err
		}
	}
	return nil
}
