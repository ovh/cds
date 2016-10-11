package database

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"

	"log"

	"github.com/spf13/viper"
)

var (
	dbDriver   string
	dbUser     string
	dbPassword string
	dbName     string
	dbHost     string
	dbPort     string
	dbSSLMode  string
	db         *sql.DB
	mutex      = &sync.Mutex{}
)

// QueryExecuter execute and query SQL query
type QueryExecuter interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

// Executer execute SQL query
type Executer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

// Querier execute query in database
type Querier interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

// Scanner is implemented by sql.Row and sql.Rows
type Scanner interface {
	Scan(dest ...interface{}) error
}

// DB returns the current sql.DB object
func DB() *sql.DB {
	if db == nil {
		_, err := Init()
		if err != nil {
			log.Printf("Database> cannot init db connection : %s\n", err)
			return nil
		}
	}
	if err := db.Ping(); err != nil {
		log.Printf("Database> cannot ping db : %s\n", err)
		db = nil
		return nil
	}
	return db
}

//Set is for tetsing purpose, we need to set manually the connection
func Set(d *sql.DB) {
	db = d
}

// Init initialize sql.DB object by checking environment variables and connecting to database
func Init() (*sql.DB, error) {
	mutex.Lock()
	defer mutex.Unlock()

	// Try to close before reinit
	if db != nil {
		if err := db.Close(); err != nil {
			log.Printf("[CRITICAL] cannot close connection to DB : %s", err)
		}
	}

	var err error

	dbDriver = "postgres"
	dbUser = viper.GetString("db_user")
	dbPassword = viper.GetString("db_password")
	dbName = viper.GetString("db_name")
	dbHost = viper.GetString("db_host")
	dbPort = viper.GetString("db_port")
	dbSSLMode = viper.GetString("db_sslmode")

	if dbUser == "" ||
		dbPassword == "" ||
		dbName == "" ||
		dbHost == "" ||
		dbPort == "" {
		log.Printf("Missing database infos for PostgreSQL, fallbacking on RamSQL:\n")
		log.Printf("got user=%s, password=%dchar, name=%s, host=%s, port=%s sslmode=%s\n", dbUser, len(dbPassword), dbName, dbHost, dbPort, dbSSLMode)

		dbDriver = "ramsql"
	}

	timeout := viper.GetInt("db_timeout")
	if timeout < 200 || timeout > 15000 {
		timeout = 3000
	}

	// connect_timeout in seconds
	// statement_timeout in milliseconds
	// yeah...
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=%s connect_timeout=10 statement_timeout=%d", dbUser, dbPassword, dbName, dbHost, dbPort, dbSSLMode, timeout)

	db, err = sql.Open(dbDriver, dsn)
	if err != nil {
		db = nil
		log.Printf("Cannot open database: %s\n", err)
		return nil, err
	}

	if err = db.Ping(); err != nil {
		db = nil
		return nil, err
	}

	var initSchema bool
	switch dbDriver {
	case "ramsql":
		initSchema = true
	case "postgres":
		//Check if schema is empty
		rows, err := db.Query("SELECT tablename FROM pg_catalog.pg_tables WHERE tableowner like $1", dbUser+"%")

		if err != nil {
			log.Printf(err.Error())
			return db, err
		}
		defer rows.Close()
		if !rows.Next() {
			log.Printf("Database schema has to be initialized")
			initSchema = true
		}
		if err := rows.Err(); err != nil {
			log.Printf(err.Error())
			return db, err
		}
	}

	//If driver is ramsql or PG schema is empty, need to init
	if initSchema {
		if err = InitSchemas(db); err != nil {
			log.Printf("InitSchema: %s\n", err)
			return db, err
		}
	}

	max := viper.GetInt("db_maxconn")
	if max <= 0 || max > 100 {
		max = 20
	}
	log.Printf("Database> Setting MaxOpenConns to %d\n", max)
	db.SetMaxOpenConns(max)
	db.SetMaxIdleConns(int(max / 2))

	return db, nil
}

// Status returns database driver and status in a printable string
func Status() string {
	if db == nil {
		return fmt.Sprintf("Database: %s KO (no connection)", dbDriver)
	}
	err := db.Ping()
	if err != nil {
		return fmt.Sprintf("Database: %s KO (%s)", dbDriver, err)
	}

	return fmt.Sprintf("Database: %s OK (%d conns)", dbDriver, db.Stats().OpenConnections)
}

// InitSchemas checks that all tables are correct, and create them if not
func InitSchemas(sqlDB *sql.DB) error {
	sqlfile := path.Join(os.Getenv("GOPATH"), "src", "github.com/ovh", "cds", "engine", "sql", "create_table.sql")
	sqlcontent, err := ioutil.ReadFile(sqlfile)
	if err != nil {
		return err
	}

	queries := strings.Split(string(sqlcontent), ";")

	for _, q := range queries {
		q = strings.Trim(q, " \n")
		if q == "" {
			continue
		}
		_, err := sqlDB.Exec(q)
		if err != nil {
			return err
		}
	}
	return nil
}
