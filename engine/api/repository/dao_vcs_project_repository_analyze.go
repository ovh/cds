package repository

import (
	"context"
	"time"

	"github.com/ovh/cds/sdk/telemetry"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func InsertAnalysis(ctx context.Context, db gorpmapper.SqlExecutorWithTx, analysis *sdk.ProjectRepositoryAnalysis) error {
	analysis.ID = sdk.UUID()
	analysis.Created = time.Now()
	analysis.LastModified = time.Now()
	if analysis.Status == "" {
		analysis.Status = sdk.RepositoryAnalysisStatusInProgress
	}

	dbData := dbProjectRepositoryAnalysis{ProjectRepositoryAnalysis: *analysis}
	if err := gorpmapping.InsertAndSign(ctx, db, &dbData); err != nil {
		return err
	}

	if dbData.Data.Initiator == nil {
		dbData.Data.Initiator = &sdk.V2WorkflowRunInitiator{
			UserID: dbData.Data.DeprecatedCDSUserID,
		}
	}
	*analysis = dbData.ProjectRepositoryAnalysis
	return nil
}

func UpdateAnalysis(ctx context.Context, db gorpmapper.SqlExecutorWithTx, analysis *sdk.ProjectRepositoryAnalysis) error {
	analysis.LastModified = time.Now()
	dbData := dbProjectRepositoryAnalysis{ProjectRepositoryAnalysis: *analysis}
	if err := gorpmapping.UpdateAndSign(ctx, db, &dbData); err != nil {
		return err
	}
	if dbData.Data.Initiator == nil {
		dbData.Data.Initiator = &sdk.V2WorkflowRunInitiator{
			UserID: dbData.Data.DeprecatedCDSUserID,
		}
	}
	*analysis = dbData.ProjectRepositoryAnalysis
	return nil
}

func getAnalysis(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) (*sdk.ProjectRepositoryAnalysis, error) {
	var dbData dbProjectRepositoryAnalysis
	if _, err := gorpmapping.Get(ctx, db, query, &dbData); err != nil {
		return nil, err
	}
	if dbData.ID == "" {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	isValid, err := gorpmapping.CheckSignature(dbData, dbData.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "project_repository_analysis %d data corrupted", dbData.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	if dbData.Data.Initiator == nil {
		dbData.Data.Initiator = &sdk.V2WorkflowRunInitiator{
			UserID: dbData.Data.DeprecatedCDSUserID,
		}
	}
	return &dbData.ProjectRepositoryAnalysis, nil
}

func getAnalyses(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.ProjectRepositoryAnalysis, error) {
	var dbData []dbProjectRepositoryAnalysis
	if err := gorpmapping.GetAll(ctx, db, query, &dbData); err != nil {
		return nil, err
	}
	analyses := make([]sdk.ProjectRepositoryAnalysis, 0, len(dbData))
	for _, a := range dbData {
		isValid, err := gorpmapping.CheckSignature(a, a.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "project_repository_analysis %d data corrupted", a.ID)
			continue
		}
		analyses = append(analyses, a.ProjectRepositoryAnalysis)
	}
	return analyses, nil
}

func DeleteOldestAnalysis(ctx context.Context, db gorpmapper.SqlExecutorWithTx, projectRepositoryID string) error {
	query := gorpmapping.NewQuery("SELECT * from project_repository_analysis WHERE project_repository_id = $1 ORDER BY created asc LIMIT 1").Args(projectRepositoryID)
	analysis, err := getAnalysis(ctx, db, query)
	if err != nil {
		return err
	}

	dbData := dbProjectRepositoryAnalysis{ProjectRepositoryAnalysis: *analysis}
	if err := gorpmapping.Delete(db, &dbData); err != nil {
		return err
	}
	return nil
}

func CountAnalysesByRepo(db gorp.SqlExecutor, projectRepositoryID string) (int64, error) {
	nb, err := gorpmapping.GetInt(db, gorpmapping.NewQuery("SELECT count(id) FROM project_repository_analysis WHERE project_repository_id = $1").Args(projectRepositoryID))
	if err != nil {
		return 0, err
	}
	return nb, nil
}

func LoadAnalysesByRepo(ctx context.Context, db gorp.SqlExecutor, projectRepositoryID string) ([]sdk.ProjectRepositoryAnalysis, error) {
	query := gorpmapping.NewQuery("SELECT * from project_repository_analysis where project_repository_id = $1 ORDER BY created ASC").Args(projectRepositoryID)
	return getAnalyses(ctx, db, query)
}

func LoadRepositoryIDsAnalysisInProgress(ctx context.Context, db gorp.SqlExecutor) ([]sdk.ProjectRepositoryAnalysis, error) {
	query := gorpmapping.NewQuery("SELECT * FROM project_repository_analysis WHERE status = $1 LIMIT 100").Args(sdk.RepositoryAnalysisStatusInProgress)
	return getAnalyses(ctx, db, query)
}

func LoadRepositoryAnalysisById(ctx context.Context, db gorp.SqlExecutor, projectRepoID, analysisID string) (*sdk.ProjectRepositoryAnalysis, error) {
	ctx, next := telemetry.Span(ctx, "repository.LoadRepositoryAnalysisById")
	defer next()
	query := gorpmapping.NewQuery("SELECT * FROM project_repository_analysis WHERE project_repository_id = $1 AND id = $2").Args(projectRepoID, analysisID)
	return getAnalysis(ctx, db, query)
}
