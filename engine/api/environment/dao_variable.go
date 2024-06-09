package environment

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func loadAllVariables(db gorp.SqlExecutor, query gorpmapping.Query, opts ...gorpmapping.GetOptionFunc) ([]sdk.EnvironmentVariable, error) {
	var ctx = context.Background()
	var res []dbEnvironmentVariable
	vars := make([]sdk.EnvironmentVariable, 0, len(res))

	if err := gorpmapping.GetAll(ctx, db, query, &res, opts...); err != nil {
		return nil, err
	}

	for i := range res {
		isValid, err := gorpmapping.CheckSignature(res[i], res[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "environment.getAllVariables> environment key %d data corrupted", res[i].ID)
			continue
		}
		vars = append(vars, res[i].Variable())
	}
	return vars, nil
}

// LoadAllVariablesForEnvsWithDecryption load all variables for all given environments
func LoadAllVariablesForEnvsWithDecryption(ctx context.Context, db gorp.SqlExecutor, envIDS []int64) (map[int64][]sdk.EnvironmentVariable, error) {
	return loadAllVariablesForEnvs(ctx, db, envIDS, gorpmapping.GetOptions.WithDecryption)
}

func loadAllVariablesForEnvs(ctx context.Context, db gorp.SqlExecutor, envIDS []int64, opts ...gorpmapping.GetOptionFunc) (map[int64][]sdk.EnvironmentVariable, error) {
	var res []dbEnvironmentVariable
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM environment_variable
		WHERE environment_id = ANY($1)
		ORDER BY environment_id
	`).Args(pq.Int64Array(envIDS))
	if err := gorpmapping.GetAll(ctx, db, query, &res, opts...); err != nil {
		return nil, err
	}

	envsVars := make(map[int64][]sdk.EnvironmentVariable)

	for i := range res {
		dbEnvVar := res[i]
		isValid, err := gorpmapping.CheckSignature(dbEnvVar, dbEnvVar.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "environment.loadAllVariablesForEnvs> environment variable id %d data corrupted", dbEnvVar.ID)
			continue
		}
		if _, ok := envsVars[dbEnvVar.EnvironmentID]; !ok {
			envsVars[dbEnvVar.EnvironmentID] = make([]sdk.EnvironmentVariable, 0)
		}
		envsVars[dbEnvVar.EnvironmentID] = append(envsVars[dbEnvVar.EnvironmentID], dbEnvVar.Variable())
	}
	return envsVars, nil
}

// LoadAllVariables Get all variable for the given environment
func LoadAllVariables(db gorp.SqlExecutor, envID int64) ([]sdk.EnvironmentVariable, error) {
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM environment_variable
		WHERE environment_id = $1
		ORDER BY name
			  `).Args(envID)
	return loadAllVariables(db, query)
}

