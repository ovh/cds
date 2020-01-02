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

func auditCleanerRoutine(ctx context.Context, DBFunc func() *gorp.DbMap) {
	tick := time.NewTicker(delay * time.Minute).C

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "Exiting auditCleanerRoutine: %v", ctx.Err())
			}
			return
		case <-tick:
			db := DBFunc()
			if db != nil {
				err := actionAuditCleaner(DBFunc())
				if err != nil {
					log.Warning(ctx, "AuditCleanerRoutine> Action clean failed: %s", err)
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
	defer tx.Rollback() // nolint

	// Load all action and the number of version in database
	query := `SELECT action_id, COUNT(versionned) FROM action_audit GROUP BY action_id`
	rows, err := tx.Query(query)
	if err != nil {
		return err
	}
	defer rows.Close() // nolint
	var toDel []int64
	var actionID, count int64
	for rows.Next() {
		err = rows.Scan(&actionID, &count)
		if err != nil {
			return err
		}

		if count > maxVersion {
			toDel = append(toDel, actionID)
		}
	}

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
