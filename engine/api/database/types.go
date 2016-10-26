package database

import "database/sql"

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
