package project

import (
	"database/sql"
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// GetVariableAudit Get variable audit for the given project. DEPRECATED
func GetVariableAudit(db gorp.SqlExecutor, key string) ([]sdk.VariableAudit, error) {
	audits := []sdk.VariableAudit{}
	query := `
		SELECT project_variable_audit_old.id, project_variable_audit_old.versionned, project_variable_audit_old.data, project_variable_audit_old.author
		FROM project_variable_audit_old
		JOIN project ON project.id = project_variable_audit_old.project_id
		WHERE project.projectkey = $1
		ORDER BY project_variable_audit_old.versionned DESC
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

// insertAudit insert an audit on a project variable
func insertAudit(db gorp.SqlExecutor, pva *sdk.ProjectVariableAudit) error {
	dbProjVarAudit := dbProjectVariableAudit(*pva)
	if err := db.Insert(&dbProjVarAudit); err != nil {
		return sdk.WrapError(err, "Cannot insert audit for variable %d", pva.VariableID)
	}
	*pva = sdk.ProjectVariableAudit(dbProjVarAudit)
	return nil
}

// LoadVariableAudits Load audits for the given variable
func LoadVariableAudits(db gorp.SqlExecutor, projectID, varID int64) ([]sdk.ProjectVariableAudit, error) {
	var res []dbProjectVariableAudit
	query := "SELECT * FROM project_variable_audit WHERE project_id = $1 AND variable_id = $2 ORDER BY versionned DESC"
	if _, err := db.Select(&res, query, projectID, varID); err != nil {
		if err != sql.ErrNoRows {
			return nil, err
		}
		return []sdk.ProjectVariableAudit{}, nil
	}

	pvas := make([]sdk.ProjectVariableAudit, len(res))
	for i := range res {
		dbPva := &res[i]
		if err := dbPva.PostGet(db); err != nil {
			return nil, err
		}
		pva := sdk.ProjectVariableAudit(*dbPva)
		pvas[i] = pva
	}
	return pvas, nil
}
