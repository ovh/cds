package entity

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/rockbears/log"
	"github.com/rockbears/yaml"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func getEntity(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query, opts ...gorpmapping.GetOptionFunc) (*sdk.Entity, error) {
	var res dbEntity
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
		log.Error(ctx, "entity %d / %s data corrupted", res.ID, res.Name)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &res.Entity, nil
}

func getEntities(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query, opts ...gorpmapping.GetOptionFunc) ([]sdk.Entity, error) {
	var res []dbEntity
	if err := gorpmapping.GetAll(ctx, db, query, &res, opts...); err != nil {
		return nil, err
	}
	entities := make([]sdk.Entity, 0, len(res))
	for _, r := range res {
		isValid, err := gorpmapping.CheckSignature(r, r.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "entity %d / %s data corrupted", r.ID, r.Name)
			continue
		}
		entities = append(entities, r.Entity)
	}
	return entities, nil
}

func Insert(ctx context.Context, db gorpmapper.SqlExecutorWithTx, e *sdk.Entity) error {
	if e.ID == "" {
		e.ID = sdk.UUID()
	}
	e.LastUpdate = time.Now()
	dbData := &dbEntity{Entity: *e}
	if err := gorpmapping.InsertAndSign(ctx, db, dbData); err != nil {
		return err
	}
	*e = dbData.Entity
	return nil
}

func Update(ctx context.Context, db gorpmapper.SqlExecutorWithTx, e *sdk.Entity) error {
	e.LastUpdate = time.Now()
	dbData := &dbEntity{Entity: *e}
	if err := gorpmapping.UpdateAndSign(ctx, db, dbData); err != nil {
		return err
	}
	*e = dbData.Entity
	return nil
}

func Delete(_ context.Context, db gorpmapper.SqlExecutorWithTx, e *sdk.Entity) error {
	return gorpmapping.Delete(db, &dbEntity{Entity: *e})
}

// LoadByRepositoryAndBranch loads an entity by his repository, branch
func LoadByRepositoryAndBranch(ctx context.Context, db gorp.SqlExecutor, projectRepositoryID string, branch string, opts ...gorpmapping.GetOptionFunc) ([]sdk.Entity, error) {
	query := gorpmapping.NewQuery(`
		SELECT * from entity
		WHERE project_repository_id = $1 AND branch = $2`).Args(projectRepositoryID, branch)
	return getEntities(ctx, db, query, opts...)
}

// LoadByRepository loads all an entities in the given repository,
func LoadByRepository(ctx context.Context, db gorp.SqlExecutor, projectRepositoryID string, opts ...gorpmapping.GetOptionFunc) ([]sdk.Entity, error) {
	query := gorpmapping.NewQuery(`
		SELECT * from entity
		WHERE project_repository_id = $1`).Args(projectRepositoryID)
	return getEntities(ctx, db, query, opts...)
}

// LoadByRepositoryAndType loads an entity by his repository, type
func LoadByRepositoryAndType(ctx context.Context, db gorp.SqlExecutor, projectRepositoryID string, t string, opts ...gorpmapping.GetOptionFunc) ([]sdk.Entity, error) {
	query := gorpmapping.NewQuery(`
		SELECT * from entity
		WHERE project_repository_id = $1 AND type = $2`).Args(projectRepositoryID, t)
	return getEntities(ctx, db, query, opts...)
}

// LoadByTypeAndBranch loads an entity by his repository, type and branch
func LoadByTypeAndBranch(ctx context.Context, db gorp.SqlExecutor, projectRepositoryID string, t string, branch string, opts ...gorpmapping.GetOptionFunc) ([]sdk.Entity, error) {
	query := gorpmapping.NewQuery(`
		SELECT * from entity
		WHERE project_repository_id = $1 AND type = $2 AND branch = $3`).Args(projectRepositoryID, t, branch)
	return getEntities(ctx, db, query, opts...)
}

// LoadByBranchTypeName loads an entity by its repository, branch, type and name
func LoadByBranchTypeName(ctx context.Context, db gorp.SqlExecutor, projectRepositoryID string, branch string, t string, name string, opts ...gorpmapping.GetOptionFunc) (*sdk.Entity, error) {
	ctx, next := telemetry.Span(ctx, "entity.LoadByBranchTypeName")
	defer next()
	query := gorpmapping.NewQuery(`
		SELECT * from entity
		WHERE project_repository_id = $1 AND branch = $2 AND type = $3 AND name = $4`).Args(projectRepositoryID, branch, t, name)
	return getEntity(ctx, db, query, opts...)
}

