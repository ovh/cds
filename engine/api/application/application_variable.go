package application

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/keys"
	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
)

var (
	// ErrNoVariable when request requires specific variable in the applicatoin
	ErrNoVariable = fmt.Errorf("variable not in the application")
)

type structarg struct {
	clearsecret   bool
	encryptsecret bool
}

// FuncArg defines the base type for functional argument of application helpers
type FuncArg func(args *structarg)

// WithClearPassword is a function argument to GetAllVariable
func WithClearPassword() FuncArg {
	return func(args *structarg) {
		args.clearsecret = true
	}
}

// GetVariableAudit Get variable audit for the given application
// Deprecated
func GetVariableAudit(db gorp.SqlExecutor, key, appName string) ([]sdk.VariableAudit, error) {
	audits := []sdk.VariableAudit{}
	query := `
		SELECT application_variable_audit_old.id, application_variable_audit_old.versionned, application_variable_audit_old.data, application_variable_audit_old.author
		FROM application_variable_audit_old
		JOIN application ON application.id = application_variable_audit_old.application_id
		JOIN project ON project.id = application.project_id
		WHERE application.name = $1 AND project.projectkey = $2
		ORDER BY application_variable_audit_old.versionned DESC
	`
	rows, err := db.Query(query, appName, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var audit sdk.VariableAudit
		var data string
		err := rows.Scan(&audit.ID, &audit.Versionned, &data, &audit.Author)
		if err != nil {
			return nil, err
		}
		var vars []sdk.Variable
		err = json.Unmarshal([]byte(data), &vars)
		if err != nil {
			return nil, err
		}
		audit.Variables = vars
		for i := range audit.Variables {
			v := &audit.Variables[i]
			if sdk.NeedPlaceholder(v.Type) {
				v.Value = sdk.PasswordPlaceholder
			}
		}

		audits = append(audits, audit)
	}
	return audits, nil
}

// GetAllVariable Get all variable for the given application
func GetAllVariable(db gorp.SqlExecutor, key, appName string, args ...FuncArg) ([]sdk.Variable, error) {
	c := structarg{}
	for _, f := range args {
		f(&c)
	}

	variables := []sdk.Variable{}
	query := `SELECT application_variable.id, application_variable.var_name, application_variable.var_value,
						application_variable.cipher_value, application_variable.var_type
	          FROM application_variable
	          JOIN application ON application.id = application_variable.application_id
	          JOIN project ON project.id = application.project_id
	          WHERE application.name = $1 AND project.projectKey = $2
	          ORDER BY var_name`
	rows, err := db.Query(query, appName, key)
	if err != nil {
		return variables, err
	}
	defer rows.Close()
	for rows.Next() {
		var v sdk.Variable
		var typeVar string
		var clearVal sql.NullString
		var cipherVal []byte
		err = rows.Scan(&v.ID, &v.Name, &clearVal, &cipherVal, &typeVar)
		if err != nil {
			return nil, err
		}
		v.Type = typeVar

		if c.encryptsecret && sdk.NeedPlaceholder(v.Type) {
			v.Value = string(cipherVal)
		} else {
			v.Value, err = secret.DecryptS(v.Type, clearVal, cipherVal, c.clearsecret)
		}

		if err != nil {
			return nil, err
		}

		variables = append(variables, v)
	}
	return variables, err
}

// LoadVariableByID retrieve a specific variable
func LoadVariableByID(db gorp.SqlExecutor, appID int64, varID int64, fargs ...FuncArg) (*sdk.Variable, error) {
	c := structarg{}
	for _, f := range fargs {
		f(&c)
	}

	query := `SELECT id, var_name, var_value, var_type, cipher_value FROM application_variable
			WHERE application_id = $1 AND id = $2`

	var v sdk.Variable
	var value sql.NullString
	var cipher []byte
	if err := db.QueryRow(query, appID, varID).Scan(&v.ID, &v.Name, &value, &v.Type, &cipher); err != nil {
		return nil, err
	}

	var errC error
	v.Value, errC = secret.DecryptS(v.Type, value, cipher, c.clearsecret)
	return &v, errC
}

// LoadVariable retrieve a specific variable
func LoadVariable(db gorp.SqlExecutor, appID int64, varName string, fargs ...FuncArg) (*sdk.Variable, error) {
	c := structarg{}
	for _, f := range fargs {
		f(&c)
	}

	query := `SELECT id, var_name, var_value, var_type, cipher_value FROM application_variable
			WHERE application_id = $1 AND var_name = $2`

	var v sdk.Variable
	var value sql.NullString
	var cipher []byte
	if err := db.QueryRow(query, appID, varName).Scan(&v.ID, &v.Name, &value, &v.Type, &cipher); err != nil {
		return nil, err
	}
	var errC error
	v.Value, errC = secret.DecryptS(v.Type, value, cipher, c.clearsecret)
	return &v, errC
}

