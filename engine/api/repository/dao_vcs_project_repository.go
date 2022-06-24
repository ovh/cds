package repository

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
)

func Insert(ctx context.Context, db gorpmapper.SqlExecutorWithTx, repo *sdk.ProjectRepository) error {
	repo.ID = sdk.UUID()
	repo.Created = time.Now()
	dbData := &dbProjectRepository{ProjectRepository: *repo}
	if err := gorpmapping.InsertAndSign(ctx, db, dbData); err != nil {
		return err
	}
	*repo = dbData.ProjectRepository
	return nil
}

func Update(ctx context.Context, db gorpmapper.SqlExecutorWithTx, repo *sdk.ProjectRepository) error {
	dbData := &dbProjectRepository{ProjectRepository: *repo}
	if err := gorpmapping.UpdateAndSign(ctx, db, dbData); err != nil {
		return err
	}
	*repo = dbData.ProjectRepository
	return nil
}

func Delete(db gorpmapper.SqlExecutorWithTx, vcsProjectID string, name string) error {
	_, err := db.Exec("DELETE FROM project_repository WHERE vcs_project_id = $1 AND name = $2", vcsProjectID, name)
	return sdk.WrapError(err, "cannot delete project_repository %s / %s", vcsProjectID, name)
}

func LoadRepositoryByVCSAndID(ctx context.Context, db gorp.SqlExecutor, vcsProjectID, repoID string) (*sdk.ProjectRepository, error) {
	query := gorpmapping.NewQuery(`SELECT project_repository.* FROM project_repository WHERE project_repository.vcs_project_id = $1 AND project_repository.id = $2`).Args(vcsProjectID, repoID)
	var res dbProjectRepository
	if _, err := gorpmapping.Get(ctx, db, query, &res); err != nil {
		return nil, err
	}

	isValid, err := gorpmapping.CheckSignature(res, res.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "project_repository %d / %s data corrupted", res.ID, res.Name)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &res.ProjectRepository, nil
}

func LoadRepositoryByName(ctx context.Context, db gorp.SqlExecutor, vcsProjectID string, repoName string) (*sdk.ProjectRepository, error) {
	query := gorpmapping.NewQuery(`SELECT project_repository.* FROM project_repository WHERE project_repository.vcs_project_id = $1 AND project_repository.name = $2`).Args(vcsProjectID, repoName)
	var res dbProjectRepository
	if _, err := gorpmapping.Get(ctx, db, query, &res); err != nil {
		return nil, err
	}

	isValid, err := gorpmapping.CheckSignature(res, res.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "project_repository %d / %s data corrupted", res.ID, res.Name)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &res.ProjectRepository, nil
}

func LoadAllRepositoriesByVCSProjectID(ctx context.Context, db gorp.SqlExecutor, vcsProjectID string) ([]sdk.ProjectRepository, error) {
	query := gorpmapping.NewQuery(`SELECT project_repository.* FROM project_repository WHERE project_repository.vcs_project_id = $1`).Args(vcsProjectID)
	var res []dbProjectRepository
	if err := gorpmapping.GetAll(ctx, db, query, &res); err != nil {
		return nil, err
	}

	repositories := make([]sdk.ProjectRepository, 0, len(res))
	for _, r := range res {
		isValid, err := gorpmapping.CheckSignature(r, r.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "project_repository %d / %s data corrupted", r.ID, r.Name)
			continue
		}
		repositories = append(repositories, r.ProjectRepository)
	}
	return repositories, nil
}

func LoadAllRepositories(ctx context.Context, db gorp.SqlExecutor) ([]sdk.ProjectRepository, error) {
	query := gorpmapping.NewQuery(`SELECT project_repository.* FROM project_repository`)
	var res []dbProjectRepository
	if err := gorpmapping.GetAll(ctx, db, query, &res); err != nil {
		return nil, err
	}

	repositories := make([]sdk.ProjectRepository, 0, len(res))
	for _, r := range res {
		isValid, err := gorpmapping.CheckSignature(r, r.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "project_repository %d / %s data corrupted", r.ID, r.Name)
			continue
		}
		repositories = append(repositories, r.ProjectRepository)
	}
	return repositories, nil
}
