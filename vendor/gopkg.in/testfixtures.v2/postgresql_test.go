// +build postgresql

package testfixtures

import (
	_ "github.com/lib/pq"
)

func init() {
	databases = append(databases,
		databaseTest{
			"postgres",
			"PG_CONN_STRING",
			"testdata/schema/postgresql.sql",
			&PostgreSQL{},
		},
		databaseTest{
			"postgres",
			"PG_CONN_STRING",
			"testdata/schema/postgresql.sql",
			&PostgreSQL{UseAlterConstraint: true},
		},
	)
}
