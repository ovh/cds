package test

import (
	"database/sql"
	"os"
	"path"
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

	sqlfile := path.Join(os.Getenv("GOPATH"), "src", "github.com", "ovh", "cds", "engine", "sql", "create_table.sql")

	if err = database.InitSchemas(db, sqlfile); err != nil {
		t.Fatalf("Cannot setup database schemas: %s", err)
	}

	return db
}
