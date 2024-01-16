package project_secret

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func get(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...gorpmapper.GetOptionFunc) (*sdk.ProjectSecret, error) {
	var dbSecret dbProjectSecret
	found, err := gorpmapping.Get(ctx, db, q, &dbSecret, opts...)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.NewErrorFrom(sdk.ErrNotFound, "unable to find secret")
	}
	return &dbSecret.ProjectSecret, nil
}

func getAll(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...gorpmapper.GetOptionFunc) ([]sdk.ProjectSecret, error) {
	var dbSecrets []dbProjectSecret
	if err := gorpmapping.GetAll(ctx, db, q, &dbSecrets, opts...); err != nil {
		return nil, err
	}
	secrets := make([]sdk.ProjectSecret, 0, len(dbSecrets))
	for _, s := range dbSecrets {
		secrets = append(secrets, s.ProjectSecret)
	}
	return secrets, nil
}

func Insert(_ context.Context, db gorpmapper.SqlExecutorWithTx, secret *sdk.ProjectSecret) error {
	secret.ID = sdk.UUID()
	secret.LastModified = time.Now()

	dbSecret := dbProjectSecret{ProjectSecret: *secret}
	if err := gorpmapping.Insert(db, &dbSecret); err != nil {
		return err
	}
	*secret = dbSecret.ProjectSecret
	return nil
}

func Update(_ context.Context, db gorpmapper.SqlExecutorWithTx, secret *sdk.ProjectSecret) error {
	secret.LastModified = time.Now()

	dbSecret := dbProjectSecret{ProjectSecret: *secret}
	if err := gorpmapping.Update(db, &dbSecret); err != nil {
		return err
	}
	*secret = dbSecret.ProjectSecret
	return nil
}

func Delete(_ context.Context, db gorpmapper.SqlExecutorWithTx, secret sdk.ProjectSecret) error {
	dbSecret := dbProjectSecret{ProjectSecret: secret}
	if err := gorpmapping.Delete(db, &dbSecret); err != nil {
		return err
	}
	return nil
}

func LoadByName(ctx context.Context, db gorp.SqlExecutor, projectKey, name string, opts ...gorpmapper.GetOptionFunc) (*sdk.ProjectSecret, error) {
	query := gorpmapping.NewQuery("SELECT * FROM project_secret WHERE project_key = $1 AND name = $2").
		Args(projectKey, name)
	return get(ctx, db, query, opts...)
}

func LoadByProjectKey(ctx context.Context, db gorp.SqlExecutor, projectKey string, opts ...gorpmapper.GetOptionFunc) ([]sdk.ProjectSecret, error) {
	query := gorpmapping.NewQuery("SELECT * FROM project_secret WHERE project_key = $1").
		Args(projectKey)
	return getAll(ctx, db, query, opts...)
}
