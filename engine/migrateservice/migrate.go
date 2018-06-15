package migrateservice

import (
	"fmt"

	"github.com/ovh/cds/sdk"

	"github.com/rubenv/sql-migrate"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/database/dbmigrate"
)

func (s *dbmigservice) doMigrate() error {
	dbConn, err := database.Init(
		s.cfg.DB.User,
		s.cfg.DB.Password,
		s.cfg.DB.Name,
		s.cfg.DB.Host,
		s.cfg.DB.Port,
		s.cfg.DB.SSLMode,
		s.cfg.DB.ConnectTimeout,
		s.cfg.DB.Timeout,
		s.cfg.DB.MaxConn)
	if err != nil {
		return sdk.WrapError(fmt.Errorf("cannot connect to database: %v", err), "doMigrate")
	}

	if _, err := dbmigrate.Do(dbConn.DB, s.cfg.Directory, migrate.Up, false, -1); err != nil {
		return sdk.WrapError(err, "doMigrate")
	}

	return nil
}

func (s *dbmigservice) getMigrate() ([]dbmigrate.MigrationStatus, error) {
	dbConn, err := database.Init(
		s.cfg.DB.User,
		s.cfg.DB.Password,
		s.cfg.DB.Name,
		s.cfg.DB.Host,
		s.cfg.DB.Port,
		s.cfg.DB.SSLMode,
		s.cfg.DB.ConnectTimeout,
		s.cfg.DB.Timeout,
		s.cfg.DB.MaxConn)
	if err != nil {
		return nil, sdk.WrapError(fmt.Errorf("cannot connect to database: %v", err), "getMigrate")
	}

	return dbmigrate.Get(dbConn.DB, s.cfg.Directory)
}
