package application

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"

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

// WithEncryptPassword is a function argument to GetAllVariable to get secret encrypted
func WithEncryptPassword() FuncArg {
	return func(args *structarg) {
		args.encryptsecret = true
	}
}

// CreateAudit Create variable audit for the given application
func CreateAudit(db gorp.SqlExecutor, key string, app *sdk.Application, u *sdk.User) error {
	variables, err := GetAllVariable(db, key, app.Name, WithEncryptPassword())
	if err != nil {
		return err
	}
	for i := range variables {
		v := &variables[i]
		if sdk.NeedPlaceholder(v.Type) {
			v.Value = base64.StdEncoding.EncodeToString([]byte(v.Value))
		}
	}

	data, err := json.Marshal(variables)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO application_variable_audit (versionned, application_id, data, author)
		VALUES (NOW(), $1, $2, $3)
	`
	_, err = db.Exec(query, app.ID, string(data), u.Username)
	return err
}

// GetAudit retrieve the current application variable audit
func GetAudit(db gorp.SqlExecutor, key, appName string, auditID int64) ([]sdk.Variable, error) {
	query := `
		SELECT application_variable_audit.data
		FROM application_variable_audit
		JOIN application ON application.id = application_variable_audit.application_id
		JOIN project ON project.id = application.project_id
		WHERE application.name = $1 AND project.projectkey = $2 AND application_variable_audit.id = $3
		ORDER BY application_variable_audit.versionned DESC
	`
	var data string
	err := db.QueryRow(query, appName, key, auditID).Scan(&data)
	if err != nil {
		return nil, err
	}
	var variables []sdk.Variable
	err = json.Unmarshal([]byte(data), &variables)
	for i := range variables {
		v := &variables[i]
		if sdk.NeedPlaceholder(v.Type) {
			decode, err := base64.StdEncoding.DecodeString(v.Value)
			if err != nil {
				return nil, err
			}
			v.Value = string(decode)
		}
	}

	return variables, err
}

// GetVariableAudit Get variable audit for the given application
func GetVariableAudit(db gorp.SqlExecutor, key, appName string) ([]sdk.VariableAudit, error) {
	audits := []sdk.VariableAudit{}
	query := `
		SELECT application_variable_audit.id, application_variable_audit.versionned, application_variable_audit.data, application_variable_audit.author
		FROM application_variable_audit
		JOIN application ON application.id = application_variable_audit.application_id
		JOIN project ON project.id = application.project_id
		WHERE application.name = $1 AND project.projectkey = $2
		ORDER BY application_variable_audit.versionned DESC
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
		v.Type = sdk.VariableTypeFromString(typeVar)

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

// LoadVariable retrieve a specific variable
func LoadVariable(db gorp.SqlExecutor, appID int64, varName string) (sdk.Variable, error) {
	query := `SELECT id, var_name, var_value, var_type FROM application_variable
			WHERE application_id = $1 AND var_name = $2`

	var v sdk.Variable
	err := db.QueryRow(query, appID, varName).Scan(&v.ID, &v.Name, &v.Value, &v.Type)
	if err != nil {
		return v, err
	}
	if sdk.NeedPlaceholder(v.Type) {
		v.Value = sdk.PasswordPlaceholder
	}

	return v, nil
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
		v.Type = sdk.VariableTypeFromString(typeVar)
		v.Value, err = secret.DecryptS(v.Type, clearVal, cipherVal, c.clearsecret)
		if err != nil {
			return nil, err
		}
		variables = append(variables, v)
	}
	return variables, err
}

// InsertVariable Insert a new variable in the given application
func InsertVariable(db gorp.SqlExecutor, app *sdk.Application, variable sdk.Variable, u *sdk.User) error {

	if sdk.NeedPlaceholder(variable.Type) && variable.Value == sdk.PasswordPlaceholder {
		return fmt.Errorf("You try to insert a placeholder for new variable %s", variable.Name)
	}

	clear, cipher, err := secret.EncryptS(variable.Type, variable.Value)
	if err != nil {
		return err
	}

	query := `INSERT INTO application_variable(application_id, var_name, var_value, cipher_value, var_type)
		  VALUES($1, $2, $3, $4, $5)`
	_, err = db.Exec(query, app.ID, variable.Name, clear, cipher, string(variable.Type))
	if err != nil && strings.Contains(err.Error(), "application_variable_pkey") {
		return sdk.ErrVariableExists
	}
	if err != nil {
		return err
	}
	return UpdateLastModified(db, app, u)
}

// UpdateVariable Update a variable in the given application
func UpdateVariable(db gorp.SqlExecutor, app *sdk.Application, variable sdk.Variable, u *sdk.User) error {
	// If we are updating a batch of variables, some of them might be secrets, we don't want to crush the value
	if sdk.NeedPlaceholder(variable.Type) && variable.Value == sdk.PasswordPlaceholder {
		return nil
	}
	clear, cipher, err := secret.EncryptS(variable.Type, variable.Value)
	if err != nil {
		return err
	}

	query := `UPDATE application_variable SET var_name= $1, var_value=$2, cipher_value=$3 WHERE id = $4`
	result, err := db.Exec(query, variable.Name, clear, cipher, variable.ID)
	if err != nil {
		return err
	}
	rowAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowAffected == 0 {
		return ErrNoVariable
	}

	// Update application
	return UpdateLastModified(db, app, u)
}

// DeleteVariable Delete a variable from the given pipeline
func DeleteVariable(db gorp.SqlExecutor, app *sdk.Application, variableName string, u *sdk.User) error {
	query := `DELETE FROM application_variable
		  WHERE application_variable.application_id = $1 AND application_variable.var_name = $2`
	result, err := db.Exec(query, app.ID, variableName)
	if err != nil {
		return err
	}

	rowAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowAffected == 0 {
		return ErrNoVariable
	}
	return UpdateLastModified(db, app, u)
}

// DeleteAllVariable Delete all variables from the given pipeline
func DeleteAllVariable(db gorp.SqlExecutor, applicationID int64) error {
	query := `DELETE FROM application_variable
	          WHERE application_variable.application_id = $1`
	_, err := db.Exec(query, applicationID)
	if err != nil {
		return err
	}

	query = "UPDATE application SET last_modified = current_timestamp WHERE id=$1"
	_, err = db.Exec(query, applicationID)
	return err
}

// AddKeyPairToApplication generate a ssh key pair and add them as application variables
func AddKeyPairToApplication(db gorp.SqlExecutor, app *sdk.Application, keyname string, u *sdk.User) error {
	pub, priv, errGenerate := keys.Generatekeypair(keyname)
	if errGenerate != nil {
		return errGenerate
	}

	v := sdk.Variable{
		Name:  keyname,
		Type:  sdk.KeyVariable,
		Value: priv,
	}

	if err := InsertVariable(db, app, v, u); err != nil {
		return err
	}

	p := sdk.Variable{
		Name:  keyname + ".pub",
		Type:  sdk.TextVariable,
		Value: pub,
	}

	return InsertVariable(db, app, p, u)
}
