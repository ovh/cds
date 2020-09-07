package migrateservice

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/go-gorp/gorp"
	migrate "github.com/rubenv/sql-migrate"

	"github.com/ovh/cds/engine/api/database/dbmigrate"
	"github.com/ovh/cds/engine/database"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func doMigrateAll(ctx context.Context, cfg Configuration) ([]sdk.DatabaseMigrationStatus, error) {
	var mErr sdk.MultiError
	var globalStatus []sdk.DatabaseMigrationStatus

	if !cfg.ServiceAPI.Enable && !cfg.ServiceCDN.Enable {
		return nil, sdk.WithStack(fmt.Errorf("invalid migration configuration, no service configured"))
	}

	// Set *_UPGRADE_TO to define the maximum migration file you want to upgrade for a service
	// Set *_DOWNGRADE_TO to define the maximum migration file you want to downgrade for a service

	if cfg.ServiceAPI.Enable {
		status, err := doMigrate(ctx, cfg.ServiceAPI.DB, cfg.Directory+"/api", os.Getenv("API_UPGRADE_TO"), os.Getenv("API_DOWNGRADE_TO"))
		if err != nil {
			mErr.Append(err)
		}
		for i := range status {
			status[i].Database = cfg.ServiceAPI.DB.Name
		}
		globalStatus = append(globalStatus, status...)
	}

	if cfg.ServiceCDN.Enable {
		status, err := doMigrate(ctx, cfg.ServiceCDN.DB, cfg.Directory+"/cdn", os.Getenv("CDN_UPGRADE_TO"), os.Getenv("CDN_DOWNGRADE_TO"))
		if err != nil {
			mErr.Append(err)
		}
		for i := range status {
			status[i].Database = cfg.ServiceCDN.DB.Name
		}
		globalStatus = append(globalStatus, status...)
	}

	if !mErr.IsEmpty() {
		return nil, &mErr
	}
	return globalStatus, nil
}

func doMigrate(ctx context.Context, dbConfig database.DBConfiguration, directory, upgradeTo, downgradeTo string) ([]sdk.DatabaseMigrationStatus, error) {
	if upgradeTo != "" && downgradeTo != "" {
		return nil, sdk.WithStack(fmt.Errorf("invalid migration configuration, UPGRADE_TO and DOWNGRADE_TO can't be used together"))
	}

	log.Info(ctx, "DBMigrate> Starting Database migration...")
	dbConn, err := database.Init(
		ctx,
		dbConfig.User,
		dbConfig.Role,
		dbConfig.Password,
		dbConfig.Name,
		dbConfig.Schema,
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.SSLMode,
		dbConfig.ConnectTimeout,
		dbConfig.Timeout,
		dbConfig.MaxConn)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot connect to database with name %s", dbConfig.Name)
	}

	return execMigrate(ctx, dbConn.DB, gorp.PostgresDialect{}, directory, upgradeTo, downgradeTo)
}

func execMigrate(ctx context.Context, db func() *sql.DB, dialect gorp.Dialect, directory, upgradeTo, downgradeTo string) ([]sdk.DatabaseMigrationStatus, error) {
	statusBefore, err := dbmigrate.Get(db, directory, dialect)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	if upgradeTo == "" && downgradeTo == "" {
		if _, err := dbmigrate.Do(db, dialect, directory, migrate.Up, false, -1); err != nil {
			return nil, sdk.WithStack(err)
		}
	} else if upgradeTo != "" {
		var idxToUpgrade, lastIdxApplied = -1, 0
		for i, s := range statusBefore {
			if s.Migrated {
				lastIdxApplied = i
				continue
			}
			if s.ID == upgradeTo && !s.Migrated {
				idxToUpgrade = i
				break
			}
		}
		if idxToUpgrade == -1 {
			return nil, sdk.WithStack(fmt.Errorf("invalid migration configuration %s not found in %s", upgradeTo, directory))
		}
		if _, err := dbmigrate.Do(db, dialect, directory, migrate.Up, false, idxToUpgrade-lastIdxApplied); err != nil {
			return nil, sdk.WithStack(err)
		}
	} else if downgradeTo != "" {
		var idxToDowngrade, lastIdxApplied = -1, 0
		for i, s := range statusBefore {
			if s.Migrated {
				lastIdxApplied = i
			}
			if s.ID == downgradeTo && s.Migrated {
				idxToDowngrade = i
			}
		}
		if lastIdxApplied == 0 {
			return nil, sdk.WithStack(fmt.Errorf("nothing to perform"))
		}
		if idxToDowngrade == -1 {
			return nil, sdk.WithStack(fmt.Errorf("invalid migration configuration %s not found in %s", downgradeTo, directory))
		}
		if _, err := dbmigrate.Do(db, dialect, directory, migrate.Down, false, lastIdxApplied-idxToDowngrade+1); err != nil {
			return nil, sdk.WithStack(err)
		}
	}

	log.Info(ctx, "DBMigrate> Retrieving migration status for database according directory %s...", directory)
	statusAfter, err := dbmigrate.Get(db, directory, dialect)
	if err != nil {
		return nil, sdk.WithStack(err)
	}

	return statusAfter, nil
}
