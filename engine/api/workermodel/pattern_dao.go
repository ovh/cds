package workermodel

import (
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

const patternColumns = `
	worker_model_pattern.id,
	worker_model_pattern.name,
	worker_model_pattern.type`

// InsertPattern in database.
func InsertPattern(db gorp.SqlExecutor, modelPattern *sdk.ModelPattern) error {
	dbmodelPattern := workerModelPattern(*modelPattern)
	if err := db.Insert(&dbmodelPattern); err != nil {
		return sdk.WithStack(err)
	}
	*modelPattern = sdk.ModelPattern(dbmodelPattern)
	return nil
}

// UpdatePattern in database.
func UpdatePattern(db gorp.SqlExecutor, modelPattern *sdk.ModelPattern) error {
	dbmodelPattern := workerModelPattern(*modelPattern)
	if _, err := db.Update(&dbmodelPattern); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

// DeletePattern removes from database the worker model pattern.
func DeletePattern(db gorp.SqlExecutor, ID int64) error {
	wmp := workerModelPattern(sdk.ModelPattern{ID: ID})
	count, err := db.Delete(&wmp)
	if err != nil {
		return sdk.WithStack(err)
	}
	if count == 0 {
		return sdk.WithStack(sdk.ErrNotFound)
	}
	return nil
}

// LoadPatterns retrieves model patterns from database.
func LoadPatterns(db gorp.SqlExecutor) ([]sdk.ModelPattern, error) {
	var wmPatterns []workerModelPattern
	query := fmt.Sprintf(`SELECT %s from WORKER_MODEL_PATTERN ORDER BY name`, patternColumns)
	if _, err := db.Select(&wmPatterns, query); err != nil {
		return nil, sdk.WithStack(err)
	}

	workerModelPatterns := make([]sdk.ModelPattern, len(wmPatterns))
	for i := range wmPatterns {
		if err := wmPatterns[i].PostGet(db); err != nil {
			return nil, sdk.WithStack(err)
		}
		workerModelPatterns[i] = sdk.ModelPattern(wmPatterns[i])
	}
	return workerModelPatterns, nil
}

// LoadPatternByName retrieves model patterns from database given its name and type.
func LoadPatternByName(db gorp.SqlExecutor, patternType, name string) (*sdk.ModelPattern, error) {
	var wmp workerModelPattern
	query := fmt.Sprintf(`SELECT %s FROM worker_model_pattern WHERE name = $1 AND type = $2`, patternColumns)
	if err := db.SelectOne(&wmp, query, name, patternType); err != nil {
		return nil, sdk.WithStack(err)
	}
	workerModelPattern := sdk.ModelPattern(wmp)
	return &workerModelPattern, nil
}
