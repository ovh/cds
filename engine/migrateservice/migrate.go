package migrateservice

import (
	"fmt"

	"github.com/rubenv/sql-migrate"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/database/dbmigrate"
	"github.com/ovh/cds/sdk/log"
)

func (s *dbmigservice) doMigrate() error {
	log.Info("%+v", s.cfg.DB)
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
		return fmt.Errorf("cannot connect to database: %v", err)
	}

	if _, err := dbmigrate.Do(dbConn.DB, s.cfg.Directory, migrate.Up, false, -1); err != nil {
		return err
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
		return nil, fmt.Errorf("cannot connect to database: %v", err)
	}

	return dbmigrate.Get(dbConn.DB, s.cfg.Directory)
}
