package testfixtures

import (
	"database/sql"
	"path/filepath"
)

// SQLite is the SQLite Helper for this package
type SQLite struct {
	baseHelper
}

func (*SQLite) paramType() int {
	return paramTypeQuestion
}

func (*SQLite) databaseName(q queryable) (string, error) {
	var seq int
	var main, dbName string
	err := q.QueryRow("PRAGMA database_list").Scan(&seq, &main, &dbName)
	if err != nil {
		return "", err
	}
	dbName = filepath.Base(dbName)
	return dbName, nil
}

func (*SQLite) tableNames(q queryable) ([]string, error) {
	query := `
		SELECT name
		FROM sqlite_master
		WHERE type='table';
	`
	rows, err := q.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err = rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return tables, nil
}

func (*SQLite) disableReferentialIntegrity(db *sql.DB, loadFn loadFunction) (err error) {
	defer func() {
		if _, err2 := db.Exec("PRAGMA defer_foreign_keys = OFF"); err2 != nil && err == nil {
			err = err2
		}
	}()

	if _, err = db.Exec("PRAGMA defer_foreign_keys = ON"); err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err = loadFn(tx); err != nil {
		return err
	}

	return tx.Commit()
}
