package stats

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/log"
	"github.com/runabove/venom"
)

// TestEvent inserts in stats the max number of tests for each application and total in stats
func TestEvent(db gorp.SqlExecutor, projectID, appID int64, tests venom.Tests) {
	query := `UPDATE stats SET unit_test = unit_test + $1 WHERE day = current_date`

	// Update global daily stats table
	_, err := db.Exec(query, tests.Total)
	if err != nil {
		log.Warning("TestEvent: Cannot update stats table: %s\n", err)
	}

	// Insert activity row for day/project/app if not present
	err = checkActivityRow(db, projectID, appID)
	if err != nil {
		log.Warning("PipelineEvent: Cannot create activity row: %s\n", err)
		return
	}

	// Update activity row
	query = `UPDATE activity SET unit_test = $3
	WHERE day = current_date AND project_id = $1 AND application_id = $2
	AND unit_test < $3`

	_, err = db.Exec(query, projectID, appID, tests.Total)
	if err != nil {
		log.Warning("PipelineEvent: Cannot update stats table: %s\n", err)
	}
}

// PipelineEvent inserts in stats table data related to build
func PipelineEvent(db gorp.SqlExecutor, t string, projectID, appID int64) {
	// Update stats table
	query := `UPDATE stats SET %s = %s + 1
	WHERE day = current_date`
	query = fmt.Sprintf(query, string(t), string(t))

	_, err := db.Exec(query)
	if err != nil {
		log.Warning("PipelineEvent: Cannot update stats table: %s\n", err)
	}

	// Insert activity row for day/project/app if not present
	err = checkActivityRow(db, projectID, appID)
	if err != nil {
		log.Warning("PipelineEvent: Cannot create activity row: %s\n", err)
		return
	}

	// Update activity row
	query = `UPDATE activity SET %s = %s + 1
	WHERE day = current_date AND project_id = $1 AND application_id = $2`
	query = fmt.Sprintf(query, string(t), string(t))

	_, err = db.Exec(query, projectID, appID)
	if err != nil {
		log.Warning("PipelineEvent: Cannot update stats table: %s\n", err)
	}
}

func checkActivityRow(db gorp.SqlExecutor, projectID, appID int64) error {
	query := `SELECT day FROM activity
	WHERE day = current_date AND project_id = $1 AND application_id = $2`
	var d time.Time
	err := db.QueryRow(query, projectID, appID).Scan(&d)
	if err == nil {
		return nil
	}
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	query = `INSERT INTO activity (day, project_id, application_id, build, unit_test, testing, deployment) VALUES
	(current_date, $1, $2, 0, 0, 0, 0)`
	_, err = db.Exec(query, projectID, appID)
	if err != nil {
		return err
	}

	return nil
}
