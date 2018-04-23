package api

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk/log"
)

const (
	maxVersion = 10
	delay      = 1
)

func auditCleanerRoutine(c context.Context, DBFunc func(context.Context) *gorp.DbMap) {
	tick := time.NewTicker(delay * time.Minute).C

	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("Exiting auditCleanerRoutine: %v", c.Err())
			}
			return
		case <-tick:
			db := DBFunc(c)
			if db != nil {
				err := actionAuditCleaner(DBFunc(c))
				if err != nil {
					log.Warning("AuditCleanerRoutine> Action clean failed: %s", err)
				}
			}
		}
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
		if _, err := tx.Exec(query, id, maxVersion); err != nil {
			return err
		}
	}

	return tx.Commit()
}
