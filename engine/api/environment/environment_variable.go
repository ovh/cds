package environment

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// insertAudit Insert an audit for an environment variable
func insertAudit(db gorp.SqlExecutor, eva *sdk.EnvironmentVariableAudit) error {
	dbEnvVarAudit := dbEnvironmentVariableAudit(*eva)
	if err := db.Insert(&dbEnvVarAudit); err != nil {
		return sdk.WrapError(err, "Cannot insert audit for variable %d", eva.VariableID)
	}
	*eva = sdk.EnvironmentVariableAudit(dbEnvVarAudit)
	return nil
}

// LoadVariableAudits Load audits for the given variable
func LoadVariableAudits(db gorp.SqlExecutor, envID, varID int64) ([]sdk.EnvironmentVariableAudit, error) {
	var res []dbEnvironmentVariableAudit
	query := "SELECT * FROM environment_variable_audit WHERE environment_id = $1 AND variable_id = $2 ORDER BY versionned DESC"
	if _, err := db.Select(&res, query, envID, varID); err != nil {
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		if err != nil && err == sql.ErrNoRows {
			return []sdk.EnvironmentVariableAudit{}, nil
		}
	}

	evas := make([]sdk.EnvironmentVariableAudit, len(res))
	for i := range res {
		dbEva := &res[i]
		if err := dbEva.PostGet(db); err != nil {
			return nil, err
		}
		pva := sdk.EnvironmentVariableAudit(*dbEva)
		evas[i] = pva
	}
	return evas, nil
}
