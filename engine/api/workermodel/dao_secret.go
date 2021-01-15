package workermodel

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

// LoadSecretsByModelID retrieves all worker model secrets for given model id.
func LoadSecretsByModelID(ctx context.Context, db gorp.SqlExecutor, workerModelID int64) (sdk.WorkerModelSecrets, error) {
	var dbSecrets []workerModelSecret

	q := gorpmapping.NewQuery(`
    SELECT *
    FROM worker_model_secret
    WHERE worker_model_id = $1
  `).Args(workerModelID)

	if err := gorpmapping.GetAll(ctx, db, q, &dbSecrets, gorpmapping.GetOptions.WithDecryption); err != nil {
		return nil, sdk.WrapError(err, "cannot load secrets for worker model with id %d", workerModelID)
	}

	// Check signature of data, if invalid do not return it
	verifiedSecrets := make(sdk.WorkerModelSecrets, 0, len(dbSecrets))
	for i := range dbSecrets {
		isValid, err := gorpmapping.CheckSignature(dbSecrets[i], dbSecrets[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "workermodel.LoadSecretsByModelID> worker model secret %s data corrupted", dbSecrets[i].ID)
			continue
		}
		verifiedSecrets = append(verifiedSecrets, dbSecrets[i].WorkerModelSecret)
	}

	return verifiedSecrets, nil
}

// LoadSecretByModelIDAndName retrieves a worker model secret for given model id and secret name.
func LoadSecretByModelIDAndName(ctx context.Context, db gorp.SqlExecutor, workerModelID int64, name string) (*sdk.WorkerModelSecret, error) {
	var dbSecret workerModelSecret

	q := gorpmapping.NewQuery(`
    SELECT *
    FROM worker_model_secret
    WHERE worker_model_id = $1 AND name = $2
  `).Args(workerModelID, name)

	found, err := gorpmapping.Get(ctx, db, q, &dbSecret, gorpmapping.GetOptions.WithDecryption)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot load secret for worker model with id %d with name %s", workerModelID, name)
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	isValid, err := gorpmapping.CheckSignature(dbSecret, dbSecret.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "workermodel.LoadSecretsByModelID> worker model secret %s data corrupted", dbSecret.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	return &dbSecret.WorkerModelSecret, nil
}

// InsertSecret in database.
func InsertSecret(ctx context.Context, db gorpmapper.SqlExecutorWithTx, s *sdk.WorkerModelSecret) error {
	s.ID = sdk.UUID()
	s.Created = time.Now()
	dbSecret := workerModelSecret{WorkerModelSecret: *s}
	if err := gorpmapping.InsertAndSign(ctx, db, &dbSecret); err != nil {
		return sdk.WrapError(err, "unable to insert worker model secret")
	}
	*s = dbSecret.WorkerModelSecret
	return nil
}

// UpdateSecret in database.
func UpdateSecret(ctx context.Context, db gorpmapper.SqlExecutorWithTx, s *sdk.WorkerModelSecret) error {
	dbSecret := workerModelSecret{WorkerModelSecret: *s}
	if err := gorpmapping.UpdateAndSign(ctx, db, &dbSecret); err != nil {
		return sdk.WrapError(err, "unable to update worker model secret with id: %s", s.ID)
	}
	*s = dbSecret.WorkerModelSecret
	return nil
}

// DeleteSecretForModelID remove registry secret from database for given model.
func DeleteSecretForModelID(db gorp.SqlExecutor, workerModelID int64, field string) error {
	_, err := db.Exec("DELETE FROM worker_model_secret WHERE worker_model_id = $1 AND name = $2", workerModelID, field)
	return sdk.WrapError(err, "unable to remove worker model secret for worker model id %d", workerModelID)
}
