package database

import "database/sql"

// QueryExecuter executes and queries SQL query
type QueryExecuter interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

// Executer executes SQL query
type Executer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

// Querier executes query in database
type Querier interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

// Scanner is implemented by sql.Row and sql.Rows
type Scanner interface {
	Scan(dest ...interface{}) error
}
