package migrateservice

import (
	"database/sql"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/go-gorp/gorp"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func Test_doMigrate(t *testing.T) {
	db := initDb(t)

	// Upgrade all the things
	s := dbmigservice{}
	s.cfg.Directory = "fixtures"

	dbFunc := func() *sql.DB { return db.Db }

	err := s.doMigrate(dbFunc, gorp.SqliteDialect{}, "", "")
	require.NoError(t, err)

	migrations, err := s.getMigrate(dbFunc, gorp.SqliteDialect{})
	require.NoError(t, err)
	assert.Len(t, migrations, 3)
	for _, m := range migrations {
		t.Log(m.ID, m.AppliedAt, m.Migrated)
		assert.True(t, m.Migrated)
	}

	time.Sleep(1 * time.Second)

	// Downgrade the last
	err = s.doMigrate(dbFunc, gorp.SqliteDialect{}, "", "3_end.sql")
	require.NoError(t, err)

	migrations, err = s.getMigrate(dbFunc, gorp.SqliteDialect{})
	require.NoError(t, err)
	assert.Len(t, migrations, 3)
	for _, m := range migrations {
		t.Log(m.ID, m.AppliedAt, m.Migrated)
		if m.ID == "3_end.sql" {
			assert.False(t, m.Migrated)
		} else {
			assert.True(t, m.Migrated)
		}
	}

	time.Sleep(1 * time.Second)

	// Upgrade the last
	err = s.doMigrate(dbFunc, gorp.SqliteDialect{}, "3_end.sql", "")
	require.NoError(t, err)

	migrations, err = s.getMigrate(dbFunc, gorp.SqliteDialect{})
	require.NoError(t, err)
	assert.Len(t, migrations, 3)
	for _, m := range migrations {
		t.Log(m.ID, m.AppliedAt, m.Migrated)
		assert.True(t, m.Migrated)
	}

	time.Sleep(1 * time.Second)

	// Downgrade the 2 last
	err = s.doMigrate(dbFunc, gorp.SqliteDialect{}, "", "2_record.sql")
	require.NoError(t, err)

	migrations, err = s.getMigrate(dbFunc, gorp.SqliteDialect{})
	require.NoError(t, err)
	assert.Len(t, migrations, 3)
	for _, m := range migrations {
		t.Log(m.ID, m.AppliedAt, m.Migrated)
		if m.ID == "3_end.sql" || m.ID == "2_record.sql" {
			assert.False(t, m.Migrated)
		} else {
			assert.True(t, m.Migrated)
		}
	}

	time.Sleep(1 * time.Second)

	// Upgrade the 2nd but not the 3rd
	err = s.doMigrate(dbFunc, gorp.SqliteDialect{}, "2_record.sql", "")
	require.NoError(t, err)

	migrations, err = s.getMigrate(dbFunc, gorp.SqliteDialect{})
	require.NoError(t, err)
	assert.Len(t, migrations, 3)
	for _, m := range migrations {
		t.Log(m.ID, m.AppliedAt, m.Migrated)
		if m.ID == "3_end.sql" {
			assert.False(t, m.Migrated)
		} else {
			assert.True(t, m.Migrated)
		}
	}

	time.Sleep(1 * time.Second)

	// Upgrade the last
	err = s.doMigrate(dbFunc, gorp.SqliteDialect{}, "3_end.sql", "")
	require.NoError(t, err)

	migrations, err = s.getMigrate(dbFunc, gorp.SqliteDialect{})
	require.NoError(t, err)
	assert.Len(t, migrations, 3)
	for _, m := range migrations {
		t.Log(m.ID, m.AppliedAt, m.Migrated)
		assert.True(t, m.Migrated)
	}
}

func initDb(t *testing.T) *gorp.DbMap {
	f := "test-" + test.GetTestName(t) + "-" + sdk.RandomString(10) + "-" + fmt.Sprintf("%d", time.Now().Unix())
	db, err := sql.Open("sqlite3", f)
	checkErr(t, err, "sql.Open failed")
	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.SqliteDialect{}}
	return dbmap
}

func checkErr(t *testing.T, err error, msg string) {
	if err != nil {
		log.Fatalln(msg, err)
	}
}
