package pipeline

import (
	"database/sql"
	"time"

	"github.com/ovh/cds/engine/api/build"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// AWOLPipelineKiller will search in database for actions :
// - Having building status
// - Without any logs ouput in the last 15 minutes
func AWOLPipelineKiller() {
	// If this goroutine exits, then it's a crash
	defer log.Fatalf("Goroutine of pipeline.AWOLPipelineKiller exited - Exit CDS Engine")

	for {
		time.Sleep(1 * time.Minute)
		db := database.DB()

		if db != nil {
			ids, err := loadAWOLActionBuild(db)
			if err != nil {
				log.Warning("AWOLPipelineKiller> Cannot load awol building actions: %s\n", err)
			}

			for _, id := range ids {
				err = killAWOLAction(db, id)
				if err != nil {
					log.Warning("AWOLPipelineKiller> Cannot kill action build %d: %s\n", id, err)
					time.Sleep(1 * time.Second) // Do not spam an unavailable database
				}
			}
		}
	}
}

func killAWOLAction(db *sql.DB, actionBuildID int64) error {
	log.Warning("killAWOLAction> Killing action_build %d\n", actionBuildID)

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	build.InsertLog(tx, actionBuildID, "SYSTEM", "Killed (Reason: Timeout)\n")
	err = build.UpdateActionBuildStatus(tx, &sdk.ActionBuild{ID: actionBuildID}, sdk.StatusFail)
	if err != nil {
		return err
	}

	query := `UPDATE worker SET status = $1, action_build_id = NULL WHERE action_build_id = $2`
	_, err = tx.Exec(query, string(sdk.StatusDisabled), actionBuildID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// SELECT action_build.id
// JOIN WITH build_log even if there is no log !
// WHERE action_build status is building, obviously
// WHERE action_build was started at least 15 minutes ago
// WHERE LAAAAAAAAAAAAAAST logs are older than 15 minutes OR no logs at all
func loadAWOLActionBuild(db *sql.DB) ([]int64, error) {
	query := `
		SELECT action_build.id FROM action_build
		LEFT OUTER JOIN build_log ON build_log.action_build_id = action_build.id
		WHERE status = 'Building'
		AND action_build.start < NOW() - INTERVAL '15 minutes'
		GROUP BY action_build.id
		HAVING MAX(build_log.timestamp) < NOW() - INTERVAL '15 minutes' OR MAX(build_log.timestamp) IS NULL
		`
	var ids []int64
	var tmp int64

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&tmp)
		if err != nil {
			return nil, err
		}
		ids = append(ids, tmp)
	}

	return ids, nil
}
