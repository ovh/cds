package stats

import (
	"context"
	"database/sql"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// StartRoutine starts a routine collecting regular build statistics
func StartRoutine(c context.Context, DBFunc func() *gorp.DbMap) {
	go func() {
		tick := time.NewTicker(10 * time.Second).C
		for {
			select {
			case <-c.Done():
				if c.Err() != nil {
					log.Error("Exiting Stat routine: %v", c.Err())
				}
				return
			case <-tick:
				db := DBFunc()
				if db != nil {
					err := createTodaysRow(db)
					if err != nil {
						log.Error("StatsRoutine: Cannot create today's row: %s\n", err)
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
		}
	}()
}

func updateWorkerStats(db gorp.SqlExecutor) error {
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
		log.Error("updateWorkerStats> Bug in the matrice ! multiple (%d) rows with same date", ra)
	}

	return nil
}

func updatePipelineStats(db gorp.SqlExecutor) error {
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
		log.Error("updatePipelineStats> Bug in the matrice ! multiple (%d) rows with same date", ra)
	}

	return nil
}

func createTodaysRow(db gorp.SqlExecutor) error {
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
