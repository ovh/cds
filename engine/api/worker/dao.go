package worker

import (
	"strings"

	"github.com/ovh/cds/engine/api/accesstoken"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func Insert(db gorp.SqlExecutor, w *sdk.Worker) error {
	return gorpmapping.Insert(db, w)
}

func Update(db gorp.SqlExecutor, w *sdk.Worker) error {
	return gorpmapping.Update(db, w)
}

// Delete remove worker from database, it also removes the associated access_token
func Delete(db gorp.SqlExecutor, id string) error {
	accessTokenID, err := db.SelectNullStr("SELECT access_token_id FROM worker WHERE id = $1", id)
	if err != nil {
		return sdk.WithStack(err)
	}
	query := `DELETE FROM worker WHERE id = $1`
	if _, err := db.Exec(query, id); err != nil {
		return sdk.WithStack(err)
	}

	if accessTokenID.Valid {
		if err := accesstoken.Delete(db, accessTokenID.String); err != nil {
			return err
		}
	}

	return nil
}

func LoadByAccessTokenID(db gorp.SqlExecutor, id string) (*sdk.Worker, error) {
	query := gorpmapping.NewQuery("SELECT * FROM worker WHERE access_token_id = $1").Args(id)
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
				ORDER BY last_beat ASC`).Args(strings.Join(status, ","), timeout)
	if err := gorpmapping.GetAll(db, query, &workers); err != nil {
		return nil, err
	}
	return workers, nil
}

// SetStatus sets job_run_id and status to building on given worker
func SetStatus(db gorp.SqlExecutor, workerID string, status string) error {
	query := `UPDATE worker SET status = $1 WHERE id = $2`
	if status == sdk.StatusDisabled || status == sdk.StatusWaiting {
		query = `UPDATE worker SET status = $1, job_run_id = NULL WHERE id = $2`
	}

	if _, err := db.Exec(query, status, workerID); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

// SetToBuilding sets job_run_id and status to building on given worker
func SetToBuilding(db gorp.SqlExecutor, workerID string, actionBuildID int64, jobType string) error {
	query := `UPDATE worker SET status = $1, job_run_id = $2, job_type = $3 WHERE id = $4`

	res, errE := db.Exec(query, sdk.StatusDisabled, actionBuildID, jobType, workerID)
	if errE != nil {
		return sdk.WithStack(errE)
	}

	_, err := res.RowsAffected()
	return err
}
