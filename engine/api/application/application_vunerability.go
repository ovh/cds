package application

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// InsertVulnerability Insert a new vulnerability
func InsertVulnerability(db gorp.SqlExecutor, v sdk.Vulnerability) error {
	dbVuln := dbApplicationVulnerability(v)
	if err := db.Insert(&dbVuln); err != nil {
		return sdk.WrapError(err, "InsertVulnerability> Unable to insert vulnerabilities")
	}
	return nil
}

// LoadVulnerabilitiesByRun loads vulnerabilities for the given run
func LoadVulnerabilitiesByRun(db gorp.SqlExecutor, nodeRunID int64) ([]sdk.Vulnerability, error) {
	results := make([]dbApplicationVulnerability, 0)
	query := `SELECT * FROM application_vulnerability 
            WHERE workflow_node_run_id=$1`
	if _, err := db.Select(&results, query, nodeRunID); err != nil {
		if err != sql.ErrNoRows {
			return nil, sdk.WrapError(err, "LoadVulnerabilitiesByRun> unable to load vulnerabilities for run %d", nodeRunID)
		}
		return nil, sdk.ErrNotFound
	}
	vulnerabilities := make([]sdk.Vulnerability, len(results))
	for i := range results {
		vulnerabilities[i] = sdk.Vulnerability(results[i])
	}
	return vulnerabilities, nil
}