// GetAllVariableByID Get all variable for the given application
func GetAllVariableByID(db gorp.SqlExecutor, applicationID int64, fargs ...FuncArg) ([]sdk.Variable, error) {
	c := structarg{}
	for _, f := range fargs {
		f(&c)
	}

	variables := []sdk.Variable{}
	query := `SELECT application_variable.id, application_variable.var_name, application_variable.var_value, application_variable.cipher_value, application_variable.var_type
	          FROM application_variable
	          WHERE application_variable.application_id = $1
	          ORDER BY var_name`
	rows, err := db.Query(query, applicationID)
	if err != nil {
		return variables, err
	}
	defer rows.Close()
	for rows.Next() {
		var v sdk.Variable
		var typeVar string
		var clearVal sql.NullString
		var cipherVal []byte
		err = rows.Scan(&v.ID, &v.Name, &clearVal, &cipherVal, &typeVar)
		if err != nil {
			return nil, err
		}
		v.Type = typeVar
		v.Value, err = secret.DecryptS(v.Type, clearVal, cipherVal, c.clearsecret)
		if err != nil {
			return nil, err
		}
		variables = append(variables, v)
	}
	return variables, err
}

// InsertVariable Insert a new variable in the given application
func InsertVariable(db gorp.SqlExecutor, store cache.Store, app *sdk.Application, variable sdk.Variable, u *sdk.User) error {
	//Check variable name
	rx := sdk.NamePatternRegex
	if !rx.MatchString(variable.Name) {
		return sdk.NewError(sdk.ErrInvalidName, fmt.Errorf("Invalid variable name. It should match %s", sdk.NamePattern))
	}

	if sdk.NeedPlaceholder(variable.Type) && variable.Value == sdk.PasswordPlaceholder {
		return fmt.Errorf("You try to insert a placeholder for new variable %s", variable.Name)
	}

	clear, cipher, err := secret.EncryptS(variable.Type, variable.Value)
	if err != nil {
		return sdk.WrapError(err, "InsertVariable> Cannot encrypt secret")
	}

	query := `INSERT INTO application_variable(application_id, var_name, var_value, cipher_value, var_type)
		  VALUES($1, $2, $3, $4, $5) RETURNING id`
	if err := db.QueryRow(query, app.ID, variable.Name, clear, cipher, string(variable.Type)).Scan(&variable.ID); err != nil && strings.Contains(err.Error(), "application_variable_pkey") {
		return sdk.ErrVariableExists
	}
	if err != nil {
		return sdk.WrapError(err, "InsertVariable> Cannot insert variable %s", variable.Name)
	}

	ava := &sdk.ApplicationVariableAudit{
		ApplicationID: app.ID,
		Type:          sdk.AuditAdd,
		Author:        u.Username,
		VariableAfter: &variable,
		VariableID:    variable.ID,
		Versionned:    time.Now(),
	}

	if err := inserAudit(db, ava); err != nil {
		return sdk.WrapError(err, "InsertVariable> Cannot insert audit for variable %d", variable.ID)
	}

	return UpdateLastModified(db, store, app, u)
}

// UpdateVariable Update a variable in the given application
func UpdateVariable(db gorp.SqlExecutor, store cache.Store, app *sdk.Application, variable *sdk.Variable, variableBefore *sdk.Variable, u *sdk.User) error {

	rx := sdk.NamePatternRegex
	if !rx.MatchString(variable.Name) {
		return sdk.NewError(sdk.ErrInvalidName, fmt.Errorf("Invalid variable name. It should match %s", sdk.NamePattern))
	}

	if sdk.NeedPlaceholder(variable.Type) && variable.Value == sdk.PasswordPlaceholder {
		variable.Value = variableBefore.Value
	}
	clear, cipher, err := secret.EncryptS(variable.Type, variable.Value)
	if err != nil {
		return sdk.WrapError(err, "UpdateVariable> Cannot encrypt secret %s", variable.Name)
	}

	query := `UPDATE application_variable SET var_name= $1, var_value=$2, cipher_value=$3, var_type = $5 WHERE id = $4`
	result, err := db.Exec(query, variable.Name, clear, cipher, variable.ID, variable.Type)
	if err != nil {
		return sdk.WrapError(err, "Cannot update variable %s", variable.Name)
	}
	rowAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowAffected == 0 {
		return ErrNoVariable
	}

	ava := &sdk.ApplicationVariableAudit{
		ApplicationID:  app.ID,
		Type:           sdk.AuditUpdate,
		Author:         u.Username,
		VariableAfter:  variable,
		VariableBefore: variableBefore,
		VariableID:     variable.ID,
		Versionned:     time.Now(),
	}

	if err := inserAudit(db, ava); err != nil {
		return sdk.WrapError(err, "UpdateVariable> Cannot insert audit for variable %s", variable.Name)
	}

	// Update application
	return UpdateLastModified(db, store, app, u)
}

