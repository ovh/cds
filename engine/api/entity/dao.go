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

// LoadByRepositoryAndRef loads an entity by his repository, ref
func LoadByRepositoryAndRefAndCommit(ctx context.Context, db gorp.SqlExecutor, projectRepositoryID string, ref string, commit string, opts ...gorpmapping.GetOptionFunc) ([]sdk.Entity, error) {
	query := gorpmapping.NewQuery(`
		SELECT * from entity
		WHERE project_repository_id = $1 AND ref = $2 AND commit = $3
		ORDER BY name ASC
	`).Args(projectRepositoryID, ref, commit)
	return getEntities(ctx, db, query, opts...)
}

// LoadByRepository loads all an entities in the given repository,
func LoadByRepository(ctx context.Context, db gorp.SqlExecutor, projectRepositoryID string, opts ...gorpmapping.GetOptionFunc) ([]sdk.Entity, error) {
	query := gorpmapping.NewQuery(`
		SELECT * from entity
		WHERE project_repository_id = $1`).Args(projectRepositoryID)
	return getEntities(ctx, db, query, opts...)
}

// LoadByTypeAndRef loads an entity by his repository, type and ref
func LoadByTypeAndRefCommit(ctx context.Context, db gorp.SqlExecutor, projectRepositoryID string, t string, ref string, commit string, opts ...gorpmapping.GetOptionFunc) ([]sdk.Entity, error) {
	query := gorpmapping.NewQuery(`
		SELECT * from entity
		WHERE project_repository_id = $1 AND type = $2 AND ref = $3 AND commit = $4`).Args(projectRepositoryID, t, ref, commit)
	return getEntities(ctx, db, query, opts...)
}

// LoadByRefTypeNameCommit loads an entity by its repository, ref, type, name and commit
func LoadByRefTypeNameCommit(ctx context.Context, db gorp.SqlExecutor, projectRepositoryID string, ref string, entityType string, name string, commit string, opts ...gorpmapping.GetOptionFunc) (*sdk.Entity, error) {
	ctx, next := telemetry.Span(ctx, "entity.LoadByRefTypeNameCommit")
	defer next()
	query := gorpmapping.NewQuery(`
		SELECT * from entity
		WHERE project_repository_id = $1 AND ref = $2 AND type = $3 AND name = $4 AND commit = $5`).Args(projectRepositoryID, ref, entityType, name, commit)
	return getEntity(ctx, db, query, opts...)
}

// LoadAndUnmarshalByRefTypeName loads an entity by his repository, ref, type, name and unmarshal it
func LoadAndUnmarshalByRefTypeName(ctx context.Context, db gorp.SqlExecutor, projectRepositoryID string, ref string, commit string, t string, name string, out interface{}, opts ...gorpmapping.GetOptionFunc) error {
	ent, err := LoadByRefTypeNameCommit(ctx, db, projectRepositoryID, ref, t, name, commit, opts...)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal([]byte(ent.Data), out); err != nil {
		return sdk.WrapError(err, "unable to read %s / %s @ %s", projectRepositoryID, name, ref)
	}
	return nil
}

func UnsafeLoadAllByType(_ context.Context, db gorp.SqlExecutor, t string) ([]sdk.EntityFullName, error) {
	query := `
    SELECT entity.name as name,
           vcs_project.name as vcs_name,
           project_repository.name as repo_name,
           entity.ref as ref,
           entity.project_key as project_key
    FROM entity
    JOIN project_repository ON entity.project_repository_id = project_repository.id
    JOIN vcs_project ON project_repository.vcs_project_id = vcs_project.id
    WHERE entity.type = $1 AND commit = 'HEAD'
    ORDER BY entity.project_key, vcs_project.name, project_repository.name, entity.name, entity.ref
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
           entity.ref as ref,
           entity.project_key as project_key
    FROM entity
    JOIN project_repository ON entity.project_repository_id = project_repository.id
    JOIN vcs_project ON project_repository.vcs_project_id = vcs_project.id
    WHERE entity.type = $1 AND entity.project_key = ANY($2) AND commit = 'HEAD'
    ORDER BY entity.project_key, vcs_project.name, project_repository.name, entity.name, entity.ref
  `
	var entities []sdk.EntityFullName
	if _, err := db.Select(&entities, query, t, pq.StringArray(keys)); err != nil {
		return nil, err
	}
	return entities, nil
}

func LoadEntityByPathAndRefAndCommit(ctx context.Context, db gorp.SqlExecutor, repositoryID string, path string, ref string, commit string) (*sdk.Entity, error) {
	ctx, next := telemetry.Span(ctx, "entity.LoadEntityByPathAndRef")
	defer next()

	q := gorpmapping.NewQuery("SELECT * FROM entity WHERE project_repository_id = $1 AND file_path = $2 AND ref = $3 AND commit = $4").Args(repositoryID, path, ref, commit)
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
