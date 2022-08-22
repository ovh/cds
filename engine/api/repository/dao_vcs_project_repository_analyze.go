package repository

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func InsertAnalyze(ctx context.Context, db gorpmapper.SqlExecutorWithTx, analyze *sdk.ProjectRepositoryAnalyze) error {
	analyze.ID = sdk.UUID()
	analyze.Created = time.Now()
	analyze.LastModified = time.Now()
	if analyze.Status == "" {
		analyze.Status = sdk.RepositoryAnalyzeStatusInProgress
	}

	dbData := dbProjectRepositoryAnalyze{ProjectRepositoryAnalyze: *analyze}
	if err := gorpmapping.InsertAndSign(ctx, db, &dbData); err != nil {
		return err
	}

	*analyze = dbData.ProjectRepositoryAnalyze
	return nil
}

func UpdateAnalyze(ctx context.Context, db gorpmapper.SqlExecutorWithTx, analyze *sdk.ProjectRepositoryAnalyze) error {
	analyze.LastModified = time.Now()
	dbData := dbProjectRepositoryAnalyze{ProjectRepositoryAnalyze: *analyze}
	if err := gorpmapping.UpdateAndSign(ctx, db, &dbData); err != nil {
		return err
	}
	*analyze = dbData.ProjectRepositoryAnalyze
	return nil
}

func getAnalyze(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) (*sdk.ProjectRepositoryAnalyze, error) {
	var dbData dbProjectRepositoryAnalyze
	if _, err := gorpmapping.Get(ctx, db, query, &dbData); err != nil {
		return nil, err
	}
	if dbData.ID == "" {
		return nil, sdk.ErrNotFound
	}
	isValid, err := gorpmapping.CheckSignature(dbData, dbData.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "project_repository_analyze %d data corrupted", dbData.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &dbData.ProjectRepositoryAnalyze, nil
}

func getAllAnalyzes(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.ProjectRepositoryAnalyze, error) {
	var dbData []dbProjectRepositoryAnalyze
	if err := gorpmapping.GetAll(ctx, db, query, &dbData); err != nil {
		return nil, err
	}
	analyzes := make([]sdk.ProjectRepositoryAnalyze, 0, len(dbData))
	for _, a := range dbData {
		isValid, err := gorpmapping.CheckSignature(a, a.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "project_repository_analyze %d data corrupted", a.ID)
			continue
		}
		analyzes = append(analyzes, a.ProjectRepositoryAnalyze)
	}
	return analyzes, nil
}

func DeleteOldestAnalyze(ctx context.Context, db gorpmapper.SqlExecutorWithTx, projectRepositoryID string) error {
	query := gorpmapping.NewQuery("SELECT * from project_repository_analyze WHERE project_repository_id = $1 ORDER BY created asc LIMIT 1").Args(projectRepositoryID)
	analyze, err := getAnalyze(ctx, db, query)
	if err != nil {
		return err
	}

	dbData := dbProjectRepositoryAnalyze{ProjectRepositoryAnalyze: *analyze}
	if err := gorpmapping.Delete(db, &dbData); err != nil {
		return err
	}
	return nil
}

func CountAnalyzeByRepo(db gorp.SqlExecutor, projectRepositoryID string) (int64, error) {
	nb, err := gorpmapping.GetInt(db, gorpmapping.NewQuery("SELECT count(id) FROM project_repository_analyze WHERE project_repository_id = $1").Args(projectRepositoryID))
	if err != nil {
		return 0, err
	}
	return nb, nil
}

func LoadAllAnalyzesByRepo(ctx context.Context, db gorp.SqlExecutor, projectRepositoryID string) ([]sdk.ProjectRepositoryAnalyze, error) {
	query := gorpmapping.NewQuery("SELECT * from project_repository_analyze where project_repository_id = $1 ORDER BY created ASC").Args(projectRepositoryID)
	return getAllAnalyzes(ctx, db, query)
}

func LoadRepositoryIDsAnalysisInProgress(ctx context.Context, db gorp.SqlExecutor) ([]sdk.ProjectRepositoryAnalyze, error) {
	query := gorpmapping.NewQuery("SELECT * FROM project_repository_analyze WHERE status = $1").Args(sdk.RepositoryAnalyzeStatusInProgress)
	return getAllAnalyzes(ctx, db, query)
}

func LoadRepositoryAnalyzeById(ctx context.Context, db gorp.SqlExecutor, projectRepoID, analyzeID string) (*sdk.ProjectRepositoryAnalyze, error) {
	query := gorpmapping.NewQuery("SELECT * FROM project_repository_analyze WHERE project_repository_id = $1 AND id = $2").Args(projectRepoID, analyzeID)
	return getAnalyze(ctx, db, query)
}
