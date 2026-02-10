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

// InsertRunFilter insère un nouveau filtre dans la base
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

// DeleteRunFilter supprime un filtre
func DeleteRunFilter(db gorpmapper.SqlExecutorWithTx, projectKey string, filterID string) error {
	query := "DELETE FROM project_run_filter WHERE id = $1 AND project_key = $2"
	_, err := db.Exec(query, filterID, projectKey)
	return sdk.WithStack(err)
}

// LoadRunFiltersByProjectKey charge tous les filtres d'un projet, triés par order puis par name
// Le tri secondaire par name garantit un ordre déterministe pour les filtres
// ayant le même order (cas typique : filtres migrés ayant tous order=0)
func LoadRunFiltersByProjectKey(ctx context.Context, db gorp.SqlExecutor, projectKey string) ([]sdk.ProjectRunFilter, error) {
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM project_run_filter
		WHERE project_key = $1
		ORDER BY "order" ASC, name ASC
	`).Args(projectKey)

	return getRunFilters(ctx, db, query)
}

// LoadRunFilterByNameAndProjectKey charge un filtre par son nom
func LoadRunFilterByNameAndProjectKey(ctx context.Context, db gorp.SqlExecutor, projectKey string, name string) (*sdk.ProjectRunFilter, error) {
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM project_run_filter
		WHERE project_key = $1 AND name = $2
	`).Args(projectKey, name)

	return getRunFilter(ctx, db, query)
}

// UpdateRunFilterOrder met à jour uniquement le champ order d'un filtre
func UpdateRunFilterOrder(ctx context.Context, db gorpmapper.SqlExecutorWithTx, projectKey string, name string, order int64) error {
	query := `
		UPDATE project_run_filter
		SET "order" = $1, last_modified = $2
		WHERE project_key = $3 AND name = $4
	`
	_, err := db.Exec(query, order, time.Now(), projectKey, name)
	return sdk.WithStack(err)
}

// RecomputeRunFilterOrder recalcule les ordres des filtres restants après suppression.
// Les filtres sont rechargés triés par order ASC, name ASC, puis réassignés de 0 à N-1.
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

// Helper: getRunFilter charge un seul filtre
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

// Helper: getRunFilters charge plusieurs filtres
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
