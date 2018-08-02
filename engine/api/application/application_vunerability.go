package application

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// InsertVulnerabilities Insert vulnerabilities
func InsertVulnerabilities(db gorp.SqlExecutor, vs []sdk.Vulnerability, appID int64) error {
	if _, err := db.Exec("DELETE FROM application_vulnerability WHERE application_id = $1", appID); err != nil {
		return sdk.WrapError(err, "InsertVulnerability> Unable to remove old vulnerabilities")
	}
	for _, v := range vs {
		v.ApplicationID = appID
		dbVuln := dbApplicationVulnerability(v)
		if err := db.Insert(&dbVuln); err != nil {
			return sdk.WrapError(err, "InsertVulnerability> Unable to insert vulnerabilities")
		}
	}
	return nil
}

// LoadLatestVulnerabilities load vulnerabilities for the given application
func LoadVulnerabilities(db gorp.SqlExecutor, appID int64) ([]sdk.Vulnerability, error) {
	results := make([]dbApplicationVulnerability, 0)
	query := `SELECT *
            FROM application_vulnerability
            WHERE application_id = $1`
	if _, err := db.Select(&results, query, appID); err != nil {
		if err != sql.ErrNoRows {
			return nil, sdk.WrapError(err, "LoadVulnerabilities> unable to load latest vulnerabilities for application %d", appID)
		}
		return nil, sdk.ErrNotFound
	}
	vulnerabilities := make([]sdk.Vulnerability, len(results))
	for i := range results {
		vulnerabilities[i] = sdk.Vulnerability(results[i])
	}
	return vulnerabilities, nil
}