// DeleteVariable Delete a variable from the given pipeline
func DeleteVariable(db gorp.SqlExecutor, store cache.Store, app *sdk.Application, variable *sdk.Variable, u *sdk.User) error {
	query := `DELETE FROM application_variable
		  WHERE application_variable.application_id = $1 AND application_variable.var_name = $2`
	result, err := db.Exec(query, app.ID, variable.Name)
	if err != nil {
		return sdk.WrapError(err, "DeleteVariable> Cannot delete variable %s", variable.Name)
	}

	rowAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowAffected == 0 {
		return ErrNoVariable
	}

	ava := &sdk.ApplicationVariableAudit{
		ApplicationID:  app.ID,
		Type:           sdk.AuditDelete,
		Author:         u.Username,
		VariableBefore: variable,
		VariableID:     variable.ID,
		Versionned:     time.Now(),
	}

	if err := inserAudit(db, ava); err != nil {
		return sdk.WrapError(err, "DeleteVariable> Cannot insert audit for variable %s", variable.Name)
	}

	return UpdateLastModified(db, store, app, u)
}

// DeleteAllVariable Delete all variables from the given pipeline
func DeleteAllVariable(db gorp.SqlExecutor, applicationID int64) error {
	query := `DELETE FROM application_variable
	          WHERE application_variable.application_id = $1`
	_, err := db.Exec(query, applicationID)
	if err != nil {
		return err
	}
	return nil
}

// AddKeyPairToApplication generate a ssh key pair and add them as application variables
// DEPCRECATED
func AddKeyPairToApplication(db gorp.SqlExecutor, store cache.Store, app *sdk.Application, keyname string, u *sdk.User) error {
	k, errGenerate := keys.GenerateSSHKey(keyname)
	if errGenerate != nil {
		return sdk.WrapError(errGenerate, "AddKeyPairToApplication> Cannot generate key")
	}

	v := sdk.Variable{
		Name:  keyname,
		Type:  sdk.KeyVariable,
		Value: k.Private,
	}

	if err := InsertVariable(db, store, app, v, u); err != nil {
		return err
	}

	p := sdk.Variable{
		Name:  keyname + ".pub",
		Type:  sdk.TextVariable,
		Value: k.Public,
	}

	return InsertVariable(db, store, app, p, u)
}

// insertAudit  insert an application variable audit
func inserAudit(db gorp.SqlExecutor, ava *sdk.ApplicationVariableAudit) error {
	dbAppVarAudit := dbApplicationVariableAudit(*ava)
	if err := db.Insert(&dbAppVarAudit); err != nil {
		return sdk.WrapError(err, "Cannot Insert Audit for variable %d", ava.VariableID)
	}
	*ava = sdk.ApplicationVariableAudit(dbAppVarAudit)
	return nil
}

// LoadVariableAudits Load audits for the given variable
func LoadVariableAudits(db gorp.SqlExecutor, appID, varID int64) ([]sdk.ApplicationVariableAudit, error) {
	var res []dbApplicationVariableAudit
	query := "SELECT * FROM application_variable_audit WHERE application_id = $1 AND variable_id = $2 ORDER BY versionned DESC"
	if _, err := db.Select(&res, query, appID, varID); err != nil {
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		if err != nil && err == sql.ErrNoRows {
			return []sdk.ApplicationVariableAudit{}, nil
		}
	}

	avas := make([]sdk.ApplicationVariableAudit, len(res))
	for i := range res {
		dbAva := &res[i]
		if err := dbAva.PostGet(db); err != nil {
			return nil, err
		}
		ava := sdk.ApplicationVariableAudit(*dbAva)
		avas[i] = ava
	}
	return avas, nil
}

// CountInVarValue counts how many time a pattern is in variable value for the given project
func CountInVarValue(db gorp.SqlExecutor, key string, value string) ([]string, error) {
	query := `
		SELECT DISTINCT application.name
		FROM application_variable
		JOIN application ON application.id = application_variable.application_id
		JOIN project ON project.id = application.project_id
		WHERE var_value like $2 AND project.projectkey = $1;
	`
	var appsName []string
	if _, err := db.Select(&appsName, query, key, fmt.Sprintf("%%%s%%", value)); err != nil {
		return nil, sdk.WrapError(err, "application.CountInVarValue> Unable to count usage")
	}
	return appsName, nil
}
