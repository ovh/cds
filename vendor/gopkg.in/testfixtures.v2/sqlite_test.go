// +build sqlite

package testfixtures

import (
	_ "github.com/mattn/go-sqlite3"
)

func init() {
	databases = append(databases, databaseTest{
		"sqlite3",
		"SQLITE_CONN_STRING",
		"testdata/schema/sqlite.sql",
		&SQLite{},
	})
}
