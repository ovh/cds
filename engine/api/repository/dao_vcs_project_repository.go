package repository

import (
	"context"
	"github.com/ovh/cds/sdk/telemetry"
	"strings"
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
	_, err := db.Exec("DELETE FROM project_repository WHERE vcs_project_id = $1 AND name = $2", vcsProjectID, strings.ToLower(name))
	return sdk.WrapError(err, "cannot delete project_repository %s / %s", vcsProjectID, name)
}

func getRepository(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query, opts ...gorpmapping.GetOptionFunc) (*sdk.ProjectRepository, error) {
	var res dbProjectRepository
	found, err := gorpmapping.Get(ctx, db, query, &res, opts...)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
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

func getRepositories(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query, opts ...gorpmapping.GetOptionFunc) ([]sdk.ProjectRepository, error) {
	var res []dbProjectRepository
	if err := gorpmapping.GetAll(ctx, db, query, &res, opts...); err != nil {
		return nil, err
	}

	repos := make([]sdk.ProjectRepository, 0, len(res))
	for _, r := range res {
		isValid, err := gorpmapping.CheckSignature(r, r.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "project_repository %d / %s data corrupted", r.ID, r.Name)
			continue
		}
		repos = append(repos, r.ProjectRepository)
	}

	return repos, nil
}

func LoadRepositoryByVCSAndID(ctx context.Context, db gorp.SqlExecutor, vcsProjectID, repoID string, opts ...gorpmapping.GetOptionFunc) (*sdk.ProjectRepository, error) {
	ctx, next := telemetry.Span(ctx, "repository.LoadRepositoryByVCSAndID")
	defer next()
	query := gorpmapping.NewQuery(`SELECT project_repository.* FROM project_repository WHERE project_repository.vcs_project_id = $1 AND project_repository.id = $2`).Args(vcsProjectID, repoID)
	repo, err := getRepository(ctx, db, query, opts...)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to get repository %s", repo.ID)
	}
	return repo, nil
}

func LoadRepositoryByName(ctx context.Context, db gorp.SqlExecutor, vcsProjectID string, repoName string, opts ...gorpmapping.GetOptionFunc) (*sdk.ProjectRepository, error) {
	ctx, next := telemetry.Span(ctx, "repository.LoadRepositoryByName")
	defer next()
	query := gorpmapping.NewQuery(`SELECT project_repository.* FROM project_repository WHERE project_repository.vcs_project_id = $1 AND project_repository.name = $2`).Args(vcsProjectID, strings.ToLower(repoName))
	repo, err := getRepository(ctx, db, query, opts...)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to get repository %s from vcs %s", repoName, vcsProjectID)
	}
	return repo, nil
}

func LoadByNameWithoutVCSServer(ctx context.Context, db gorp.SqlExecutor, repoName string) ([]sdk.ProjectRepository, error) {
	ctx, next := telemetry.Span(ctx, "repository.LoadByNameWithoutVCSServer")
	defer next()
	query := gorpmapping.NewQuery(`SELECT project_repository.* FROM project_repository WHERE project_repository.name = $1`).Args(strings.ToLower(repoName))
	repos, err := getRepositories(ctx, db, query)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to get repositories %s", repoName)
	}
	return repos, nil
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

func LoadRepositoryByID(ctx context.Context, db gorp.SqlExecutor, id string, opts ...gorpmapping.GetOptionFunc) (*sdk.ProjectRepository, error) {
	ctx, next := telemetry.Span(ctx, "repository.LoadRepositoryByID")
	defer next()
	query := gorpmapping.NewQuery(`SELECT project_repository.* FROM project_repository WHERE id = $1`).Args(id)
	repo, err := getRepository(ctx, db, query, opts...)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to get repository %s", id)
	}
	return repo, nil
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
