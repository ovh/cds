package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

// DBConnectionFactory is a database connection factory on postgres with gorp
type DBConnectionFactory struct {
	DBRole            string
	DBUser            string
	DBPassword        string
	DBName            string
	DBSchema          string
	DBHost            string
	DBPort            int
	DBSSLMode         string
	DBTimeout         int
	DBConnectTimeout  int
	DBConnMaxIdleTime string
	DBConnMaxLifetime string
	DBMaxConn         int
	Database          *sql.DB
	mutex             *sync.Mutex
}

// DB returns the current sql.DB object
func (f *DBConnectionFactory) DB() *sql.DB {
	if f.Database == nil {
		if f.DBName == "" {
			return nil
		}
		newF, err := Init(context.TODO(), DBConfiguration{
			User:            f.DBUser,
			Role:            f.DBRole,
			Password:        f.DBPassword,
			Name:            f.DBName,
			Schema:          f.DBSchema,
			Host:            f.DBHost,
			Port:            f.DBPort,
			SSLMode:         f.DBSSLMode,
			MaxConn:         f.DBMaxConn,
			ConnectTimeout:  f.DBConnectTimeout,
			ConnMaxIdleTime: f.DBConnMaxIdleTime,
			ConnMaxLifetime: f.DBConnMaxLifetime,
			Timeout:         f.DBTimeout,
		})
		if err != nil {
			err = sdk.WithStack(err)
			ctx := sdk.ContextWithStacktrace(context.TODO(), err)
			log.Error(ctx, "unable to init db connection: %v", err)
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
func Init(ctx context.Context, dbConfig DBConfiguration) (*DBConnectionFactory, error) {
	if dbConfig.Schema == "" {
		dbConfig.Schema = "public"
	}

	f := &DBConnectionFactory{
		DBRole:            dbConfig.Role,
		DBUser:            dbConfig.User,
		DBPassword:        dbConfig.Password,
		DBName:            dbConfig.Name,
		DBSchema:          dbConfig.Schema,
		DBHost:            dbConfig.Host,
		DBPort:            dbConfig.Port,
		DBSSLMode:         dbConfig.SSLMode,
		DBTimeout:         dbConfig.Timeout,
		DBConnectTimeout:  dbConfig.ConnectTimeout,
		DBConnMaxIdleTime: dbConfig.ConnMaxIdleTime,
		DBConnMaxLifetime: dbConfig.ConnMaxLifetime,
		DBMaxConn:         dbConfig.MaxConn,
		mutex:             &sync.Mutex{},
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
		return nil, fmt.Errorf("missing database infos")
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
	connector, err := pq.NewConnector(dsn)
	if err != nil {
		log.Error(ctx, "cannot open database: %s", err)
		return nil, sdk.WithStack(err)
	}
	f.Database = sql.OpenDB(connector)

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
	if f.DBConnMaxIdleTime != "" {
		connMaxIdleTime, err := time.ParseDuration(f.DBConnMaxIdleTime)
		if err != nil {
			return nil, sdk.WrapError(err, "unable to parse connMaxIdleTime with %s on database.", f.DBConnMaxIdleTime)
		}
		f.Database.SetConnMaxIdleTime(connMaxIdleTime)
	}
	if f.DBConnMaxLifetime != "" {
		connMaxLifetime, err := time.ParseDuration(f.DBConnMaxLifetime)
		if err != nil {
			return nil, sdk.WrapError(err, "unable to parse connMaxLifetime with %s on database.", f.DBConnMaxIdleTime)
		}
		f.Database.SetConnMaxLifetime(connMaxLifetime)
	}

	if _, err := f.Database.Exec(fmt.Sprintf("SET statement_timeout = %d", f.DBTimeout)); err != nil {
		log.Error(ctx, "unable to set statement_timeout with %d on database: %s", f.DBTimeout, err)
		return nil, sdk.WrapError(err, "unable to set statement_timeout with %d", f.DBTimeout)
	}

	// Set role if specified
	if f.DBRole != "" {
		log.Debug(ctx, "database> setting role %s on database", f.DBRole)
		if _, err := f.Database.Exec("SET ROLE '" + f.DBRole + "'"); err != nil {
			log.Error(ctx, "unable to set role %s on database: %v", f.DBRole, err)
			return nil, sdk.WrapError(err, "unable to set role %s", f.DBRole)
		}
	}

	return f, nil
}

func (f *DBConnectionFactory) dsn() string {
	dsn := fmt.Sprintf("user=%s password='%s' dbname=%s host=%s port=%d sslmode=%s connect_timeout=%d", f.DBUser, f.DBPassword, f.DBName, f.DBHost, f.DBPort, f.DBSSLMode, f.DBConnectTimeout)
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
