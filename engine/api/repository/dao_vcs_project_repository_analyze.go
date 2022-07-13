package repository

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func InsertAnalyze(ctx context.Context, db gorpmapper.SqlExecutorWithTx, analyze *sdk.ProjectRepositoryAnalyze) error {
	// Count number of analyze
	nb, err := countAnalyzeByRepo(db, analyze.ProjectRepositoryID)
	if err != nil {
		return err
	}
	if nb >= 50 {
		// Delete the oldest analyze
		if err := deleteOldestAnalyze(ctx, db, analyze.ProjectRepositoryID); err != nil {
			return err
		}
	}

	analyze.ID = sdk.UUID()
	analyze.Created = time.Now()
	analyze.LastModified = time.Now()
	if analyze.Status == "" {
		analyze.Status = sdk.RepositoryAnalyzeStatusInProgress
	}
	if err := gorpmapping.Insert(db, analyze); err != nil {
		return err
	}
	return nil
}

func UpdateAnalyze(db gorpmapper.SqlExecutorWithTx, analyze *sdk.ProjectRepositoryAnalyze) error {
	analyze.LastModified = time.Now()
	if err := gorpmapping.Update(db, analyze); err != nil {
		return err
	}
	return nil
}

func deleteOldestAnalyze(ctx context.Context, db gorpmapper.SqlExecutorWithTx, projectRepositoryID string) error {
	var analyze sdk.ProjectRepositoryAnalyze
	query := gorpmapping.NewQuery("SELECT * from project_repository_analyze WHERE project_repository_id = $1 ORDER BY created asc LIMIT 1").Args(projectRepositoryID)
	if _, err := gorpmapping.Get(ctx, db, query, &analyze); err != nil {
		return err
	}
	if err := gorpmapping.Delete(db, &analyze); err != nil {
		return err
	}
	return nil
}

func countAnalyzeByRepo(db gorp.SqlExecutor, projectRepositoryID string) (int64, error) {
	nb, err := gorpmapping.GetInt(db, gorpmapping.NewQuery("SELECT count(id) FROM project_repository_analyze WHERE project_repository_id = $1").Args(projectRepositoryID))
	if err != nil {
		return 0, err
	}
	return nb, nil
}

func ListAnalyzesByRepo(ctx context.Context, db gorp.SqlExecutor, projectRepositoryID string) ([]sdk.ProjectRepositoryAnalyze, error) {
	var analyzes []sdk.ProjectRepositoryAnalyze
	query := gorpmapping.NewQuery("SELECT * from project_repository_analyze where project_repository_id = $1 ORDER BY created DESC").Args(projectRepositoryID)
	if err := gorpmapping.GetAll(ctx, db, query, &analyzes); err != nil {
		return nil, err
	}
	return analyzes, nil
}
