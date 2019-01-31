package migrate

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/blang/semver"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// MinCompatibleRelease represent the minimum release which is working with these migrations, need to update when we delete migration in our codebase
const MinCompatibleRelease = "0.36.1"

var migrations = []sdk.Migration{}

// Add usefull to add new migrations
func Add(migration sdk.Migration) {
	if migration.Major == 0 && migration.Minor == 0 && migration.Patch == 0 && migration.Release != "" && !strings.HasPrefix(migration.Release, "snapshot") {
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
				log.Info("Migration [%s]: begin", currentMigration.Name)
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
				log.Info("Migration [%s]: Done", currentMigration.Name)
			}, panicDump)
		}(migration)
	}
}

// CleanMigrationsList Delete all elements in local migrations
func CleanMigrationsList() {
	migrations = []sdk.Migration{}
}

// Status returns monitoring status, if there are cds migration in progress it returns WARN
func Status(db gorp.SqlExecutor) sdk.MonitoringStatusLine {
	count, err := db.SelectInt("SELECT COUNT(id) FROM cds_migration WHERE status <> $1 AND status <> $2", sdk.MigrationStatusDone, sdk.MigrationStatusCanceled)
	if err != nil {
		return sdk.MonitoringStatusLine{Component: "CDS Migration", Status: sdk.MonitoringStatusWarn, Value: fmt.Sprintf("KO Cannot request in database : %v", err)}
	}
	status := sdk.MonitoringStatusOK
	if count > 0 {
		status = sdk.MonitoringStatusWarn
	}
	return sdk.MonitoringStatusLine{Component: "Nb of CDS Migrations in progress", Value: fmt.Sprintf("%d", count), Status: status}
}
