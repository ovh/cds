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

// LoadVulnerabilities load vulnerabilities for the given application
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

// LoadVulnerability load the given vulnerability
func LoadVulnerability(db gorp.SqlExecutor, appID int64, vulnID int64) (sdk.Vulnerability, error) {
	var dbVuln dbApplicationVulnerability
	query := `SELECT *
            FROM application_vulnerability
            WHERE application_id = $1 AND id = $2`
	if err := db.SelectOne(&dbVuln, query, appID, vulnID); err != nil {
		if err != sql.ErrNoRows {
			return sdk.Vulnerability{}, sdk.WrapError(err, "LoadVulnerability> unable to load vulnerability %d for application %d", vulnID, appID)
		}
		return sdk.Vulnerability{}, sdk.ErrNotFound
	}
	return sdk.Vulnerability(dbVuln), nil
}

// UpdateVulnerability updates a vulnerability
func UpdateVulnerability(db gorp.SqlExecutor, v sdk.Vulnerability) error {
	dbVuln := dbApplicationVulnerability(v)
	if _, err := db.Update(&dbVuln); err != nil {
		return sdk.WrapError(err, "UpdateVulnerability> Unable to update vulnerability")
	}
	return nil
}
