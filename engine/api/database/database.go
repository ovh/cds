package database

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/go-gorp/gorp"
	_ "github.com/lib/pq"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// DBConnectionFactory is a database connection factory on postgres with gorp
type DBConnectionFactory struct {
	dbDriver         string
	dbRole           string
	dbUser           string
	dbPassword       string
	dbName           string
	dbHost           string
	dbPort           int
	dbSSLMode        string
	dbTimeout        int
	dbConnectTimeout int
	dbMaxConn        int
	db               *sql.DB
	mutex            *sync.Mutex
}

// DB returns the current sql.DB object
func (f *DBConnectionFactory) DB() *sql.DB {
	if f.db == nil {
		if f.dbName == "" {
			return nil
		}
		newF, err := Init(f.dbUser, f.dbRole, f.dbPassword, f.dbName, f.dbHost, f.dbPort, f.dbSSLMode, f.dbConnectTimeout, f.dbTimeout, f.dbMaxConn)
		if err != nil {
			log.Error("Database> cannot init db connection : %s", err)
			return nil
		}
		*f = *newF
	}
	if err := f.db.Ping(); err != nil {
		log.Error("Database> cannot ping db : %s", err)
		f.db = nil
		return nil
	}
	return f.db
}

// GetDBMap returns a gorp.DbMap pointer
func (f *DBConnectionFactory) GetDBMap() *gorp.DbMap {
	return DBMap(f.DB())
}

//Set is for tetsing purpose, we need to set manually the connection
func (f *DBConnectionFactory) Set(d *sql.DB) {
	f.db = d
}

// Init initialize sql.DB object by checking environment variables and connecting to database
func Init(user, role, password, name, host string, port int, sslmode string, connectTimeout, timeout, maxconn int) (*DBConnectionFactory, error) {
	f := &DBConnectionFactory{
		dbDriver:         "postgres",
		dbRole:           role,
		dbUser:           user,
		dbPassword:       password,
		dbName:           name,
		dbHost:           host,
		dbPort:           port,
		dbSSLMode:        sslmode,
		dbTimeout:        timeout,
		dbConnectTimeout: connectTimeout,
		dbMaxConn:        maxconn,
		mutex:            &sync.Mutex{},
	}

	f.mutex.Lock()
	defer f.mutex.Unlock()

	// Try to close before reinit
	if f.db != nil {
		if err := f.db.Close(); err != nil {
			log.Error("Cannot close connection to DB : %s", err)
		}
	}

	var err error

	if f.dbUser == "" ||
		f.dbPassword == "" ||
		f.dbName == "" ||
		f.dbHost == "" ||
		f.dbPort == 0 {
		return nil, fmt.Errorf("Missing database infos")
	}

	if f.dbTimeout < 200 || f.dbTimeout > 30000 {
		f.dbTimeout = 3000
	}

	if f.dbConnectTimeout <= 0 {
		f.dbConnectTimeout = 10
	}

	// connect_timeout in seconds
	// statement_timeout in milliseconds
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%d sslmode=%s connect_timeout=%d statement_timeout=%d", f.dbUser, f.dbPassword, f.dbName, f.dbHost, f.dbPort, f.dbSSLMode, f.dbConnectTimeout, f.dbTimeout)
	f.db, err = sql.Open(f.dbDriver, dsn)
	if err != nil {
		f.db = nil
		log.Error("cannot open database: %s", err)
		return nil, err
	}

	if err = f.db.Ping(); err != nil {
		f.db = nil
		return nil, err
	}

	f.db.SetMaxOpenConns(f.dbMaxConn)
	f.db.SetMaxIdleConns(int(f.dbMaxConn / 2))

	// Set role if specified
	if role != "" {
		log.Debug("database> setting role %s on database", role)
		if _, err := f.db.Exec("SET ROLE '" + role + "'"); err != nil {
			log.Error("unable to set role %s on database: %s", role, err)
			return nil, sdk.WrapError(err, "unable to set role %s", role)
		}
	}

	return f, nil
}

// Status returns database driver and status in a printable string
func (f *DBConnectionFactory) Status() sdk.MonitoringStatusLine {
	if f.db == nil {
		return sdk.MonitoringStatusLine{Component: "Database", Value: "No Connection", Status: sdk.MonitoringStatusAlert}
	}

	if err := f.db.Ping(); err != nil {
		return sdk.MonitoringStatusLine{Component: "Database", Value: "No Ping", Status: sdk.MonitoringStatusAlert}
	}

	return sdk.MonitoringStatusLine{Component: "Database", Value: fmt.Sprintf("%d conns", f.db.Stats().OpenConnections), Status: sdk.MonitoringStatusOK}
}

// Close closes the database, releasing any open resources.
func (f *DBConnectionFactory) Close() error {
	if f.db != nil {
		return f.db.Close()
	}
	return nil
}
