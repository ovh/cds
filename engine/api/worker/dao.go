package worker

import (
	"context"
	"strings"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func Insert(ctx context.Context, db gorp.SqlExecutor, w *sdk.Worker) error {
	dbData := &dbWorker{Worker: *w}
	if err := gorpmapping.InsertAndSign(ctx, db, dbData); err != nil {
		return err
	}
	*w = dbData.Worker
	return nil
}

// Delete remove worker from database, it also removes the associated consumer.
func Delete(db gorp.SqlExecutor, id string) error {
	consumerID, err := db.SelectNullStr("SELECT auth_consumer_id FROM worker WHERE id = $1", id)
	if err != nil {
		return sdk.WithStack(err)
	}

	query := `DELETE FROM worker WHERE id = $1`
	if _, err := db.Exec(query, id); err != nil {
		return sdk.WithStack(err)
	}

	if consumerID.Valid {
		if err := authentication.DeleteConsumerByID(db, consumerID.String); err != nil {
			return err
		}
	}

	if _, err := db.Exec("UPDATE workflow_node_run_job SET worker_id = NULL WHERE worker_id = $1", id); err != nil {
		return sdk.WrapError(err, "cannot update workflow_node_run_job to remove worker id in job if exists")
	}

	return nil
}

// ReleaseAllFromHatchery remove dependency to given given hatchery for all workers linked to it.
func ReleaseAllFromHatchery(db gorp.SqlExecutor, hatcheryID int64) error {
	if _, err := db.Exec("UPDATE worker SET hatchery_id = NULL WHERE hatchery_id = $1", hatcheryID); err != nil {
		return sdk.WrapError(err, "cannot release workers for hatchery with id %d", hatcheryID)
	}
	return nil
}

func LoadByConsumerID(ctx context.Context, db gorp.SqlExecutor, id string) (*sdk.Worker, error) {
	query := gorpmapping.NewQuery("SELECT * FROM worker WHERE auth_consumer_id = $1").Args(id)
	var w dbWorker
	found, err := gorpmapping.Get(ctx, db, query, &w)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	isValid, err := gorpmapping.CheckSignature(w, w.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return nil, sdk.WithStack(sdk.ErrInvalidData)
	}
	return &w.Worker, nil
}

func LoadByID(ctx context.Context, db gorp.SqlExecutor, id string) (*sdk.Worker, error) {
	query := gorpmapping.NewQuery("SELECT * FROM worker WHERE id = $1").Args(id)
	var w dbWorker
	found, err := gorpmapping.Get(ctx, db, query, &w)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	isValid, err := gorpmapping.CheckSignature(w, w.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return nil, sdk.WithStack(sdk.ErrInvalidData)
	}
	return &w.Worker, nil
}

func LoadAll(ctx context.Context, db gorp.SqlExecutor) ([]sdk.Worker, error) {
	var wks []dbWorker
	query := gorpmapping.NewQuery(`SELECT * FROM worker ORDER BY name ASC`)
	if err := gorpmapping.GetAll(ctx, db, query, &wks); err != nil {
		return nil, err
	}
	workers := make([]sdk.Worker, len(wks))
	for i := range wks {
		isValid, err := gorpmapping.CheckSignature(wks[i], wks[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			return nil, sdk.WithStack(sdk.ErrInvalidData)
		}
		workers[i] = wks[i].Worker
	}
	return workers, nil
}

func LoadAllByHatcheryID(ctx context.Context, db gorp.SqlExecutor, hatcheryID int64) ([]sdk.Worker, error) {
	var wks []dbWorker
	query := gorpmapping.NewQuery(`SELECT * FROM worker WHERE hatchery_id = $1 ORDER BY name ASC`).Args(hatcheryID)
	if err := gorpmapping.GetAll(ctx, db, query, &wks); err != nil {
		return nil, err
	}
	workers := make([]sdk.Worker, len(wks))
	for i := range wks {
		isValid, err := gorpmapping.CheckSignature(wks[i], wks[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			return nil, sdk.WithStack(sdk.ErrInvalidData)
		}
		workers[i] = wks[i].Worker
	}
	return workers, nil
}

func LoadDeadWorkers(ctx context.Context, db gorp.SqlExecutor, timeout float64, status []string) ([]sdk.Worker, error) {
	var wks []dbWorker
	query := gorpmapping.NewQuery(`SELECT *
				FROM worker
				WHERE status = ANY(string_to_array($1, ',')::text[])
				AND now() - last_beat > $2 * INTERVAL '1' SECOND
				ORDER BY last_beat ASC`).Args(strings.Join(status, ","), timeout)
	if err := gorpmapping.GetAll(ctx, db, query, &wks); err != nil {
		return nil, err
	}
	workers := make([]sdk.Worker, len(wks))
	for i := range wks {
		isValid, err := gorpmapping.CheckSignature(wks[i], wks[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			return nil, sdk.WithStack(sdk.ErrInvalidData)
		}
		workers[i] = wks[i].Worker
	}
	return workers, nil
}

// SetStatus sets job_run_id and status to building on given worker
func SetStatus(ctx context.Context, db gorp.SqlExecutor, workerID string, status string) error {
	w, err := LoadByID(ctx, db, workerID)
	if err != nil {
		return err
	}
	w.Status = status
	if status == sdk.StatusBuilding || status == sdk.StatusWaiting {
		w.JobRunID = nil
	}
	dbData := &dbWorker{Worker: *w}
	if err := gorpmapping.UpdateAndSign(ctx, db, dbData); err != nil {
		return err
	}
	return nil
}

// SetToBuilding sets job_run_id and status to building on given worker
func SetToBuilding(ctx context.Context, db gorp.SqlExecutor, workerID string, jobRunID int64, key []byte) error {
	w, err := LoadByID(ctx, db, workerID)
	if err != nil {
		return err
	}
	w.Status = sdk.StatusBuilding
	w.JobRunID = &jobRunID
	w.PrivateKey = key

	dbData := &dbWorker{Worker: *w}
	if err := gorpmapping.UpdateAndSign(ctx, db, dbData); err != nil {
		return err
	}
	return nil
}

// LoadWorkerByIDWithDecryptKey load worker with decrypted private key
func LoadWorkerByIDWithDecryptKey(ctx context.Context, db gorp.SqlExecutor, workerID string) (*sdk.Worker, error) {
	var work dbWorker
	query := gorpmapping.NewQuery(`SELECT * FROM worker WHERE id = $1`).Args(workerID)
	found, err := gorpmapping.Get(ctx, db, query, &work, gorpmapping.GetOptions.WithDecryption)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	isValid, err := gorpmapping.CheckSignature(work, work.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return nil, sdk.WithStack(sdk.ErrInvalidData)
	}
	return &work.Worker, err
}

// LoadWorkerByName load worker by name
func LoadWorkerByName(ctx context.Context, db gorp.SqlExecutor, workerName string) (*sdk.Worker, error) {
	var work dbWorker
	query := gorpmapping.NewQuery(`SELECT * FROM worker WHERE name = $1`).Args(workerName)
	found, err := gorpmapping.Get(ctx, db, query, &work)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	isValid, err := gorpmapping.CheckSignature(work, work.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		return nil, sdk.WithStack(sdk.ErrInvalidData)
	}
	return &work.Worker, err
}
