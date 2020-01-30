package migrate

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/blang/semver"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// MinCompatibleRelease represent the minimum release which is working with these migrations, need to update when we delete migration in our codebase
const MinCompatibleRelease = "0.41.0"

var migrations = []sdk.Migration{}

// Add adds a migration to the list of migrations to run at API start time
// example of usage:
// migrate.Add(sdk.Migration{Name: "MyMigration", Release: "0.39.3", Mandatory: true, ExecFunc: func(ctx context.Context) error {
//	return migrate.MyMigration(ctx, a.Cache, a.DBConnectionFactory.GetDBMap)
// }})
func Add(ctx context.Context, migration sdk.Migration) {
	if migration.Major == 0 && migration.Minor == 0 && migration.Patch == 0 && migration.Release != "" && !strings.HasPrefix(migration.Release, "snapshot") {
		v, err := semver.Parse(migration.Release)
		if err != nil {
			log.Error(ctx, "Cannot parse your release reference : %v", err)
		}
		migration.Major = v.Major
		migration.Minor = v.Minor
		migration.Patch = v.Patch
	}
	migrations = append(migrations, migration)
}

// Run run all local migrations
func Run(ctx context.Context, db gorp.SqlExecutor, panicDump func(s string) (io.WriteCloser, error)) {
	var wg = new(sync.WaitGroup)
	for _, migration := range migrations {
		func(currentMigration sdk.Migration) {
			if currentMigration.Blocker {
				wg.Add(1)
			}

			sdk.GoRoutine(ctx, "migrate_"+currentMigration.Name, func(contex context.Context) {
				defer func() {
					if currentMigration.Blocker {
						wg.Done()
					}
				}()
				mig, errMig := GetByName(db, currentMigration.Name)
				if errMig != nil {
					log.Error(ctx, "Cannot get migration %s : %v", currentMigration.Name, errMig)
					return
				}
				if mig != nil {
					if mig.Status == sdk.MigrationStatusDone || mig.Status == sdk.MigrationStatusCanceled || mig.Status == sdk.MigrationStatusNotExecuted {
						log.Info(ctx, "Migration> %s> Already done (status: %s)", currentMigration.Name, mig.Status)
						return
					}

					// set the previous migration id for for the case where the migration was reset
					currentMigration.ID = mig.ID
					currentMigration.Status = sdk.MigrationStatusInProgress
				} else {
					if !currentMigration.Automatic {
						currentMigration.Status = sdk.MigrationStatusNotExecuted
					} else {
						currentMigration.Status = sdk.MigrationStatusInProgress
						currentMigration.Progress = "Begin"
					}
					if err := Insert(db, &currentMigration); err != nil {
						log.Error(ctx, "Cannot insert migration %s : %v", currentMigration.Name, err)
						return
					}
				}

				if currentMigration.Status != sdk.MigrationStatusInProgress {
					return
				}

				log.Info(ctx, "Migration [%s]: begin", currentMigration.Name)
				if err := currentMigration.ExecFunc(contex); err != nil {
					log.Error(ctx, "migration %s in ERROR : %v", currentMigration.Name, err)
					currentMigration.Error = err.Error()
				}
				currentMigration.Progress = "Migration done"
				currentMigration.Done = time.Now()
				currentMigration.Status = sdk.MigrationStatusDone

				if err := Update(db, &currentMigration); err != nil {
					log.Error(ctx, "Cannot update migration %s : %v", currentMigration.Name, err)
				}
				log.Info(ctx, "Migration [%s]: Done", currentMigration.Name)
			}, panicDump)
		}(migration)
	}
	wg.Wait()
}

// CleanMigrationsList Delete all elements in local migrations
func CleanMigrationsList() {
	migrations = []sdk.Migration{}
}

// Status returns monitoring status, if there are cds migration in progress it returns WARN
func Status(db gorp.SqlExecutor) sdk.MonitoringStatusLine {
	count, err := db.SelectInt("SELECT COUNT(id) FROM cds_migration WHERE status <> $1 AND status <> $2 AND status <> $3",
		sdk.MigrationStatusDone, sdk.MigrationStatusCanceled, sdk.MigrationStatusNotExecuted)
	if err != nil {
		return sdk.MonitoringStatusLine{Component: "CDS Migration", Status: sdk.MonitoringStatusWarn, Value: fmt.Sprintf("KO Cannot request in database : %v", err)}
	}
	status := sdk.MonitoringStatusOK
	if count > 0 {
		status = sdk.MonitoringStatusWarn
	}
	return sdk.MonitoringStatusLine{Component: "Nb of CDS Migrations in progress", Value: fmt.Sprintf("%d", count), Status: status}
}