// LoadAllVariablesWithDecryption Get all variable for the given environment, it also decrypt all the secure content
func LoadAllVariablesWithDecryption(db gorp.SqlExecutor, envID int64) ([]sdk.EnvironmentVariable, error) {
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM environment_variable
		WHERE environment_id = $1
		ORDER BY name
			  `).Args(envID)
	return loadAllVariables(db, query, gorpmapping.GetOptions.WithDecryption)
}

func loadVariable(db gorp.SqlExecutor, q gorpmapping.Query, opts ...gorpmapping.GetOptionFunc) (*sdk.EnvironmentVariable, error) {
	var v dbEnvironmentVariable
	found, err := gorpmapping.Get(context.Background(), db, q, &v, opts...)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	isValid, err := gorpmapping.CheckSignature(v, v.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(context.Background(), "environment.loadVariable> environment variable %d data corrupted", v.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	res := v.Variable()
	return &res, err
}

// LoadVariable retrieve a specific variable
func LoadVariable(db gorp.SqlExecutor, envID int64, varName string) (*sdk.EnvironmentVariable, error) {
	query := gorpmapping.NewQuery(`SELECT * FROM environment_variable
			WHERE environment_id = $1 AND name=$2`).Args(envID, varName)
	return loadVariable(db, query)
}

// LoadVariableWithDecryption retrieve a specific variable with decrypted content
func LoadVariableWithDecryption(db gorp.SqlExecutor, envID int64, varID int64, varName string) (*sdk.EnvironmentVariable, error) {
	query := gorpmapping.NewQuery(`SELECT * FROM environment_variable
			WHERE environment_id = $1 AND id = $2 AND name=$3`).Args(envID, varID, varName)
	return loadVariable(db, query, gorpmapping.GetOptions.WithDecryption)
}

// DeleteAllVariables Delete all variables from the given environment.
func DeleteAllVariables(db gorp.SqlExecutor, environmentID int64) error {
	query := `DELETE FROM environment_variable
	          WHERE environment_variable.environment_id = $1`
	if _, err := db.Exec(query, environmentID); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

// InsertVariable Insert a new variable in the given environment
func InsertVariable(db gorpmapper.SqlExecutorWithTx, envID int64, v *sdk.EnvironmentVariable, u sdk.Identifiable) error {
	//Check variable name
	rx := sdk.NamePatternRegex
	if !rx.MatchString(v.Name) {
		return sdk.NewErrorFrom(sdk.ErrInvalidName, "variable name should match %s", sdk.NamePattern)
	}

	if sdk.NeedPlaceholder(v.Type) && v.Value == sdk.PasswordPlaceholder {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "you try to insert a placeholder for new variable %s", v.Name)
	}

	dbVar := newdbEnvironmentVariable(*v, envID)
	if err := gorpmapping.InsertAndSign(context.Background(), db, &dbVar); err != nil {
		return sdk.WrapError(err, "cannot insert variable %s", v.Name)
	}

	*v = dbVar.Variable()

	ava := &sdk.EnvironmentVariableAudit{
		EnvironmentID: envID,
		Type:          sdk.AuditAdd,
		Author:        u.GetUsername(),
		VariableAfter: *v,
		VariableID:    v.ID,
		Versionned:    time.Now(),
	}

	if err := insertAudit(db, ava); err != nil {
		return sdk.WrapError(err, "cannot insert audit for variable %d", v.ID)
	}
	return nil
}

// UpdateVariable Update a variable in the given environment
func UpdateVariable(db gorpmapper.SqlExecutorWithTx, envID int64, variable *sdk.EnvironmentVariable, variableBefore *sdk.EnvironmentVariable, u sdk.Identifiable) error {
	rx := sdk.NamePatternRegex
	if !rx.MatchString(variable.Name) {
		return sdk.NewErrorFrom(sdk.ErrInvalidName, "variable name should match %s", sdk.NamePattern)
	}

	dbVar := newdbEnvironmentVariable(*variable, envID)

	if err := gorpmapping.UpdateAndSign(context.Background(), db, &dbVar); err != nil {
		return err
	}

	*variable = dbVar.Variable()

	if variableBefore == nil && u == nil {
		return nil
	}

	ava := &sdk.EnvironmentVariableAudit{
		EnvironmentID:  envID,
		Type:           sdk.AuditUpdate,
		Author:         u.GetUsername(),
		VariableAfter:  *variable,
		VariableBefore: variableBefore,
		VariableID:     variable.ID,
		Versionned:     time.Now(),
	}

	if err := insertAudit(db, ava); err != nil {
		return sdk.WrapError(err, "cannot insert audit for variable %s", variable.Name)
	}

	return nil
}

// DeleteVariable Delete a variable from the given pipeline
func DeleteVariable(db gorp.SqlExecutor, envID int64, variable *sdk.EnvironmentVariable, u sdk.Identifiable) error {
	query := `DELETE FROM environment_variable
		  WHERE environment_variable.environment_id = $1 AND environment_variable.name = $2`
	result, err := db.Exec(query, envID, variable.Name)
	if err != nil {
		return sdk.WrapError(err, "Cannot delete variable %s", variable.Name)
	}

	rowAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowAffected == 0 {
		return sdk.ErrNotFound
	}

	ava := &sdk.EnvironmentVariableAudit{
		EnvironmentID:  envID,
		Type:           sdk.AuditDelete,
		Author:         u.GetUsername(),
		VariableBefore: variable,
		VariableID:     variable.ID,
		Versionned:     time.Now(),
	}

	if err := insertAudit(db, ava); err != nil {
		return sdk.WrapError(err, "Cannot insert audit for variable %s", variable.Name)
	}
	return nil
}
