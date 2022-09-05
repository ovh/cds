package entity

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
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

// LoadByType loads an entity by his repository, type
func LoadByType(ctx context.Context, db gorp.SqlExecutor, projectRepositoryID string, t string, opts ...gorpmapping.GetOptionFunc) ([]sdk.Entity, error) {
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

// LoadByBranchTypeName loads an entity by his repository, branch, type and name
func LoadByBranchTypeName(ctx context.Context, db gorp.SqlExecutor, projectRepositoryID string, branch string, t string, name string, opts ...gorpmapping.GetOptionFunc) (*sdk.Entity, error) {
	query := gorpmapping.NewQuery(`
		SELECT * from entity
		WHERE project_repository_id $1 AND branch = $2 AND type = $3 AND name = $4`).Args(projectRepositoryID, branch, t, name)
	return getEntity(ctx, db, query, opts...)
}
