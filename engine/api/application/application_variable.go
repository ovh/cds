package application

import (
	"database/sql"
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

var (
	// ErrNoVariable when request requires specific variable in the applicatoin
	ErrNoVariable = fmt.Errorf("variable not in the application")
)

// GetVariableAudit Get variable audit for the given application
// Deprecated
func GetVariableAudit(db gorp.SqlExecutor, key, appName string) ([]sdk.VariableAudit, error) {
	// FIXME refactor using application_variable_audit.
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
		err = sdk.JSONUnmarshal([]byte(data), &vars)
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
		if err != sql.ErrNoRows {
			return nil, err
		}
		return []sdk.ApplicationVariableAudit{}, nil
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
