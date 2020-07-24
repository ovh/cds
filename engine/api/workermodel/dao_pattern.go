package workermodel

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// InsertPattern in database.
func InsertPattern(db gorp.SqlExecutor, modelPattern *sdk.ModelPattern) error {
	return gorpmapping.Insert(db, modelPattern)
}

// UpdatePattern in database.
func UpdatePattern(db gorp.SqlExecutor, modelPattern *sdk.ModelPattern) error {
	return gorpmapping.Update(db, modelPattern)
}

// DeletePatternByID removes from database the worker model pattern.
func DeletePatternByID(db gorp.SqlExecutor, patternID int64) error {
	_, err := db.Exec("DELETE FROM worker_model_pattern WHERE id = $1", patternID)
	return sdk.WrapError(err, "unable to remove worker model pattern with id %d", patternID)
}

// LoadPatterns retrieves model patterns from database.
func LoadPatterns(ctx context.Context, db gorp.SqlExecutor) ([]sdk.ModelPattern, error) {
	ps := []sdk.ModelPattern{}

	query := gorpmapping.NewQuery(`
    SELECT *
    FROM worker_model_pattern
    ORDER BY name
  `)

	if err := gorpmapping.GetAll(ctx, db, query, &ps); err != nil {
		return nil, sdk.WrapError(err, "cannot load worker model patterns")
	}

	return ps, nil
}

// LoadPatternByNameAndType retrieves model patterns from database given its name and type.
func LoadPatternByNameAndType(ctx context.Context, db gorp.SqlExecutor, patternType, patternName string) (*sdk.ModelPattern, error) {
	var p sdk.ModelPattern

	query := gorpmapping.NewQuery(`
    SELECT *
    FROM worker_model_pattern
    WHERE name = $1 AND type = $2
  `).Args(patternName, patternType)

	found, err := gorpmapping.Get(ctx, db, query, &p)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot load worker model pattern with type %s and name %s", patternType, patternName)
	}
	if !found {
		return nil, sdk.NewErrorFrom(sdk.ErrNotFound, "no worker model pattern found with type %s and name %s", patternType, patternName)
	}

	return &p, nil
}
