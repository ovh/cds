package testfixtures

import (
	"database/sql"
	"log"
	"os"
	"testing"
)

func BenchmarkWithoutCache(b *testing.B) {
	db, helper := getDatabase()

	for i := 0; i < b.N; i++ {
		if err := LoadFixtures("testdata/fixtures", db, helper); err != nil {
			log.Fatal(err)
		}
	}
}

func BenchmarkWithCache(b *testing.B) {
	db, helper := getDatabase()

	c, err := NewFolder(db, helper, "testdata/fixtures")
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		if err := c.Load(); err != nil {
			log.Fatal(err)
		}
	}
}

func getDatabase() (db *sql.DB, helper Helper) {
	if len(databases) == 0 {
		log.Fatal("No database specified")
	}

	var (
		database = databases[0]
		err      error
	)

	db, err = sql.Open(database.name, os.Getenv(database.connEnv))
	if err != nil {
		log.Fatal(err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	helper = database.helper

	return
}
