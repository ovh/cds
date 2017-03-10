package environment

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
)

// DEPRECATED
// CreateAudit Create environment variable audit for the given project
func CreateAudit(db gorp.SqlExecutor, key string, env *sdk.Environment, u *sdk.User) error {

	vars, err := GetAllVariable(db, key, env.Name, WithEncryptPassword())
	if err != nil {
		return err
	}

	for i := range vars {
		v := &vars[i]
		if sdk.NeedPlaceholder(v.Type) {
			v.Value = base64.StdEncoding.EncodeToString([]byte(v.Value))
		}
	}

	data, err := json.Marshal(vars)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO environment_variable_audit_old (versionned, environment_id, data, author)
		VALUES (NOW(), $1, $2, $3)
	`
	_, err = db.Exec(query, env.ID, string(data), u.Username)
	return err
}

// GetAudit retrieve the current environment variable audit
func GetAudit(db gorp.SqlExecutor, auditID int64) ([]sdk.Variable, error) {
	query := `
		SELECT environment_variable_audit_old.data
		FROM environment_variable_audit_old
		WHERE environment_variable_audit_old.id = $1
	`
	var data string
	err := db.QueryRow(query, auditID).Scan(&data)
	if err != nil {
		return nil, err
	}
	var vars []sdk.Variable
	err = json.Unmarshal([]byte(data), &vars)
	for i := range vars {
		v := &vars[i]
		if sdk.NeedPlaceholder(v.Type) {
			decode, err := base64.StdEncoding.DecodeString(v.Value)
			if err != nil {
				return nil, err
			}
			v.Value = string(decode)
		}
	}
	return vars, err
}

// GetEnvironmentAudit Get environment audit for the given project
func GetEnvironmentAudit(db gorp.SqlExecutor, key, envName string) ([]sdk.VariableAudit, error) {
	audits := []sdk.VariableAudit{}
	query := `
		SELECT environment_variable_audit_old.id, environment_variable_audit_old.versionned, environment_variable_audit_old.data, environment_variable_audit_old.author
		FROM environment_variable_audit_old
		JOIN environment ON environment.id = environment_variable_audit_old.environment_id
		JOIN project ON project.id = environment.project_id
		WHERE project.projectkey = $1 AND environment.name = $2
		ORDER BY environment_variable_audit_old.versionned DESC
	`
	rows, err := db.Query(query, key, envName)
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
		for i := range vars {
			v := &vars[i]
			if sdk.NeedPlaceholder(v.Type) {
				v.Value = sdk.PasswordPlaceholder
			}
		}
		audit.Variables = vars
		audits = append(audits, audit)
	}
	return audits, nil
}

// GetAllVariableNameByProject Get all variable from all environment
func GetAllVariableNameByProject(db gorp.SqlExecutor, key string) ([]string, error) {
	nameArray := []string{}
	query := `
		SELECT distinct(environment_variable.name)
		FROM environment_variable
		JOIN environment on environment.id = environment_variable.environment_id
		JOIN project on project.id = environment.project_id
		WHERE project.projectkey=$1`
	rows, err := db.Query(query, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		if err != nil {
			return nil, err
		}
		nameArray = append(nameArray, name)
	}
	return nameArray, nil
}

type structarg struct {
	clearsecret   bool
	encryptsecret bool
}

// GetAllVariableFuncArg defines the base type for functional argument of GetAllVariable
type GetAllVariableFuncArg func(args *structarg)

// WithClearPassword is a function argument to GetAllVariable
func WithClearPassword() GetAllVariableFuncArg {
	return func(args *structarg) {
		args.clearsecret = true
	}
}

// WithEncryptPassword is a function argument to GetAllVariable
func WithEncryptPassword() GetAllVariableFuncArg {
	return func(args *structarg) {
		args.encryptsecret = true
	}
}

// GetVariableByID Get a variable for the given environment
func GetVariableByID(db gorp.SqlExecutor, envID int64, varID int64, args ...GetAllVariableFuncArg) (sdk.Variable, error) {
	v := sdk.Variable{}
	var clearVal sql.NullString
	var cipherVal []byte
	var typeVar string

	c := structarg{}
	for _, f := range args {
		f(&c)
	}

	query := `SELECT environment_variable.id, environment_variable.name, environment_variable.value,
						environment_variable.cipher_value, environment_variable.type
	          FROM environment_variable
	          WHERE environment_id = $1 AND id = $2
	          ORDER BY name`
	if err := db.QueryRow(query, envID, varID).Scan(&v.ID, &v.Name, &clearVal, &cipherVal, &typeVar); err != nil {
		return v, sdk.WrapError(err, "GetVariableByID> Cannot get variable %d", varID)
	}

	v.Type = sdk.VariableTypeFromString(typeVar)

	if c.encryptsecret && sdk.NeedPlaceholder(v.Type) {
		v.Value = string(cipherVal)
	} else {
		var errDecrypt error
		v.Value, errDecrypt = secret.DecryptS(v.Type, clearVal, cipherVal, c.clearsecret)
		if errDecrypt != nil {
			return v, sdk.WrapError(errDecrypt, "GetVariableByID> Cannot decrypt secret %s", v.Name)
		}
	}
	return v, nil
}

// GetVariable Get a variable for the given environment
func GetVariable(db gorp.SqlExecutor, key, envName string, varName string, args ...GetAllVariableFuncArg) (*sdk.Variable, error) {
	v := sdk.Variable{}
	var clearVal sql.NullString
	var cipherVal []byte
	var typeVar string

	c := structarg{}
	for _, f := range args {
		f(&c)
	}

	query := `SELECT environment_variable.id, environment_variable.name, environment_variable.value,
						environment_variable.cipher_value, environment_variable.type
	          FROM environment_variable
	          JOIN environment ON environment.id = environment_variable.environment_id
	          JOIN project ON project.id = environment.project_id
	          WHERE environment.name = $1 AND project.projectKey = $2 AND environment_variable.name = $3
	          ORDER BY name`
	if err := db.QueryRow(query, envName, key, varName).Scan(&v.ID, &v.Name, &clearVal, &cipherVal, &typeVar); err != nil {
		return nil, err
	}

	v.Type = sdk.VariableTypeFromString(typeVar)

	if c.encryptsecret && sdk.NeedPlaceholder(v.Type) {
		v.Value = string(cipherVal)
	} else {
		var errDecrypt error
		v.Value, errDecrypt = secret.DecryptS(v.Type, clearVal, cipherVal, c.clearsecret)
		if errDecrypt != nil {
			return nil, errDecrypt
		}
	}
	return &v, nil
}

// GetAllVariable Get all variable for the given environment
func GetAllVariable(db gorp.SqlExecutor, key, envName string, args ...GetAllVariableFuncArg) ([]sdk.Variable, error) {
	c := structarg{}
	for _, f := range args {
		f(&c)
	}

	variables := []sdk.Variable{}
	query := `SELECT environment_variable.id, environment_variable.name, environment_variable.value,
						environment_variable.cipher_value, environment_variable.type
	          FROM environment_variable
	          JOIN environment ON environment.id = environment_variable.environment_id
	          JOIN project ON project.id = environment.project_id
	          WHERE environment.name = $1 AND project.projectKey = $2
	          ORDER BY name`
	rows, err := db.Query(query, envName, key)
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
			if err != nil {
				return nil, err
			}
		}
		variables = append(variables, v)
	}
	return variables, err
}

// GetAllVariableByID Get all variable for the given environment
func GetAllVariableByID(db gorp.SqlExecutor, environmentID int64, args ...GetAllVariableFuncArg) ([]sdk.Variable, error) {
	c := structarg{}
	for _, f := range args {
		f(&c)
	}
	variables := []sdk.Variable{}
	query := `SELECT environment_variable.id, environment_variable.name, environment_variable.value,
						environment_variable.cipher_value, environment_variable.type
	          FROM environment_variable
	          WHERE environment_variable.environment_id = $1
	          ORDER BY name`
	rows, err := db.Query(query, environmentID)
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

// InsertVariable Insert a new variable in the given environment
func InsertVariable(db gorp.SqlExecutor, environmentID int64, variable *sdk.Variable, u *sdk.User) error {
	query := `INSERT INTO environment_variable(environment_id, name, value, cipher_value, type)
		  VALUES($1, $2, $3, $4, $5) RETURNING id`

	clear, cipher, err := secret.EncryptS(variable.Type, variable.Value)
	if err != nil {
		return sdk.WrapError(err, "InsertVariable> Cannot encrypt secret %s", variable.Name)
	}

	err = db.QueryRow(query, environmentID, variable.Name, clear, cipher, string(variable.Type)).Scan(&variable.ID)
	if err != nil {
		return sdk.WrapError(err, "InsertVariable> Cannot insert variable %s in db", variable.Name)
	}

	eva := &sdk.EnvironmentVariableAudit{
		Author:        u.Username,
		EnvironmentID: environmentID,
		Type:          sdk.AUDIT_ADD,
		VariableAfter: variable,
		VariableID:    variable.ID,
		Versionned:    time.Now(),
	}
	if err := InsertAudit(db, eva); err != nil {
		return sdk.WrapError(err, "InsertVariable> Cannot add audit")
	}

	query = `
		UPDATE environment 
		SET last_modified = current_timestamp
		WHERE id=$1
	`
	_, err = db.Exec(query, environmentID)
	return err
}

// UpdateVariable Update a variable in the given environment
func UpdateVariable(db gorp.SqlExecutor, envID int64, variable *sdk.Variable, u *sdk.User) error {
	varBefore, errV := GetVariableByID(db, envID, variable.ID, WithClearPassword())
	if errV != nil {
		return sdk.WrapError(errV, "UpdateVariable> Cannot load variable %d", variable.ID)
	}

	// If we are updating a batch of variables, some of them might be secrets, we don't want to crush the value
	if sdk.NeedPlaceholder(variable.Type) && variable.Value == sdk.PasswordPlaceholder {
		return nil
	}

	clear, cipher, err := secret.EncryptS(variable.Type, variable.Value)
	if err != nil {
		return sdk.WrapError(err, "UpdateVariable> Cannot encrypt secret")
	}

	query := `UPDATE environment_variable
	          SET value=$1, cipher_value=$2, type=$3, name=$6
	          WHERE environment_id = $4 AND environment_variable.id = $5`
	result, err := db.Exec(query, clear, cipher, string(variable.Type), envID, variable.ID, variable.Name)
	if err != nil {
		return sdk.WrapError(err, "Cannot update variable %s in db", variable.Name)
	}
	rowAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowAffected == 0 {
		return sdk.ErrNoVariable
	}

	eva := &sdk.EnvironmentVariableAudit{
		Author:         u.Username,
		EnvironmentID:  envID,
		Type:           sdk.AUDIT_UPDATE,
		VariableBefore: &varBefore,
		VariableAfter:  variable,
		VariableID:     variable.ID,
		Versionned:     time.Now(),
	}
	if err := InsertAudit(db, eva); err != nil {
		return sdk.WrapError(err, "UpdateVariable> Cannot add audit")
	}

	query = `
		UPDATE environment
		SET last_modified = current_timestamp
		WHERE id=$1`
	_, err = db.Exec(query, envID)
	return err
}

// DeleteVariable Delete a variable from the given pipeline
func DeleteVariable(db gorp.SqlExecutor, envID int64, variable *sdk.Variable, u *sdk.User) error {
	query := `DELETE FROM environment_variable
	          WHERE environment_variable.environment_id = $1 AND environment_variable.name = $2`
	result, err := db.Exec(query, envID, variable.Name)
	rowAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowAffected == 0 {
		return sdk.ErrNoVariable
	}

	eva := &sdk.EnvironmentVariableAudit{
		Author:         u.Username,
		EnvironmentID:  envID,
		Type:           sdk.AUDIT_DELETE,
		VariableBefore: variable,
		VariableID:     variable.ID,
		Versionned:     time.Now(),
	}
	if err := InsertAudit(db, eva); err != nil {
		return sdk.WrapError(err, "DeleteVariable> Cannot add audit")
	}

	query = `
		UPDATE environment
		SET last_modified = current_timestamp
		WHERE id = $1`
	_, err = db.Exec(query, envID)
	return err
}

// DeleteAllVariable Delete all variables from the given pipeline
func DeleteAllVariable(db gorp.SqlExecutor, environmentID int64) error {
	query := `DELETE FROM environment_variable
	          WHERE environment_variable.environment_id = $1`
	_, err := db.Exec(query, environmentID)
	if err != nil {
		return err
	}

	query = `
		UPDATE environment
		SET last_modified = current_timestamp
		WHERE id=$1`
	_, err = db.Exec(query, environmentID)
	return err
}

// InsertAudit Insert an audit for an environment variable
func InsertAudit(db gorp.SqlExecutor, eva *sdk.EnvironmentVariableAudit) error {
	dbEnvVarAudit := dbEnvironmentVariableAudit(*eva)
	if err := db.Insert(&dbEnvVarAudit); err != nil {
		return sdk.WrapError(err, "Cannot insert audit for variable %d", eva.VariableID)
	}
	*eva = sdk.EnvironmentVariableAudit(dbEnvVarAudit)
	return nil

}
