package project

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func loadAllVariables(db gorp.SqlExecutor, query gorpmapping.Query, opts ...gorpmapping.GetOptionFunc) ([]sdk.Variable, error) {
	var ctx = context.Background()
	var res []dbProjectVariable
	vars := make([]sdk.Variable, 0, len(res))

	if err := gorpmapping.GetAll(ctx, db, query, &res, opts...); err != nil {
		return nil, err
	}

	for i := range res {
		isValid, err := gorpmapping.CheckSignature(res[i], res[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "project.getAllVariables> project key %d data corrupted", res[i].ID)
			continue
		}
		vars = append(vars, res[i].Variable())
	}
	return vars, nil
}

// LoadAllVariables Get all variable for the given project
func LoadAllVariables(db gorp.SqlExecutor, projID int64) ([]sdk.Variable, error) {
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM project_variable
		WHERE project_id = $1
		ORDER BY var_name
			  `).Args(projID)
	return loadAllVariables(db, query)
}

// LoadAllVariablesWithDecrytion Get all variable for the given project, it also decrypt all the secure content
func LoadAllVariablesWithDecrytion(db gorp.SqlExecutor, projID int64) ([]sdk.Variable, error) {
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM project_variable
		WHERE project_id = $1
		ORDER BY var_name
			  `).Args(projID)
	return loadAllVariables(db, query, gorpmapping.GetOptions.WithDecryption)
}

func loadVariable(db gorp.SqlExecutor, q gorpmapping.Query, opts ...gorpmapping.GetOptionFunc) (*sdk.Variable, error) {
	var v dbProjectVariable
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
		log.Error(context.Background(), "project.loadVariable> project variable %d data corrupted", v.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	res := v.Variable()
	return &res, err
}

// LoadVariable retrieve a specific variable
func LoadVariable(db gorp.SqlExecutor, projID int64, varName string) (*sdk.Variable, error) {
	query := gorpmapping.NewQuery(`SELECT * FROM project_variable
			WHERE project_id = $1 AND var_name=$2`).Args(projID, varName)
	return loadVariable(db, query)
}

// LoadVariableWithDecryption retrieve a specific variable with decrypted content
func LoadVariableWithDecryption(db gorp.SqlExecutor, projID int64, varID int64, varName string) (*sdk.Variable, error) {
	query := gorpmapping.NewQuery(`SELECT * FROM project_variable
			WHERE project_id = $1 AND id = $2 AND var_name=$3`).Args(projID, varID, varName)
	return loadVariable(db, query, gorpmapping.GetOptions.WithDecryption)
}

// DeleteAllVariables Delete all variables from the given project.
func DeleteAllVariables(db gorp.SqlExecutor, projectID int64) error {
	query := `DELETE FROM project_variable
	          WHERE project_variable.project_id = $1`
	if _, err := db.Exec(query, projectID); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

// InsertVariable Insert a new variable in the given project
func InsertVariable(db gorp.SqlExecutor, projID int64, v *sdk.Variable, u sdk.Identifiable) error {
	//Check variable name
	rx := sdk.NamePatternRegex
	if !rx.MatchString(v.Name) {
		return sdk.NewError(sdk.ErrInvalidName, fmt.Errorf("Invalid variable name. It should match %s", sdk.NamePattern))
	}

	if sdk.NeedPlaceholder(v.Type) && v.Value == sdk.PasswordPlaceholder {
		return fmt.Errorf("You try to insert a placeholder for new variable %s", v.Name)
	}

	dbVar := newDBProjectVariable(*v, projID)

	err := gorpmapping.InsertAndSign(context.Background(), db, &dbVar)
	if err != nil && strings.Contains(err.Error(), "project_variable_pkey") {
		return sdk.WithStack(sdk.ErrVariableExists)

	}
	if err != nil {
		return sdk.WrapError(err, "Cannot insert variable %s", v.Name)
	}

	*v = dbVar.Variable()

	ava := &sdk.ProjectVariableAudit{
		ProjectID:     projID,
		Type:          sdk.AuditAdd,
		Author:        u.GetUsername(),
		VariableAfter: *v,
		VariableID:    v.ID,
		Versionned:    time.Now(),
	}

	if err := insertAudit(db, ava); err != nil {
		return sdk.WrapError(err, "Cannot insert audit for variable %d", v.ID)
	}
	return nil
}

// UpdateVariable Update a variable in the given project
func UpdateVariable(db gorp.SqlExecutor, projID int64, variable *sdk.Variable, variableBefore *sdk.Variable, u sdk.Identifiable) error {
	rx := sdk.NamePatternRegex
	if !rx.MatchString(variable.Name) {
		return sdk.NewError(sdk.ErrInvalidName, fmt.Errorf("Invalid variable name. It should match %s", sdk.NamePattern))
	}

	dbVar := newDBProjectVariable(*variable, projID)

	if err := gorpmapping.UpdateAndSign(context.Background(), db, &dbVar); err != nil {
		return err
	}

	*variable = dbVar.Variable()

	if variableBefore == nil && u == nil {
		return nil
	}

	ava := &sdk.ProjectVariableAudit{
		ProjectID:      projID,
		Type:           sdk.AuditUpdate,
		Author:         u.GetUsername(),
		VariableAfter:  *variable,
		VariableBefore: variableBefore,
		VariableID:     variable.ID,
		Versionned:     time.Now(),
	}

	if err := insertAudit(db, ava); err != nil {
		return sdk.WrapError(err, "Cannot insert audit for variable %s", variable.Name)
	}

	return nil
}

// DeleteVariable Delete a variable from the given pipeline
func DeleteVariable(db gorp.SqlExecutor, projID int64, variable *sdk.Variable, u sdk.Identifiable) error {
	query := `DELETE FROM project_variable
		  WHERE project_variable.project_id = $1 AND project_variable.var_name = $2`
	result, err := db.Exec(query, projID, variable.Name)
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

	ava := &sdk.ProjectVariableAudit{
		ProjectID:      projID,
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
