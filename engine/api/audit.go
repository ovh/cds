package main

import (
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

const (
	maxVersion = 10
)

func auditCleanerRoutine(DBFunc func() *gorp.DbMap) {
	defer sdk.Exit("AuditCleanerRoutine exited")

	for {
		db := DBFunc()
		if db != nil {
			err := actionAuditCleaner(db)
			if err != nil {
				log.Warning("AuditCleanerRoutine> Action clean failed: %s\n", err)
			}
		}
		time.Sleep(1 * time.Minute)
	}
}

func actionAuditCleaner(db *gorp.DbMap) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Load all action and the number of version in database
	query := `SELECT action_id, COUNT(versionned) FROM action_audit GROUP BY action_id`
	rows, err := tx.Query(query)
	if err != nil {
		return err
	}
	var toDel []int64
	var actionID, count int64
	for rows.Next() {
		err = rows.Scan(&actionID, &count)
		if err != nil {
			rows.Close()
			return err
		}

		if count > maxVersion {
			toDel = append(toDel, actionID)
		}
	}
	rows.Close()

	// Now delete older version to keep only 20
	query = `DELETE FROM action_audit
						WHERE action_id = $1 AND versionned IN
	( SELECT versionned FROM action_audit
		WHERE action_id = $1
		ORDER BY versionned DESC
		OFFSET $2
	)`
	for _, id := range toDel {
		_, err = tx.Exec(query, id, maxVersion)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}
