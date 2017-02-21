package project

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/secret"
	"github.com/ovh/cds/sdk"
)

// GetVariableAudit Get variable audit for the given project
func GetVariableAudit(db gorp.SqlExecutor, key string) ([]sdk.VariableAudit, error) {
	audits := []sdk.VariableAudit{}
	query := `
		SELECT project_variable_audit.id, project_variable_audit.versionned, project_variable_audit.data, project_variable_audit.author
		FROM project_variable_audit
		JOIN project ON project.id = project_variable_audit.project_id
		WHERE project.projectkey = $1
		ORDER BY project_variable_audit.versionned DESC
	`
	rows, err := db.Query(query, key)
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

// GetAudit retrieve the current project variable audit
func GetAudit(db gorp.SqlExecutor, key string, auditID int64) ([]sdk.Variable, error) {
	query := `
		SELECT project_variable_audit.data
		FROM project_variable_audit
		JOIN project ON project.id = project_variable_audit.project_id
		WHERE project.projectkey = $1 AND project_variable_audit.id = $2
		ORDER BY project_variable_audit.versionned DESC
	`
	var data string
	err := db.QueryRow(query, key, auditID).Scan(&data)
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

// CreateAudit Create variable audit for the given project
func CreateAudit(db gorp.SqlExecutor, proj *sdk.Project, u *sdk.User) error {
	variables, err := GetAllVariableInProject(db, proj.ID, WithEncryptPassword())
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
		INSERT INTO project_variable_audit (versionned, project_id, data, author)
		VALUES (NOW(), $1, $2, $3)
	`
	_, err = db.Exec(query, proj.ID, string(data), u.Username)
	return err
}

// CheckVariableInProject check if the variable is already in the project or not
func CheckVariableInProject(db gorp.SqlExecutor, projectID int64, varName string) (bool, error) {
	query := `SELECT COUNT(id) FROM project_variable WHERE project_id = $1 AND var_name = $2`

	var nb int64
	err := db.QueryRow(query, projectID, varName).Scan(&nb)
	if err != nil {
		return false, err
	}
	if nb != 0 {
		return true, nil
	}
	return false, nil
}

type structarg struct {
	clearsecret   bool
	encryptsecret bool
}

// GetAllVariableFuncArg defines the base type for functional argument of GetAllVariable
type GetAllVariableFuncArg func(args *structarg)

// WithClearPassword is a function argument to GetAllVariableInProject
func WithClearPassword() GetAllVariableFuncArg {
	return func(args *structarg) {
		args.clearsecret = true
	}
}

// WithEncryptPassword is a function argument to GetAllVariableInProject.
func WithEncryptPassword() GetAllVariableFuncArg {
	return func(args *structarg) {
		args.encryptsecret = true
	}
}

// GetAllVariableInProject Get all variable for the given project
func GetAllVariableInProject(db gorp.SqlExecutor, projectID int64, args ...GetAllVariableFuncArg) ([]sdk.Variable, error) {
	c := structarg{}
	for _, f := range args {
		f(&c)
	}

	variables := []sdk.Variable{}
	query := `SELECT id, var_name, var_value, cipher_value, var_type
	          FROM project_variable
	          WHERE project_id=$1
	          ORDER BY var_name`
	rows, err := db.Query(query, projectID)
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

// GetAllVariableNameInProjectByKey Get all variable for the given project
func GetAllVariableNameInProjectByKey(db gorp.SqlExecutor, projectKey string) ([]string, error) {
	variables := []string{}
	query := `SELECT project_variable.var_name
	          FROM project_variable
	          JOIN project ON project.id = project_variable.project_id
	          WHERE project.projectKey = $1
	          ORDER BY var_name`
	rows, err := db.Query(query, projectKey)
	if err != nil {
		return variables, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		if err != nil {
			return nil, err
		}
		variables = append(variables, name)
	}
	return variables, err
}

// GetVariableInProject get the variable information for the given project
func GetVariableInProject(db gorp.SqlExecutor, projectID int64, variableName string) (*sdk.Variable, error) {
	variable := &sdk.Variable{}
	query := `SELECT id, var_name, var_value, var_type FROM project_variable
		  WHERE var_name=$1 AND project_id=$2`
	var typeVar string
	var varValue []byte
	err := db.QueryRow(query, variableName, projectID).Scan(&variable.ID, &variable.Name, &varValue, &typeVar)
	if err != nil {
		return variable, err
	}
	variable.Type = sdk.VariableTypeFromString(typeVar)
	if sdk.NeedPlaceholder(variable.Type) {
		variable.Value = sdk.PasswordPlaceholder
	} else {
		variable.Value = string(varValue)
	}
	return variable, err
}

// InsertVariableInProject Insert a new variable in the given project
func InsertVariableInProject(db gorp.SqlExecutor, proj *sdk.Project, variable sdk.Variable) error {
	query := `INSERT INTO project_variable(project_id, var_name, var_value, cipher_value, var_type)
		  VALUES($1, $2, $3, $4, $5)`

	clear, cipher, err := secret.EncryptS(variable.Type, variable.Value)
	if err != nil {
		return err
	}

	_, err = db.Exec(query, proj.ID, variable.Name, clear, cipher, string(variable.Type))
	if err != nil {
		return err
	}

	lastModified, err := UpdateProjectDB(db, proj.Key, proj.Name)
	if err == nil {
		proj.LastModified = lastModified
	}
	return err
}

// UpdateVariableInProject Update a variable in the given project
func UpdateVariableInProject(db gorp.SqlExecutor, proj *sdk.Project, variable sdk.Variable) error {
	// If we are updating a batch of variables, some of them might be secrets, we don't want to crush the value
	if sdk.NeedPlaceholder(variable.Type) && variable.Value == sdk.PasswordPlaceholder {
		return nil
	}

	clear, cipher, err := secret.EncryptS(variable.Type, variable.Value)
	if err != nil {
		return err
	}

	query := `UPDATE project_variable SET var_name=$1, var_value=$2, cipher_value=$3, var_type=$4
		   WHERE id=$5`
	_, err = db.Exec(query, variable.Name, clear, cipher, string(variable.Type), variable.ID)
	if err != nil {
		return err
	}

	lastModifier, err := UpdateProjectDB(db, proj.Key, proj.Name)
	if err == nil {
		proj.LastModified = lastModifier
	}
	return err
}

// DeleteVariableFromProject Delete a variable from the given project
func DeleteVariableFromProject(db gorp.SqlExecutor, proj *sdk.Project, variableName string) error {
	query := `DELETE FROM project_variable WHERE project_id=$1 AND var_name=$2`
	_, err := db.Exec(query, proj.ID, variableName)
	if err != nil {
		return err
	}

	lastModified, err := UpdateProjectDB(db, proj.Key, proj.Name)
	if err == nil {
		proj.LastModified = lastModified
	}
	return err
}

// DeleteAllVariableFromProject Delete all variables from the given project
func DeleteAllVariableFromProject(db gorp.SqlExecutor, projectID int64) error {
	query := `DELETE FROM project_variable WHERE project_id=$1`
	_, err := db.Exec(query, projectID)
	if err != nil {
		return err
	}

	query = "UPDATE project SET last_modified = current_timestamp WHERE id=$1"
	_, err = db.Exec(query, projectID)

	return err
}