// LoadByBranchTypeNameCommit loads an entity by its repository, branch, type, name and commit
func LoadByBranchTypeNameCommit(ctx context.Context, db gorp.SqlExecutor, projectRepositoryID string, branch string, t string, name string, commit string, opts ...gorpmapping.GetOptionFunc) (*sdk.Entity, error) {
	ctx, next := telemetry.Span(ctx, "entity.LoadByBranchTypeNameCommit")
	defer next()
	query := gorpmapping.NewQuery(`
		SELECT * from entity
		WHERE project_repository_id = $1 AND branch = $2 AND type = $3 AND name = $4 AND commit = $5`).Args(projectRepositoryID, branch, t, name, commit)
	return getEntity(ctx, db, query, opts...)
}

// LoadAndUnmarshalByBranchTypeName loads an entity by his repository, branch, type, name and unmarshal it
func LoadAndUnmarshalByBranchTypeName(ctx context.Context, db gorp.SqlExecutor, projectRepositoryID string, branch string, t string, name string, out interface{}, opts ...gorpmapping.GetOptionFunc) error {
	ent, err := LoadByBranchTypeName(ctx, db, projectRepositoryID, branch, t, name, opts...)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal([]byte(ent.Data), out); err != nil {
		return sdk.WrapError(err, "unable to read %s / %s @ %s", projectRepositoryID, name, branch)
	}
	return nil
}

func UnsafeLoadAllByType(_ context.Context, db gorp.SqlExecutor, t string) ([]sdk.EntityFullName, error) {
	query := `
    SELECT entity.name as name,
           vcs_project.name as vcs_name,
           project_repository.name as repo_name,
           entity.branch as branch,
           entity.project_key as project_key
    FROM entity
    JOIN project_repository ON entity.project_repository_id = project_repository.id
    JOIN vcs_project ON project_repository.vcs_project_id = vcs_project.id
    WHERE entity.type = $1
    ORDER BY entity.project_key, vcs_project.name, project_repository.name, entity.name, entity.branch
`
	var entities []sdk.EntityFullName
	if _, err := db.Select(&entities, query, t); err != nil {
		return nil, err
	}
	return entities, nil
}

func UnsafeLoadAllByTypeAndProjectKeys(_ context.Context, db gorp.SqlExecutor, t string, keys []string) ([]sdk.EntityFullName, error) {
	query := `
    SELECT entity.name as name,
           vcs_project.name as vcs_name,
           project_repository.name as repo_name,
           entity.branch as branch,
           entity.project_key as project_key
    FROM entity
    JOIN project_repository ON entity.project_repository_id = project_repository.id
    JOIN vcs_project ON project_repository.vcs_project_id = vcs_project.id
    WHERE entity.type = $1 AND entity.project_key = ANY($2)
    ORDER BY entity.project_key, vcs_project.name, project_repository.name, entity.name, entity.branch
    `
	var entities []sdk.EntityFullName
	if _, err := db.Select(&entities, query, t, pq.StringArray(keys)); err != nil {
		return nil, err
	}
	return entities, nil
}

func LoadEntityByPathAndBranch(ctx context.Context, db gorp.SqlExecutor, repositoryID string, path string, branch string) (*sdk.Entity, error) {
	ctx, next := telemetry.Span(ctx, "entity.LoadEntityByPathAndBranch")
	defer next()

	q := gorpmapping.NewQuery("SELECT * FROM entity WHERE project_repository_id = $1 AND file_path = $2 AND branch = $3").Args(repositoryID, path, branch)
	return getEntity(ctx, db, q)
}

func LoadAllUnsafe(ctx context.Context, db gorp.SqlExecutor) ([]sdk.Entity, error) {
	q := gorpmapping.NewQuery("SELECT * from entity")
	var res []dbEntity
	if err := gorpmapping.GetAll(ctx, db, q, &res); err != nil {
		return nil, err
	}
	entities := make([]sdk.Entity, 0, len(res))
	for _, r := range res {
		entities = append(entities, r.Entity)
	}
	return entities, nil
}
