package worker_v2

import (
  "context"

  "github.com/go-gorp/gorp"
  "github.com/rockbears/log"

  "github.com/ovh/cds/engine/api/database/gorpmapping"
  "github.com/ovh/cds/engine/gorpmapper"
  "github.com/ovh/cds/sdk"
  "github.com/ovh/cds/sdk/telemetry"
)

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

func insert(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, w *sdk.V2Worker) error {
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

func LoadByConsumerID(ctx context.Context, db gorp.SqlExecutor, authConsumerID string) (*sdk.V2Worker, error) {
  query := gorpmapping.NewQuery("SELECT * FROM v2_worker WHERE auth_consumer_id = $1").Args(authConsumerID)
  return getWorker(ctx, db, query)
}

func LoadByID(ctx context.Context, db gorp.SqlExecutor, workerID string, opts ...gorpmapping.GetOptionFunc) (*sdk.V2Worker, error) {
  query := gorpmapping.NewQuery("SELECT * FROM v2_worker WHERE id = $1").Args(workerID)
  return getWorker(ctx, db, query, opts...)
}

// LoadWorkerByName load worker
func LoadWorkerByName(ctx context.Context, db gorp.SqlExecutor, workerName string, opts ...gorpmapping.GetOptionFunc) (*sdk.V2Worker, error) {
  query := gorpmapping.NewQuery(`SELECT * FROM v2_worker WHERE name = $1`).Args(workerName)
  return getWorker(ctx, db, query, opts...)
}
