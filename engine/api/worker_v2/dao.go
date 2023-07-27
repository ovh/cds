package worker_v2

import (
	"context"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func getWorkers(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query, opts ...gorpmapping.GetOptionFunc) ([]sdk.V2Worker, error) {
	var dbWs []dbWorker
	if err := gorpmapping.GetAll(ctx, db, query, &dbWs, opts...); err != nil {
		return nil, err
	}
	var workers []sdk.V2Worker
	for _, w := range dbWs {
		isValid, err := gorpmapping.CheckSignature(w, w.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "worker %s: data corrupted", w.ID)
			continue
		}
		workers = append(workers, w.V2Worker)
	}
	return workers, nil
}

func getWorker(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query, opts ...gorpmapping.GetOptionFunc) (*sdk.V2Worker, error) {
	var dbW dbWorker
	found, err := gorpmapping.Get(ctx, db, query, &dbW, opts...)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.WrapError(sdk.ErrNotFound, "unable to find v2_worker")
	}
	isValid, err := gorpmapping.CheckSignature(dbW, dbW.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "worker %s: data corrupted", dbW.ID)
		return nil, sdk.ErrNotFound
	}
	return &dbW.V2Worker, nil
}

func Insert(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, w *sdk.V2Worker) error {
	ctx, next := telemetry.Span(ctx, "worker.insert")
	defer next()
	w.ID = sdk.UUID()

	dbWkr := &dbWorker{V2Worker: *w}
	if err := gorpmapping.InsertAndSign(ctx, tx, dbWkr); err != nil {
		return err
	}
	*w = dbWkr.V2Worker
	return nil
}

func Update(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, w *sdk.V2Worker) error {
	ctx, next := telemetry.Span(ctx, "worker.Update")
	defer next()
	dbWkr := &dbWorker{V2Worker: *w}
	if err := gorpmapping.UpdateAndSign(ctx, tx, dbWkr); err != nil {
		return err
	}
	*w = dbWkr.V2Worker
	return nil
}

func deleteWorker(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, w sdk.V2Worker) error {
	ctx, next := telemetry.Span(ctx, "worker.deleteWorker")
	defer next()
	dbWkr := &dbWorker{V2Worker: w}
	if err := gorpmapping.Delete(tx, dbWkr); err != nil {
		return err
	}
	return nil
}

func LoadByConsumerID(ctx context.Context, db gorp.SqlExecutor, authConsumerID string) (*sdk.V2Worker, error) {
	query := gorpmapping.NewQuery("SELECT * FROM v2_worker WHERE auth_consumer_id = $1").Args(authConsumerID)
	return getWorker(ctx, db, query)
}

func LoadByID(ctx context.Context, db gorp.SqlExecutor, workerID string, opts ...gorpmapping.GetOptionFunc) (*sdk.V2Worker, error) {
	ctx, next := telemetry.Span(ctx, "v2_worker.LoadByID")
	defer next()
	query := gorpmapping.NewQuery("SELECT * FROM v2_worker WHERE id = $1").Args(workerID)
	return getWorker(ctx, db, query, opts...)
}

// LoadWorkerByName load worker
func LoadWorkerByName(ctx context.Context, db gorp.SqlExecutor, workerName string, opts ...gorpmapping.GetOptionFunc) (*sdk.V2Worker, error) {
	query := gorpmapping.NewQuery(`SELECT * FROM v2_worker WHERE name = $1`).Args(workerName)
	return getWorker(ctx, db, query, opts...)
}

func LoadWorkerByStatus(ctx context.Context, db gorp.SqlExecutor, status string) ([]sdk.V2Worker, error) {
	query := gorpmapping.NewQuery(`SELECT * FROM v2_worker WHERE status = $1`).Args(status)
	return getWorkers(ctx, db, query)
}

func LoadAllWorker(ctx context.Context, db gorp.SqlExecutor) ([]sdk.V2Worker, error) {
	query := gorpmapping.NewQuery(`SELECT * FROM v2_worker`)
	return getWorkers(ctx, db, query)
}

func LoadDeadWorkers(ctx context.Context, db gorp.SqlExecutor, timeout float64, status []string) ([]sdk.V2Worker, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
		FROM v2_worker
		WHERE status = ANY(string_to_array($1, ',')::text[])
		AND now() - last_beat > $2 * INTERVAL '1' SECOND
    ORDER BY last_beat ASC
  `).Args(strings.Join(status, ","), timeout)
	return getWorkers(ctx, db, query)
}
