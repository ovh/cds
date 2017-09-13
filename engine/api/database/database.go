package database

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/go-gorp/gorp"
	_ "github.com/lib/pq"

	"github.com/ovh/cds/sdk/log"
)

// DBConnectionFactory is a database connection factory on postgres with gorp
type DBConnectionFactory struct {
	dbDriver         string
	dbUser           string
	dbPassword       string
	dbName           string
	dbHost           string
	dbPort           int
	dbSSLMode        string
	dbTimeout        int
	dbMaxConn        int
	db               *sql.DB
	mutex            *sync.Mutex
	SecretDBUser     string
	SecretDBPassword string
}

// DB returns the current sql.DB object
func (f *DBConnectionFactory) DB() *sql.DB {
	if f.db == nil {
		if f.dbName == "" {
			return nil
		}
		newF, err := Init(f.dbUser, f.dbPassword, f.dbName, f.dbHost, f.dbPort, f.dbSSLMode, f.dbTimeout, f.dbMaxConn)
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
func Init(user, password, name, host string, port int, sslmode string, timeout, maxconn int) (*DBConnectionFactory, error) {
	f := &DBConnectionFactory{
		dbDriver:   "postgres",
		dbUser:     user,
		dbPassword: password,
		dbName:     name,
		dbHost:     host,
		dbPort:     port,
		dbSSLMode:  sslmode,
		dbTimeout:  timeout,
		dbMaxConn:  maxconn,
		mutex:      &sync.Mutex{},
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

	if f.SecretDBUser != "" {
		f.dbUser = f.SecretDBUser
	}

	if f.SecretDBPassword != "" {
		f.dbPassword = f.SecretDBPassword
	}

	if f.dbTimeout < 200 || f.dbTimeout > 15000 {
		f.dbTimeout = 3000
	}

	// connect_timeout in seconds
	// statement_timeout in milliseconds
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%d sslmode=%s connect_timeout=10 statement_timeout=%d", f.dbUser, f.dbPassword, f.dbName, f.dbHost, f.dbPort, f.dbSSLMode, f.dbTimeout)
	f.db, err = sql.Open(f.dbDriver, dsn)
	if err != nil {
		f.db = nil
		log.Error("Cannot open database: %s", err)
		return nil, err
	}

	if err = f.db.Ping(); err != nil {
		f.db = nil
		return nil, err
	}

	f.db.SetMaxOpenConns(f.dbMaxConn)
	f.db.SetMaxIdleConns(int(f.dbMaxConn / 2))

	return f, nil
}

// Status returns database driver and status in a printable string
func (f *DBConnectionFactory) Status() string {
	if f.db == nil {
		return fmt.Sprintf("Database: %s KO (no connection)", f.dbDriver)
	}

	if err := f.db.Ping(); err != nil {
		return fmt.Sprintf("Database: %s KO (%s)", f.dbDriver, err)
	}

	return fmt.Sprintf("Database: %s OK (%d conns)", f.dbDriver, f.db.Stats().OpenConnections)
}

// Close closes the database, releasing any open resources.
func (f *DBConnectionFactory) Close() error {
	if f.db != nil {
		return f.db.Close()
	}
	return nil
}
