package project

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func InsertConcurrency(ctx context.Context, db gorpmapper.SqlExecutorWithTx, c *sdk.ProjectConcurrency) error {
	c.ID = sdk.UUID()
	c.LastModified = time.Now()
	dbData := &dbProjectConcurrency{ProjectConcurrency: *c}
	if err := gorpmapping.Insert(db, dbData); err != nil {
		return err
	}
	*c = dbData.ProjectConcurrency
	return nil
}

func UpdateConcurrency(ctx context.Context, db gorpmapper.SqlExecutorWithTx, c *sdk.ProjectConcurrency) error {
	c.LastModified = time.Now()
	dbData := &dbProjectConcurrency{ProjectConcurrency: *c}
	if err := gorpmapping.Update(db, dbData); err != nil {
		return err
	}
	*c = dbData.ProjectConcurrency
	return nil
}

func DeleteConcurrency(db gorpmapper.SqlExecutorWithTx, projectKey string, concurrencyID string) error {
	_, err := db.Exec("DELETE FROM project_concurrency WHERE id = $1 AND project_key = $2", concurrencyID, projectKey)
	return sdk.WrapError(err, "cannot delete project_concurrency %s / %s", projectKey, concurrencyID)
}

func getConcurrency(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) (*sdk.ProjectConcurrency, error) {
	var res dbProjectConcurrency
	found, err := gorpmapping.Get(ctx, db, query, &res)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &res.ProjectConcurrency, nil
}

func getConcurrencies(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.ProjectConcurrency, error) {
	var res []dbProjectConcurrency
	if err := gorpmapping.GetAll(ctx, db, query, &res); err != nil {
		return nil, err
	}

	concurrencies := make([]sdk.ProjectConcurrency, 0, len(res))
	for _, r := range res {
		concurrencies = append(concurrencies, r.ProjectConcurrency)
	}

	return concurrencies, nil
}

func LoadConcurrencyByIDAndProjectKey(ctx context.Context, db gorp.SqlExecutor, projKey string, id string) (*sdk.ProjectConcurrency, error) {
	query := gorpmapping.NewQuery(`SELECT project_concurrency.* FROM project_concurrency WHERE project_key = $1 AND id = $2`).Args(projKey, id)
	return getConcurrency(ctx, db, query)
}

func LoadConcurrencyByNameAndProjectKey(ctx context.Context, db gorp.SqlExecutor, projKey string, name string) (*sdk.ProjectConcurrency, error) {
	query := gorpmapping.NewQuery(`SELECT project_concurrency.* FROM project_concurrency WHERE project_key = $1 AND name = $2`).Args(projKey, name)
	return getConcurrency(ctx, db, query)
}

func LoadConcurrenciesByProjectKey(ctx context.Context, db gorp.SqlExecutor, projKey string) ([]sdk.ProjectConcurrency, error) {
	query := gorpmapping.NewQuery(`SELECT project_concurrency.* FROM project_concurrency WHERE project_key = $1`).Args(projKey)
	return getConcurrencies(ctx, db, query)
}
