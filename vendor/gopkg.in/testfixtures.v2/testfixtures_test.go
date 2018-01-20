package testfixtures

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

func TestFixtureFile(t *testing.T) {
	f := &fixtureFile{fileName: "posts.yml"}
	file := f.fileNameWithoutExtension()
	if file != "posts" {
		t.Errorf("Should be 'posts', but returned %s", file)
	}
}

type databaseTest struct {
	name       string
	connEnv    string
	schemaFile string
	helper     Helper
}

var databases = []databaseTest{}

func TestLoadFixtures(t *testing.T) {
	if len(databases) == 0 {
		t.Error("No database chosen for tests!")
	}

	for _, database := range databases {
		connString := os.Getenv(database.connEnv)

		var bytes []byte

		fmt.Printf("Test for %s\n", database.name)

		db, err := sql.Open(database.name, connString)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v\n", err)
		}

		defer db.Close()

		if err = db.Ping(); err != nil {
			log.Fatalf("Failed to ping database: %v\n", err)
		}

		bytes, err = ioutil.ReadFile(database.schemaFile)
		if err != nil {
			log.Fatalf("Could not read file %s: %v\n", database.schemaFile, err)
		}

		_, err = db.Exec(string(bytes))
		if err != nil {
			log.Fatalf("Failed to create schema: %v\n", err)
		}

		testLoadFixtures(t, db, database.helper)
		testLoadFixtureFiles(t, db, database.helper)

		// generate fixtures from database
		dir, err := ioutil.TempDir(os.TempDir(), "testfixtures_test")
		if err != nil {
			t.Error(err)
		}
		if err := GenerateFixtures(db, database.helper, dir); err != nil {
			t.Error(err)
		}

		// should be able to load generated fixtures
		context, err := NewFolder(db, database.helper, dir)
		if err != nil {
			t.Error(err)
		} else if err := context.Load(); err != nil {
			t.Error(err)
		}
	}
}

func TestQuoteKeyword(t *testing.T) {
	tests := []struct {
		Helper   Helper
		Keyword  string
		Expected string
	}{
		{&PostgreSQL{}, `posts_tags`, `"posts_tags"`},
		{&PostgreSQL{}, `test_schema.posts_tags`, `"test_schema"."posts_tags"`},
	}

	for _, test := range tests {
		actual := test.Helper.quoteKeyword(test.Keyword)

		if test.Expected != actual {
			t.Errorf("TestQuoteKeyword keyword %s should have escaped to %s. Received %s instead", test.Keyword, test.Expected, actual)
		}
	}
}

func TestDatabaseNameHelperSurfacesErrors(t *testing.T) {
	if len(databases) == 0 {
		t.Error("No database chosen for tests!")
	}

	for _, database := range databases {
		connString := os.Getenv(database.connEnv)

		fmt.Printf("Test for %s\n", database.name)

		db, err := sql.Open(database.name, connString)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v\n", err)
		}

		// Ensure a connection error occurs
		db.Close()

		_, err = database.helper.databaseName(db)
		if err == nil {
			t.Error("Expected databaseName to surface error")
		}
	}
}

func assertCount(t *testing.T, db *sql.DB, h Helper, table string, expectedCount int) {
	var count int

	row := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", h.quoteKeyword(table)))
	row.Scan(&count)
	if count != expectedCount {
		t.Errorf("%s should have %d, but has %d", table, expectedCount, count)
	}
}

func testLoadFixtures(t *testing.T, db *sql.DB, helper Helper) {
	c, err := NewFolder(db, helper, "testdata/fixtures")
	if err != nil {
		t.Errorf("Error creating context: %v", err)
	}

	if err := c.Load(); err != nil {
		t.Errorf("Error on loading fixtures: %v", err)
	}

	assertCount(t, db, helper, "posts", 2)
	assertCount(t, db, helper, "comments", 4)
	assertCount(t, db, helper, "tags", 3)
	assertCount(t, db, helper, "posts_tags", 2)
	assertCount(t, db, helper, "users", 2)

	// this insert is to test if the PostgreSQL sequences were reset
	var sql string
	switch helper.paramType() {
	case paramTypeDollar:
		sql = "INSERT INTO posts (title, content, created_at, updated_at) VALUES ($1, $2, $3, $4)"
	case paramTypeQuestion:
		sql = "INSERT INTO posts (title, content, created_at, updated_at) VALUES (?, ?, ?, ?)"
	case paramTypeColon:
		sql = "INSERT INTO posts (id, title, content, created_at, updated_at) VALUES (POSTS_SEQ.NEXTVAL, :1, :2, :3, :4)"
	}
	_, err = db.Exec(
		sql,
		"Post title",
		"Post content",
		time.Now(),
		time.Now(),
	)
	if err != nil {
		t.Errorf("Error inserting post: %v", err)
	}
}

func testLoadFixtureFiles(t *testing.T, db *sql.DB, helper Helper) {
	tables := []string{"posts_tags", "comments", "posts", "tags"}
	for _, table := range tables {
		db.Exec("DELETE FROM %s", helper.quoteKeyword(table))
	}

	fixturesFiles := []string{
		"testdata/fixtures/posts.yml",
		"testdata/fixtures/comments.yml",
		"testdata/fixtures/tags.yml",
		"testdata/fixtures/posts_tags.yml",
	}

	c, err := NewFiles(db, helper, fixturesFiles...)
	if err != nil {
		t.Errorf("Error on creating context: %v", err)
	}

	if err := c.Load(); err != nil {
		t.Errorf("Error on loading fixtures: %v", err)
	}

	assertCount(t, db, helper, "posts", 2)
	assertCount(t, db, helper, "comments", 4)
	assertCount(t, db, helper, "tags", 3)
	assertCount(t, db, helper, "posts_tags", 2)
}
