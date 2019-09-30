package migrateservice

import (
	"database/sql"
	"fmt"

	"github.com/go-gorp/gorp"
	migrate "github.com/rubenv/sql-migrate"

	"github.com/ovh/cds/engine/api/database/dbmigrate"
	"github.com/ovh/cds/sdk"
)

func (s *dbmigservice) doMigrate(db func() *sql.DB, dialect gorp.Dialect, upgradeTo, downgradeTo string) error {
	status, err := dbmigrate.Get(db, s.cfg.Directory, dialect)
	if err != nil {
		return sdk.WithStack(err)
	}

	if upgradeTo != "" && downgradeTo != "" {
		return sdk.WithStack(fmt.Errorf("invalid migration configuration"))
	}

	if upgradeTo == "" && downgradeTo == "" {
		if _, err := dbmigrate.Do(db, dialect, s.cfg.Directory, migrate.Up, false, -1); err != nil {
			return sdk.WrapError(err, "doMigrate")
		}
		return nil
	}

	if upgradeTo != "" {
		var idxToUpgrade = -1
		var lastIdxApplied = 0
		for i, s := range status {
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
			return sdk.WithStack(fmt.Errorf("invalid migration configuration - %s not found", upgradeTo))
		}
		if _, err := dbmigrate.Do(db, dialect, s.cfg.Directory, migrate.Up, false, idxToUpgrade-lastIdxApplied); err != nil {
			return sdk.WrapError(err, "doMigrate")
		}
		return nil
	}

	if downgradeTo != "" {
		var idxToDowngrade = -1
		var lastIdxApplied = 0
		for i, s := range status {
			if s.Migrated {
				lastIdxApplied = i
			}
			if s.ID == downgradeTo && s.Migrated {
				idxToDowngrade = i
			}
		}
		if lastIdxApplied == 0 {
			return sdk.WithStack(fmt.Errorf("nothing to perform"))
		}
		if idxToDowngrade == -1 {
			return sdk.WithStack(fmt.Errorf("invalid migration configuration - %s not found", downgradeTo))
		}
		if _, err := dbmigrate.Do(db, dialect, s.cfg.Directory, migrate.Down, false, lastIdxApplied-idxToDowngrade+1); err != nil {
			return sdk.WrapError(err, "doMigrate")
		}
		return nil
	}

	return nil
}

func (s *dbmigservice) getMigrate(db func() *sql.DB, dialect gorp.Dialect) ([]sdk.DatabaseMigrationStatus, error) {
	return dbmigrate.Get(db, s.cfg.Directory, dialect)
}
