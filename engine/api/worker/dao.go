package worker

import (
	"context"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func getAll(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]sdk.Worker, error) {
	ws := []dbWorker{}

	if err := gorpmapping.GetAll(ctx, db, q, &ws); err != nil {
		return nil, sdk.WrapError(err, "cannot get workers")
	}

	// Check signature of data, if invalid do not return it
	verifiedWorkers := make([]sdk.Worker, 0, len(ws))
	for i := range ws {
		isValid, err := gorpmapping.CheckSignature(ws[i], ws[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "worker.getAll> worker %s data corrupted", ws[i].ID)
			continue
		}
		verifiedWorkers = append(verifiedWorkers, ws[i].Worker)
	}

	return verifiedWorkers, nil
}

func get(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...gorpmapping.GetOptionFunc) (*sdk.Worker, error) {
	var w dbWorker

	found, err := gorpmapping.Get(ctx, db, q, &w, opts...)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get worker")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	isValid, err := gorpmapping.CheckSignature(w, w.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "worker.get> worker %s data corrupted", w.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	return &w.Worker, nil
}

func Insert(ctx context.Context, db gorpmapper.SqlExecutorWithTx, w *sdk.Worker) error {
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

func LoadByConsumerID(ctx context.Context, db gorp.SqlExecutor, id string) (*sdk.Worker, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM worker
    WHERE auth_consumer_id = $1
  `).Args(id)
	return get(ctx, db, query)
}

func LoadByID(ctx context.Context, db gorp.SqlExecutor, id string, opts ...gorpmapping.GetOptionFunc) (*sdk.Worker, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM worker
    WHERE id = $1
  `).Args(id)
	return get(ctx, db, query, opts...)
}

func LoadAll(ctx context.Context, db gorp.SqlExecutor) ([]sdk.Worker, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM worker
    ORDER BY name ASC
  `)
	return getAll(ctx, db, query)
}

func LoadAllByHatcheryID(ctx context.Context, db gorp.SqlExecutor, hatcheryID int64) ([]sdk.Worker, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM worker
    WHERE hatchery_id = $1
    ORDER BY name ASC
  `).Args(hatcheryID)
	return getAll(ctx, db, query)
}

func LoadDeadWorkers(ctx context.Context, db gorp.SqlExecutor, timeout float64, status []string) ([]sdk.Worker, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
		FROM worker
		WHERE status = ANY(string_to_array($1, ',')::text[])
		AND now() - last_beat > $2 * INTERVAL '1' SECOND
    ORDER BY last_beat ASC
  `).Args(strings.Join(status, ","), timeout)
	return getAll(ctx, db, query)
}

// SetStatus sets job_run_id and status to building on given worker
func SetStatus(ctx context.Context, db gorpmapper.SqlExecutorWithTx, workerID string, status string) error {
	w, err := LoadByID(ctx, db, workerID, gorpmapping.GetOptions.WithDecryption)
	if err != nil {
		return err
	}
	w.Status = status
	if status == sdk.StatusBuilding || status == sdk.StatusWaiting || status == sdk.StatusDisabled {
		w.JobRunID = nil
	}
	dbData := &dbWorker{Worker: *w}
	if err := gorpmapping.UpdateAndSign(ctx, db, dbData); err != nil {
		return err
	}
	return nil
}

// SetToBuilding sets job_run_id and status to building on given worker
func SetToBuilding(ctx context.Context, db gorpmapper.SqlExecutorWithTx, workerID string, jobRunID int64, key []byte) error {
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
func LoadWorkerByNameWithDecryptKey(ctx context.Context, db gorp.SqlExecutor, workerName string) (*sdk.Worker, error) {
	query := gorpmapping.NewQuery(`SELECT * FROM worker WHERE name = $1`).Args(workerName)
	return get(ctx, db, query, gorpmapping.GetOptions.WithDecryption)
}

// LoadWorkerByIDWithDecryptKey load worker with decrypted private key
func LoadWorkerByName(ctx context.Context, db gorp.SqlExecutor, workerName string) (*sdk.Worker, error) {
	query := gorpmapping.NewQuery(`SELECT * FROM worker WHERE name = $1`).Args(workerName)
	return get(ctx, db, query)
}
