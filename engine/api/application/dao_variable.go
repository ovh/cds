package application

import (
	"context"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

type dbApplicationVariable struct {
	gorpmapper.SignedEntity
	ID            int64  `db:"id"`
	ApplicationID int64  `db:"application_id"`
	Name          string `db:"var_name"`
	ClearValue    string `db:"var_value"`
	CipherValue   string `db:"cipher_value" gorpmapping:"encrypted,ID,Name"`
	Type          string `db:"var_type"`
}

func (e dbApplicationVariable) Canonical() gorpmapper.CanonicalForms {
	var _ = []interface{}{e.ApplicationID, e.ID, e.Name, e.Type, e.ClearValue}
	return gorpmapper.CanonicalForms{
		"{{printf .ApplicationID}}{{printf .ID}}{{.Name}}{{.Type}}{{.ClearValue}}",
		"{{print .ApplicationID}}{{print .ID}}{{.Name}}{{.Type}}",
	}
}

func newDBApplicationVariable(v sdk.ApplicationVariable, appID int64) dbApplicationVariable {
	if sdk.NeedPlaceholder(v.Type) {
		return dbApplicationVariable{
			ID:            v.ID,
			Name:          v.Name,
			CipherValue:   v.Value,
			Type:          v.Type,
			ApplicationID: appID,
		}
	}
	return dbApplicationVariable{
		ID:            v.ID,
		Name:          v.Name,
		ClearValue:    v.Value,
		Type:          v.Type,
		ApplicationID: appID,
	}
}

func (e dbApplicationVariable) Variable() sdk.ApplicationVariable {
	if sdk.NeedPlaceholder(e.Type) {
		return sdk.ApplicationVariable{
			ID:            e.ID,
			Name:          e.Name,
			Value:         e.CipherValue,
			Type:          e.Type,
			ApplicationID: e.ApplicationID,
		}
	}

	return sdk.ApplicationVariable{
		ID:            e.ID,
		Name:          e.Name,
		Value:         e.ClearValue,
		Type:          e.Type,
		ApplicationID: e.ApplicationID,
	}
}

func loadAllVariables(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query, opts ...gorpmapping.GetOptionFunc) ([]sdk.ApplicationVariable, error) {
	var res []dbApplicationVariable
	vars := make([]sdk.ApplicationVariable, 0, len(res))

	if err := gorpmapping.GetAll(ctx, db, query, &res, opts...); err != nil {
		return nil, err
	}

	for i := range res {
		isValid, err := gorpmapping.CheckSignature(res[i], res[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "application.getAllVariables> application key %d data corrupted", res[i].ID)
			continue
		}
		vars = append(vars, res[i].Variable())
	}
	return vars, nil
}

func LoadAllVariables(ctx context.Context, db gorp.SqlExecutor, appID int64) ([]sdk.ApplicationVariable, error) {
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM application_variable
		WHERE application_id = $1
		ORDER BY application_id, var_name
	`).Args(appID)
	return loadAllVariables(ctx, db, query)
}

// LoadAllVariablesWithDecrytion Get all variable for the given application, it also decrypt all the secure content
func LoadAllVariablesWithDecrytion(ctx context.Context, db gorp.SqlExecutor, appID int64) ([]sdk.ApplicationVariable, error) {
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM application_variable
		WHERE application_id = $1
		ORDER BY var_name
	`).Args(appID)
	return loadAllVariables(ctx, db, query, gorpmapping.GetOptions.WithDecryption)
}

