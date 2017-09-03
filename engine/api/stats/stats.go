package stats

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/venom"
)

// TestEvent inserts in stats the max number of tests for each application and total in stats
func TestEvent(db gorp.SqlExecutor, projectID, appID int64, tests venom.Tests) {
	query := `UPDATE stats SET unit_test = unit_test + $1 WHERE day = current_date`

	// Update global daily stats table
	if _, err := db.Exec(query, tests.Total); err != nil {
		log.Warning("TestEvent: Cannot update stats table: %s\n", err)
	}

	// Insert activity row for day/project/app if not present
	if err := checkActivityRow(db, projectID, appID); err != nil {
		log.Warning("TestEvent: Cannot create activity row: %s\n", err)
		return
	}

	// Update activity row
	query = `UPDATE activity SET unit_test = $3
	WHERE day = current_date AND project_id = $1 AND application_id = $2
	AND unit_test < $3`

	if _, err := db.Exec(query, projectID, appID, tests.Total); err != nil {
		log.Warning("TestEvent: Cannot update activity table: %s, projectID:%d appID:%s tests.Total:%s \n", err, projectID, appID, tests.Total)
	}
}

// PipelineEvent inserts in stats table data related to build
func PipelineEvent(db gorp.SqlExecutor, t string, projectID, appID int64) {
	// Update stats table
	query := `UPDATE stats SET %s = %s + 1
	WHERE day = current_date`
	query = fmt.Sprintf(query, string(t), string(t))

	if _, err := db.Exec(query); err != nil {
		log.Warning("PipelineEvent: Cannot update stats table: %s\n", err)
	}

	// Insert activity row for day/project/app if not present
	if err := checkActivityRow(db, projectID, appID); err != nil {
		log.Warning("PipelineEvent: Cannot create activity row: %s\n", err)
		return
	}

	// Update activity row
	query = `UPDATE activity SET %s = %s + 1
	WHERE day = current_date AND project_id = $1 AND application_id = $2`
	query = fmt.Sprintf(query, string(t), string(t))

	if _, err := db.Exec(query, projectID, appID); err != nil {
		log.Warning("PipelineEvent: Cannot update activity table: %s, projectID:%d appID:%s\n", err, projectID, appID)
	}
}

func checkActivityRow(db gorp.SqlExecutor, projectID, appID int64) error {
	query := `SELECT day FROM activity
	WHERE day = current_date AND project_id = $1 AND application_id = $2`
	var d time.Time
	errq := db.QueryRow(query, projectID, appID).Scan(&d)
	if errq == nil {
		return nil
	}
	if errq != nil && errq != sql.ErrNoRows {
		return errq
	}

	query = `INSERT INTO activity (day, project_id, application_id, build, unit_test, testing, deployment) VALUES
	(current_date, $1, $2, 0, 0, 0, 0)`
	if _, err := db.Exec(query, projectID, appID); err != nil {
		return err
	}
	return nil
}
