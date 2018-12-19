package migrate

import (
	"context"
	"io"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/blang/semver"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var migrations = []sdk.Migration{}

// Add usefull to add new migrations
func Add(migration sdk.Migration) {
	if migration.Major == 0 && migration.Minor == 0 && migration.Patch == 0 && migration.Release != "" && migration.Release != "snapshot" {
		v, err := semver.Parse(migration.Release)
		if err != nil {
			log.Error("Cannot parse your release reference : %v", err)
		}
		migration.Major = v.Major
		migration.Minor = v.Minor
		migration.Patch = v.Patch
	}
	migrations = append(migrations, migration)
}

// Run run all local migrations
func Run(ctx context.Context, db gorp.SqlExecutor, panicDump func(s string) (io.WriteCloser, error)) {
	for _, migration := range migrations {
		func(currentMigration sdk.Migration) {
			sdk.GoRoutine(ctx, "migrate_"+migration.Name, func(contex context.Context) {
				var mig *sdk.Migration
				var errMig error
				mig, errMig = GetByName(db, currentMigration.Name)
				if errMig != nil {
					log.Error("Cannot get migration %s : %v", currentMigration.Name, errMig)
					return
				}
				if mig != nil {
					if mig.Status == sdk.MigrationStatusDone || mig.Status == sdk.MigrationStatusCanceled {
						log.Info("Migration> %s> Already done (status: %s)", currentMigration.Name, mig.Status)
						return
					}
				} else {
					currentMigration.Progress = "Begin"
					currentMigration.Status = sdk.MigrationStatusInProgress
					if err := Insert(db, &currentMigration); err != nil {
						log.Error("Cannot insert migration %s : %v", currentMigration.Name, err)
						return
					}
				}
				if err := currentMigration.ExecFunc(contex); err != nil {
					log.Error("migration %s in ERROR : %v", currentMigration.Name, err)
					currentMigration.Error = err.Error()
				}
				currentMigration.Progress = "Migration done"
				currentMigration.Done = time.Now()
				currentMigration.Status = sdk.MigrationStatusDone

				if err := Update(db, &currentMigration); err != nil {
					log.Error("Cannot update migration %s : %v", currentMigration.Name, err)
				}
			}, panicDump)
		}(migration)
	}
}

// PreviousMigrationsList return previous migration list
func PreviousMigrationsList() []sdk.Migration {
	releaseVersion := sdk.VersionCurrent().Version
	if releaseVersion == "" || releaseVersion == "snapshot" {
		return migrations
	}
	currentSemver, err := semver.Parse(releaseVersion)
	if err != nil {
		return migrations
	}

	var previousMigrations []sdk.Migration
	for _, migration := range migrations {
		if currentSemver.Minor > migration.Minor {
			previousMigrations = append(previousMigrations, migration)
		}
	}

	return previousMigrations
}

// CleanMigrationsList Delete all elements in local migrations
func CleanMigrationsList() {
	migrations = []sdk.Migration{}
}