func loadVariable(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query, opts ...gorpmapping.GetOptionFunc) (*sdk.ApplicationVariable, error) {
	var v dbApplicationVariable
	found, err := gorpmapping.Get(ctx, db, q, &v, opts...)
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
		log.Error(ctx, "application.loadVariable> application variable %d data corrupted", v.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	res := v.Variable()
	return &res, err
}

// LoadVariable retrieve a specific variable
func LoadVariable(ctx context.Context, db gorp.SqlExecutor, appID int64, varName string) (*sdk.ApplicationVariable, error) {
	query := gorpmapping.NewQuery(`SELECT * FROM application_variable
			WHERE application_id = $1 AND var_name=$2`).Args(appID, varName)
	return loadVariable(ctx, db, query)
}

// LoadVariableWithDecryption retrieve a specific variable with decrypted content
func LoadVariableWithDecryption(ctx context.Context, db gorp.SqlExecutor, appID int64, varID int64, varName string) (*sdk.ApplicationVariable, error) {
	query := gorpmapping.NewQuery(`SELECT * FROM application_variable
			WHERE application_id = $1 AND id = $2 AND var_name=$3`).Args(appID, varID, varName)
	return loadVariable(ctx, db, query, gorpmapping.GetOptions.WithDecryption)
}

// DeleteAllVariables Delete all variables from the given application.
func DeleteAllVariables(db gorp.SqlExecutor, applicationID int64) error {
	query := `DELETE FROM application_variable
	          WHERE application_variable.application_id = $1`
	if _, err := db.Exec(query, applicationID); err != nil {
		return sdk.WithStack(err)
	}
	return nil
}

// InsertVariable Insert a new variable in the given application
func InsertVariable(db gorpmapper.SqlExecutorWithTx, appID int64, v *sdk.ApplicationVariable, u sdk.Identifiable) error {
	//Check variable name
	rx := sdk.NamePatternRegex
	if !rx.MatchString(v.Name) {
		return sdk.NewErrorFrom(sdk.ErrInvalidName, "variable name should match pattern %s", sdk.NamePattern)
	}
	dbVar := newDBApplicationVariable(*v, appID)
	err := gorpmapping.InsertAndSign(context.Background(), db, &dbVar)
	if err != nil && strings.Contains(err.Error(), "application_variable_pkey") {
		return sdk.WithStack(sdk.ErrVariableExists)
	}
	if err != nil {
		return sdk.WrapError(err, "cannot insert variable %s", v.Name)
	}

	*v = dbVar.Variable()

	ava := &sdk.ApplicationVariableAudit{
		ApplicationID: appID,
		Type:          sdk.AuditAdd,
		Author:        u.GetUsername(),
		VariableAfter: *v,
		VariableID:    v.ID,
		Versionned:    time.Now(),
	}

	if err := inserAudit(db, ava); err != nil {
		return sdk.WrapError(err, "Cannot insert audit for variable %d", v.ID)
	}
	return nil
}

// UpdateVariable Update a variable in the given application
func UpdateVariable(db gorpmapper.SqlExecutorWithTx, appID int64, variable *sdk.ApplicationVariable, variableBefore *sdk.ApplicationVariable, u sdk.Identifiable) error {
	rx := sdk.NamePatternRegex
	if !rx.MatchString(variable.Name) {
		return sdk.NewErrorFrom(sdk.ErrInvalidName, "variable name should match pattern %s", sdk.NamePattern)
	}

	dbVar := newDBApplicationVariable(*variable, appID)

	if err := gorpmapping.UpdateAndSign(context.Background(), db, &dbVar); err != nil {
		return err
	}

	*variable = dbVar.Variable()

	if variableBefore == nil && u == nil {
		return nil
	}

	ava := &sdk.ApplicationVariableAudit{
		ApplicationID:  appID,
		Type:           sdk.AuditUpdate,
		Author:         u.GetUsername(),
		VariableAfter:  *variable,
		VariableBefore: variableBefore,
		VariableID:     variable.ID,
		Versionned:     time.Now(),
	}

	if err := inserAudit(db, ava); err != nil {
		return sdk.WrapError(err, "Cannot insert audit for variable %s", variable.Name)
	}

	return nil
}

// DeleteVariable Delete a variable from the given pipeline
func DeleteVariable(db gorp.SqlExecutor, appID int64, variable *sdk.ApplicationVariable, u sdk.Identifiable) error {
	query := `DELETE FROM application_variable
		  WHERE application_variable.application_id = $1 AND application_variable.var_name = $2`
	result, err := db.Exec(query, appID, variable.Name)
	if err != nil {
		return sdk.WrapError(err, "Cannot delete variable %s", variable.Name)
	}

	rowAffected, err := result.RowsAffected()
	if err != nil {
		return sdk.WithStack(err)
	}
	if rowAffected == 0 {
		return sdk.WithStack(ErrNoVariable)
	}

	ava := &sdk.ApplicationVariableAudit{
		ApplicationID:  appID,
		Type:           sdk.AuditDelete,
		Author:         u.GetUsername(),
		VariableBefore: variable,
		VariableID:     variable.ID,
		Versionned:     time.Now(),
	}

	if err := inserAudit(db, ava); err != nil {
		return sdk.WrapError(err, "Cannot insert audit for variable %s", variable.Name)
	}
	return nil
}

// LoadAllVariablesForAppsWithDecryption load all variables from all given applications, with decryption
func LoadAllVariablesForAppsWithDecryption(ctx context.Context, db gorp.SqlExecutor, appIDs []int64) (map[int64][]sdk.ApplicationVariable, error) {
	return loadAllVariablesForApps(ctx, db, appIDs, gorpmapping.GetOptions.WithDecryption)
}

func loadAllVariablesForApps(ctx context.Context, db gorp.SqlExecutor, appsID []int64, opts ...gorpmapping.GetOptionFunc) (map[int64][]sdk.ApplicationVariable, error) {
	var res []dbApplicationVariable
	query := gorpmapping.NewQuery(`
		SELECT *
		FROM application_variable
		WHERE application_id = ANY($1)
		ORDER BY application_id
	`).Args(pq.Int64Array(appsID))
	if err := gorpmapping.GetAll(ctx, db, query, &res, opts...); err != nil {
		return nil, err
	}

	appsVars := make(map[int64][]sdk.ApplicationVariable)

	for i := range res {
		dbAppVar := res[i]
		isValid, err := gorpmapping.CheckSignature(dbAppVar, dbAppVar.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "application.loadAllVariablesForApps> application variable %d data corrupted", dbAppVar.ID)
			continue
		}
		if _, ok := appsVars[dbAppVar.ApplicationID]; !ok {
			appsVars[dbAppVar.ApplicationID] = make([]sdk.ApplicationVariable, 0)
		}
		appsVars[dbAppVar.ApplicationID] = append(appsVars[dbAppVar.ApplicationID], dbAppVar.Variable())
	}
	return appsVars, nil
}
