package project

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

// InsertRunFilter inserts a new filter into the database
func InsertRunFilter(ctx context.Context, db gorpmapper.SqlExecutorWithTx, filter *sdk.ProjectRunFilter) error {
	filter.ID = sdk.UUID()
	filter.LastModified = time.Now()

	dbFilter := dbProjectRunFilter{*filter}
	if err := gorpmapping.Insert(db, &dbFilter); err != nil {
		if errPG, ok := err.(*pq.Error); ok && errPG.Code == "23505" {
			return sdk.WithStack(sdk.NewErrorFrom(sdk.ErrConflictData, "filter name already exists in this project"))
		}
		return sdk.WithStack(err)
	}

	*filter = dbFilter.ProjectRunFilter
	return nil
}

// DeleteRunFilter deletes a filter
func DeleteRunFilter(db gorpmapper.SqlExecutorWithTx, projectKey string, filterID string) error {
	query := "DELETE FROM project_run_filter WHERE id = $1 AND project_key = $2"
	_, err := db.Exec(query, filterID, projectKey)
	return sdk.WithStack(err)
}

// LoadRunFiltersByProjectKey loads all filters for a project, sorted by order then by name.
// The secondary sort by name ensures deterministic order for filters
// with the same order value (typical case: migrated filters all having order=0)
func LoadRunFiltersByProjectKey(ctx context.Context, db gorp.SqlExecutor, projectKey string) ([]sdk.ProjectRunFilter, error) {
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM project_run_filter
		WHERE project_key = $1
		ORDER BY "order" ASC, name ASC
	`).Args(projectKey)

	return getRunFilters(ctx, db, query)
}

// LoadRunFilterByNameAndProjectKey loads a filter by its name
func LoadRunFilterByNameAndProjectKey(ctx context.Context, db gorp.SqlExecutor, projectKey string, name string) (*sdk.ProjectRunFilter, error) {
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM project_run_filter
		WHERE project_key = $1 AND name = $2
	`).Args(projectKey, name)

	return getRunFilter(ctx, db, query)
}

// UpdateRunFilterOrder updates only the order field of a filter
func UpdateRunFilterOrder(ctx context.Context, db gorpmapper.SqlExecutorWithTx, projectKey string, name string, order int64) error {
	query := `
		UPDATE project_run_filter
		SET "order" = $1, last_modified = $2
		WHERE project_key = $3 AND name = $4
	`
	_, err := db.Exec(query, order, time.Now(), projectKey, name)
	return sdk.WithStack(err)
}

// RecomputeRunFilterOrder recomputes the order values of remaining filters after deletion.
// Filters are reloaded sorted by order ASC, name ASC, then reassigned from 0 to N-1.
func RecomputeRunFilterOrder(ctx context.Context, db gorpmapper.SqlExecutorWithTx, projectKey string) error {
	filters, err := LoadRunFiltersByProjectKey(ctx, db, projectKey)
	if err != nil {
		return err
	}
	for i, f := range filters {
		if f.Order != int64(i) {
			if err := UpdateRunFilterOrder(ctx, db, projectKey, f.Name, int64(i)); err != nil {
				return err
			}
		}
	}
	return nil
}

// Helper: getRunFilter loads a single filter
func getRunFilter(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) (*sdk.ProjectRunFilter, error) {
	var dbFilter dbProjectRunFilter
	found, err := gorpmapping.Get(ctx, db, query, &dbFilter)
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &dbFilter.ProjectRunFilter, nil
}

// Helper: getRunFilters loads multiple filters
func getRunFilters(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.ProjectRunFilter, error) {
	var dbFilters []dbProjectRunFilter
	if err := gorpmapping.GetAll(ctx, db, query, &dbFilters); err != nil {
		return nil, sdk.WithStack(err)
	}

	filters := make([]sdk.ProjectRunFilter, len(dbFilters))
	for i, dbf := range dbFilters {
		filters[i] = dbf.ProjectRunFilter
	}
	return filters, nil
}
