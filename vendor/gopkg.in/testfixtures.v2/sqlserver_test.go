// +build sqlserver

package testfixtures

import (
	_ "github.com/denisenkom/go-mssqldb"
)

func init() {
	databases = append(databases,
		databaseTest{
			"mssql",
			"SQLSERVER_CONN_STRING",
			"testdata/schema/sqlserver.sql",
			&SQLServer{},
		},
	)
}
