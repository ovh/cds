package migrateservice

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/go-gorp/gorp"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/engine/test"
	"github.com/ovh/cds/sdk"
)

func Test_doMigrate(t *testing.T) {

	db := initDb(t)

	// Upgrade all the things
	s := dbmigservice{}
	s.cfg.Directory = "fixtures"

	dbFunc := func() *sql.DB { return db.Db }

	migrations, err := execMigrate(context.TODO(), dbFunc, gorp.SqliteDialect{}, "fixtures", "", "")
	require.NoError(t, err)
	require.Len(t, migrations, 3)
	for _, m := range migrations {
		t.Log(m.ID, m.AppliedAt, m.Migrated)
		require.True(t, m.Migrated)
	}

	time.Sleep(1 * time.Second)

	// Downgrade the last
	migrations, err = execMigrate(context.TODO(), dbFunc, gorp.SqliteDialect{}, "fixtures", "", "3_end.sql")
	require.NoError(t, err)
	require.Len(t, migrations, 3)
	for _, m := range migrations {
		t.Log(m.ID, m.AppliedAt, m.Migrated)
		if m.ID == "3_end.sql" {
			require.False(t, m.Migrated)
		} else {
			require.True(t, m.Migrated)
		}
	}

	time.Sleep(1 * time.Second)

	// Upgrade the last
	migrations, err = execMigrate(context.TODO(), dbFunc, gorp.SqliteDialect{}, "fixtures", "3_end.sql", "")
	require.NoError(t, err)
	require.Len(t, migrations, 3)
	for _, m := range migrations {
		t.Log(m.ID, m.AppliedAt, m.Migrated)
		require.True(t, m.Migrated)
	}

	time.Sleep(1 * time.Second)

	// Downgrade the 2 last
	migrations, err = execMigrate(context.TODO(), dbFunc, gorp.SqliteDialect{}, "fixtures", "", "2_record.sql")
	require.NoError(t, err)
	require.Len(t, migrations, 3)
	for _, m := range migrations {
		t.Log(m.ID, m.AppliedAt, m.Migrated)
		if m.ID == "3_end.sql" || m.ID == "2_record.sql" {
			require.False(t, m.Migrated)
		} else {
			require.True(t, m.Migrated)
		}
	}

	time.Sleep(1 * time.Second)

	// Upgrade the 2nd but not the 3rd
	migrations, err = execMigrate(context.TODO(), dbFunc, gorp.SqliteDialect{}, "fixtures", "2_record.sql", "")
	require.NoError(t, err)
	require.Len(t, migrations, 3)
	for _, m := range migrations {
		t.Log(m.ID, m.AppliedAt, m.Migrated)
		if m.ID == "3_end.sql" {
			require.False(t, m.Migrated)
		} else {
			require.True(t, m.Migrated)
		}
	}

	time.Sleep(1 * time.Second)

	// Upgrade the last
	migrations, err = execMigrate(context.TODO(), dbFunc, gorp.SqliteDialect{}, "fixtures", "3_end.sql", "")
	require.NoError(t, err)
	require.Len(t, migrations, 3)
	for _, m := range migrations {
		t.Log(m.ID, m.AppliedAt, m.Migrated)
		require.True(t, m.Migrated)
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
