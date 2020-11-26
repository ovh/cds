package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// DBConnectionFactory is a database connection factory on postgres with gorp
type DBConnectionFactory struct {
	DBDriver         string
	DBRole           string
	DBUser           string
	DBPassword       string
	DBName           string
	DBSchema         string
	DBHost           string
	DBPort           int
	DBSSLMode        string
	DBTimeout        int
	DBConnectTimeout int
	DBMaxConn        int
	Database         *sql.DB
	mutex            *sync.Mutex
}

// DB returns the current sql.DB object
func (f *DBConnectionFactory) DB() *sql.DB {
	if f.Database == nil {
		if f.DBName == "" {
			return nil
		}
		newF, err := Init(context.TODO(), f.DBUser, f.DBRole, f.DBPassword, f.DBName, f.DBSchema, f.DBHost, f.DBPort, f.DBSSLMode, f.DBConnectTimeout, f.DBTimeout, f.DBMaxConn)
		if err != nil {
			err = sdk.WrapError(err, "cannot init db connection")
			log.ErrorWithFields(context.TODO(), log.Fields{
				"stack_trace": fmt.Sprintf("%+v", err),
			}, "%s", err)
			return nil
		}
		*f = *newF
	}
	if err := f.Database.Ping(); err != nil {
		log.Error(context.TODO(), "Database> cannot ping db : %s", err)
		f.Database = nil
		return nil
	}
	return f.Database
}

// GetDBMap returns a gorp.DbMap pointer
func (f *DBConnectionFactory) GetDBMap(m *gorpmapper.Mapper) func() *gorp.DbMap {
	return func() *gorp.DbMap {
		return DBMap(m, f.DB())
	}
}

//Set is for tetsing purpose, we need to set manually the connection
func (f *DBConnectionFactory) Set(d *sql.DB) {
	f.Database = d
}

// Init initialize sql.DB object by checking environment variables and connecting to database
func Init(ctx context.Context, user, role, password, name, schema, host string, port int, sslmode string, connectTimeout, timeout, maxconn int) (*DBConnectionFactory, error) {
	if schema == "" {
		schema = "public"
	}

	f := &DBConnectionFactory{
		DBDriver:         "postgres",
		DBRole:           role,
		DBUser:           user,
		DBPassword:       password,
		DBName:           name,
		DBSchema:         schema,
		DBHost:           host,
		DBPort:           port,
		DBSSLMode:        sslmode,
		DBTimeout:        timeout,
		DBConnectTimeout: connectTimeout,
		DBMaxConn:        maxconn,
		mutex:            &sync.Mutex{},
	}

	f.mutex.Lock()
	defer f.mutex.Unlock()

	// Try to close before reinit
	if f.Database != nil {
		if err := f.Database.Close(); err != nil {
			log.Error(ctx, "Cannot close connection to DB : %s", err)
		}
	}

	var err error

	if f.DBUser == "" ||
		f.DBPassword == "" ||
		f.DBName == "" ||
		f.DBHost == "" ||
		f.DBPort == 0 {
		return nil, fmt.Errorf("Missing database infos")
	}

	if f.DBTimeout < 200 || f.DBTimeout > 30000 {
		f.DBTimeout = 3000
	}

	if f.DBConnectTimeout <= 0 {
		f.DBConnectTimeout = 10
	}

	// connect_timeout in seconds
	// statement_timeout in milliseconds
	dsn := f.dsn()
	f.Database, err = sql.Open(f.DBDriver, dsn)
	if err != nil {
		f.Database = nil
		log.Error(ctx, "cannot open database: %s", err)
		return nil, err
	}

	if err = f.Database.Ping(); err != nil {
		f.Database = nil
		return nil, err
	}

	f.Database.SetMaxOpenConns(f.DBMaxConn)
	f.Database.SetMaxIdleConns(int(f.DBMaxConn / 2))

	if _, err := f.Database.Exec(fmt.Sprintf("SET statement_timeout = %d", f.DBTimeout)); err != nil {
		log.Error(ctx, "unable to set statement_timeout with %d on database: %s", f.DBTimeout, err)
		return nil, sdk.WrapError(err, "unable to set statement_timeout with %d", f.DBTimeout)
	}

	// Set role if specified
	if role != "" {
		log.Debug("database> setting role %s on database", role)
		if _, err := f.Database.Exec("SET ROLE '" + role + "'"); err != nil {
			log.Error(ctx, "unable to set role %s on database: %v", role, err)
			return nil, sdk.WrapError(err, "unable to set role %s", role)
		}
	}

	return f, nil
}

func (f *DBConnectionFactory) dsn() string {
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%d sslmode=%s connect_timeout=%d", f.DBUser, f.DBPassword, f.DBName, f.DBHost, f.DBPort, f.DBSSLMode, f.DBConnectTimeout)
	if f.DBSchema != "public" {
		dsn += fmt.Sprintf(" search_path=%s", f.DBSchema)
	}
	return dsn
}

// Status returns database driver and status in a printable string
func (f *DBConnectionFactory) Status(ctx context.Context) sdk.MonitoringStatusLine {
	if f.Database == nil {
		return sdk.MonitoringStatusLine{Component: "Database Conns", Value: "No Connection", Status: sdk.MonitoringStatusAlert}
	}

	if err := f.Database.Ping(); err != nil {
		return sdk.MonitoringStatusLine{Component: "Database Conns", Value: "No Ping", Status: sdk.MonitoringStatusAlert}
	}

	return sdk.MonitoringStatusLine{Component: "Database Conns", Value: fmt.Sprintf("%d", f.Database.Stats().OpenConnections), Status: sdk.MonitoringStatusOK}
}

// Close closes the database, releasing any open resources.
func (f *DBConnectionFactory) Close() error {
	if f.Database != nil {
		return f.Database.Close()
	}
	return nil
}

// NewListener creates a new database connection dedicated to LISTEN / NOTIFY.
func (f *DBConnectionFactory) NewListener(minReconnectInterval time.Duration, maxReconnectInterval time.Duration, eventCallback pq.EventCallbackType) *pq.Listener {
	return pq.NewListener(f.dsn(), minReconnectInterval, maxReconnectInterval, eventCallback)
}
