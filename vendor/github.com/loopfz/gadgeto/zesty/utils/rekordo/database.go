package rekordo

import (
	"database/sql"
	"errors"

	"github.com/go-gorp/gorp"
	"github.com/loopfz/gadgeto/zesty"
)

// Default database settings.
const (
	maxOpenConns = 5
	maxIdleConns = 3
)

// DatabaseConfig represents the configuration used to
// register a new database.
type DatabaseConfig struct {
	Name             string
	DSN              string
	System           DBMS
	MaxOpenConns     int
	MaxIdleConns     int
	AutoCreateTables bool
}

// RegisterDatabase creates a gorp map with tables and tc and
// registers it with zesty.
func RegisterDatabase(dbcfg *DatabaseConfig, tc gorp.TypeConverter) (zesty.DB, error) {
	dbConn, err := sql.Open(dbcfg.System.DriverName(), dbcfg.DSN)
	if err != nil {
		return nil, err
	}
	// Make sure we have proper values for the database
	// settings, and replace them with default if necessary
	// before applying to the new connection.
	if dbcfg.MaxOpenConns == 0 {
		dbcfg.MaxOpenConns = maxOpenConns
	}
	dbConn.SetMaxOpenConns(dbcfg.MaxOpenConns)
	if dbcfg.MaxIdleConns == 0 {
		dbcfg.MaxIdleConns = maxIdleConns
	}
	dbConn.SetMaxIdleConns(dbcfg.MaxIdleConns)

	// Select the proper dialect used by gorp.
	var dialect gorp.Dialect
	switch dbcfg.System {
	case DatabaseMySQL:
		dialect = gorp.MySQLDialect{}
	case DatabasePostgreSQL:
		dialect = gorp.PostgresDialect{}
	case DatabaseSqlite3:
		dialect = gorp.SqliteDialect{}
	default:
		return nil, errors.New("unknown database system")
	}
	dbmap := &gorp.DbMap{
		Db:            dbConn,
		Dialect:       dialect,
		TypeConverter: tc,
	}
	modelsMu.Lock()
	tableModels := models[dbcfg.Name]
	for _, t := range tableModels {
		dbmap.AddTableWithName(t.Model, t.Name).SetKeys(t.AutoIncrement, t.Keys...)
	}
	modelsMu.Unlock()

	if dbcfg.AutoCreateTables {
		err = dbmap.CreateTablesIfNotExists()
		if err != nil {
			return nil, err
		}
	}
	db := zesty.NewDB(dbmap)
	if err := zesty.RegisterDB(db, dbcfg.Name); err != nil {
		return nil, err
	}
	return db, nil
}

// DBMS represents a database management system.
type DBMS uint8

// Database management systems.
const (
	DatabasePostgreSQL DBMS = iota ^ 42
	DatabaseMySQL
	DatabaseSqlite3
)

// DriverName returns the name of the driver for ds.
func (d DBMS) DriverName() string {
	switch d {
	case DatabasePostgreSQL:
		return "postgres"
	case DatabaseMySQL:
		return "mysql"
	case DatabaseSqlite3:
		return "sqlite3"
	}
	return ""
}
