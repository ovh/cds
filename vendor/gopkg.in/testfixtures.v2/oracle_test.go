// +build oracle

package testfixtures

import (
	_ "github.com/mattn/go-oci8"
)

func init() {
	databases = append(databases,
		databaseTest{
			"oci8",
			"ORACLE_CONN_STRING",
			"testdata/schema/oracle.sql",
			&Oracle{},
		},
	)
}
