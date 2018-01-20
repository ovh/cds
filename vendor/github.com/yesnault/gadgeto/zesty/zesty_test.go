package zesty

import (
	"database/sql"
	"testing"

	"github.com/go-gorp/gorp"
	_ "github.com/mattn/go-sqlite3"
)

const (
	dbName = "test"
	value1 = 1
	value2 = 2
	value3 = 3
	value4 = 4
)

func expectValue(t *testing.T, dbp DBProvider, expected int64) {
	i, err := dbp.DB().SelectInt(`SELECT id FROM "t"`)
	if err != nil {
		t.Fatal(err)
	}
	if i != expected {
		t.Fatalf("unexpected value found in table, expecting %d", expected)
	}
}

func insertValue(t *testing.T, dbp DBProvider, value int64) {
	_, err := dbp.DB().Exec(`INSERT INTO "t" VALUES (?)`, value)
	if err != nil {
		t.Fatal(err)
	}
}

func updateValue(t *testing.T, dbp DBProvider, value int64) {
	_, err := dbp.DB().Exec(`UPDATE "t" SET id = ?`, value)
	if err != nil {
		t.Fatal(err)
	}
}

func rollback(t *testing.T, dbp DBProvider) {
	err := dbp.Rollback()
	if err != nil {
		t.Fatal(err)
	}
}

func rollbackTo(t *testing.T, dbp DBProvider, sp SavePoint) {
	err := dbp.RollbackTo(sp)
	if err != nil {
		t.Fatal(err)
	}
}

func tx(t *testing.T, dbp DBProvider) {
	err := dbp.Tx()
	if err != nil {
		t.Fatal(err)
	}
}

func txSavepoint(t *testing.T, dbp DBProvider) SavePoint {
	sp, err := dbp.TxSavepoint()
	if err != nil {
		t.Fatal(err)
	}
	return sp
}

func TestTransaction(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	RegisterDB(
		NewDB(&gorp.DbMap{
			Db:      db,
			Dialect: gorp.SqliteDialect{},
		}),
		dbName,
	)
	dbp, err := NewDBProvider(dbName)
	if err != nil {
		t.Fatal(err)
	}

	_, err = dbp.DB().Exec(`CREATE TABLE "t" (id BIGINT);`)
	if err != nil {
		t.Fatal(err)
	}

	// first transaction: insert value 1
	tx(t, dbp)

	insertValue(t, dbp, value1)

	expectValue(t, dbp, value1)

	// second transaction: update value to 2
	sp1 := txSavepoint(t, dbp)

	updateValue(t, dbp, value2)
	expectValue(t, dbp, value2)

	tx(t, dbp)

	updateValue(t, dbp, value3)
	expectValue(t, dbp, value3)

	tx(t, dbp)

	updateValue(t, dbp, value4)
	expectValue(t, dbp, value4)

	rollback(t, dbp)

	expectValue(t, dbp, value3)

	// rollback on second transaction: value back to 1
	rollbackTo(t, dbp, sp1)

	expectValue(t, dbp, value1)

	// noop rollback: savepoint already removed in previous rollback
	rollbackTo(t, dbp, sp1)

	expectValue(t, dbp, value1)

	// rollback on first transaction: empty table
	rollback(t, dbp)

	j, err := dbp.DB().SelectNullInt(`SELECT id FROM "t"`)
	if err != nil {
		t.Fatal(err)
	}
	if j.Valid {
		t.Fatal("wrong value, was expecting empty sql.NullInt64 (no rows found)")
	}

	// no rollback possible after exiting outermost Tx
	err = dbp.Rollback()
	if err == nil {
		t.Fatal("rollback should fail when there is no transaction")
	}
}
