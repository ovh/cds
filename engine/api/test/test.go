package test

import (
	"database/sql"
	"testing"

	// ramsql
	_ "github.com/proullon/ramsql/driver"

	"github.com/ovh/cds/engine/api/database"
)

// Setup setup db for test
func Setup(testname string, t *testing.T) *sql.DB {

	db, err := sql.Open("ramsql", testname)
	if err != nil {
		t.Fatalf("Cannot open conn to ramsql: %s\n", err)
	}

	err = database.InitSchemas(db)
	if err != nil {
		t.Fatalf("Cannot setup database schemas: %s", err)
	}

	return db
}
