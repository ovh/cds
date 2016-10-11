package stats

import (
	"database/sql"
	"time"

	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// StartRoutine starts a routine collecting regular build statistics
func StartRoutine() {

	go func() {
		defer sdk.Exit("StatsRoutine exited")

		for {

			time.Sleep(2 * time.Second)

			db := database.DB()
			if db != nil {
				err := createTodaysRow(db)
				if err != nil {
					log.Critical("StatsRoutine: Cannot create today's row: %s\n", err)
					continue
				}

				err = updateWorkerStats(db)
				if err != nil {
					log.Warning("StatsRoutine> Cannot update build stats: %s\n", err)
				}
				err = updatePipelineStats(db)
				if err != nil {
					log.Warning("StatsRoutine> Cannot update build stats: %s\n", err)
				}
			}
		}
	}()
}

func updateWorkerStats(db *sql.DB) error {
	query := `UPDATE stats
	SET max_building_worker = (SELECT COUNT(id) FROM worker WHERE status = $1)
	WHERE day = current_date
	AND max_building_worker < (SELECT COUNT(id) FROM worker WHERE status = $1)`

	res, err := db.Exec(query, string(sdk.StatusBuilding))
	if err != nil {
		return err
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if ra > 1 {
		log.Critical("updateWorkerStats> Bug in the matrice ! multiple (%d) rows with same date", ra)
	}

	return nil
}

func updatePipelineStats(db *sql.DB) error {
	query := `UPDATE stats
	SET max_building_pipeline = (SELECT COUNT(id) FROM pipeline_build WHERE status = $1)
	WHERE day = current_date
	AND max_building_pipeline < (SELECT COUNT(id) FROM pipeline_build WHERE status = $1)`

	res, err := db.Exec(query, string(sdk.StatusBuilding))
	if err != nil {
		return err
	}
	ra, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if ra > 1 {
		log.Critical("updatePipelineStats> Bug in the matrice ! multiple (%d) rows with same date", ra)
	}

	return nil
}

func createTodaysRow(db *sql.DB) error {
	query := `SELECT day FROM stats WHERE day = current_date`
	var day time.Time

	err := db.QueryRow(query).Scan(&day)
	if err == nil {
		return nil
	}

	if err != nil && err != sql.ErrNoRows {
		return err
	}

	query = `INSERT INTO stats (day, build, testing, unit_test, deployment, max_building_worker, max_building_pipeline) VALUES (current_date, 0, 0, 0, 0, 0, 0)`
	_, err = db.Exec(query)
	if err != nil {
		return err
	}
	return nil
}
