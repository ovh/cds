package worker

import (
	"math"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// DeleteWorker remove worker from database
func DeleteWorker(db gorp.SqlExecutor, id string) error {
	query := `DELETE FROM worker WHERE id = $1`
	if _, err := db.Exec(query, id); err != nil {
		return sdk.WrapError(err, "DeleteWorker")
	}

	return nil
}

func Insert(db gorp.SqlExecutor, w *sdk.Worker) error {
	return gorpmapping.Insert(db, w)
}

func LoadByID(db gorp.SqlExecutor, id string) (*sdk.Worker, error) {
	query := gorpmapping.NewQuery("SELECT * FROM worker WHERE id = $1").Args(id)
	var w sdk.Worker
	found, err := gorpmapping.Get(db, query, &w)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &w, nil
}

func LoadAll(db gorp.SqlExecutor) ([]sdk.Worker, error) {
	var workers []sdk.Worker
	query := gorpmapping.NewQuery(`SELECT * FROM worker ORDER BY name ASC`)
	if err := gorpmapping.GetAll(db, query, &workers); err != nil {
		return nil, err
	}
	return workers, nil
}

func LoadByHatcheryID(db gorp.SqlExecutor, hatcheryID int64) ([]sdk.Worker, error) {
	var workers []sdk.Worker
	query := gorpmapping.NewQuery(`SELECT * FROM worker WHERE hatchery_id = $1 ORDER BY name ASC`).Args(hatcheryID)
	if err := gorpmapping.GetAll(db, query, &workers); err != nil {
		return nil, err
	}
	return workers, nil
}

func LoadDeadWorkers(db gorp.SqlExecutor, timeout float64, status []string) ([]sdk.Worker, error) {
	var workers []sdk.Worker
	query := gorpmapping.NewQuery(`SELECT *
				FROM worker
				WHERE status = ANY(string_to_array($1, ',')::text[])
				AND now() - last_beat > $2 * INTERVAL '1' SECOND
				ORDER BY name last_beat ASC`).Args(strings.Join(status, ","), int64(math.Floor(timeout)))
	if err := gorpmapping.GetAll(db, query, &workers); err != nil {
		return nil, err
	}
	return workers, nil
}

// SetStatus sets job_run_id and status to building on given worker
func SetStatus(db gorp.SqlExecutor, workerID string, status string) error {
	query := `UPDATE worker SET status = $1 WHERE id = $2`
	if status == sdk.StatusDisabled {
		query = `UPDATE worker SET status = $1, job_run_id = NULL WHERE id = $2`
	}

	if _, err := db.Exec(query, status, workerID); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

// SetToBuilding sets job_run_id and status to building on given worker
func SetToBuilding(db gorp.SqlExecutor, store cache.Store, workerID string, actionBuildID int64, jobType string) error {
	query := `UPDATE worker SET status = $1, job_run_id = $2, job_type = $3 WHERE id = $4`

	res, errE := db.Exec(query, sdk.StatusDisabled, actionBuildID, jobType, workerID)
	if errE != nil {
		return sdk.WithStack(errE)
	}

	_, err := res.RowsAffected()
	// delete the worker from the cache
	store.Delete(cache.Key("worker", workerID))
	return err
}

// UpdateWorkerStatus changes worker status to Disabled
func UpdateWorkerStatus(db gorp.SqlExecutor, workerID string, status string) error {
	query := `UPDATE worker SET status = $1, job_run_id = NULL WHERE id = $2`

	res, err := db.Exec(query, status, workerID)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected > 1 {
		log.Error("UpdateWorkerStatus: Multiple (%d) rows affected ! (id=%s)", rowsAffected, workerID)
	}

	if rowsAffected == 0 {
		return ErrNoWorker
	}

	return nil
}
