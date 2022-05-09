package application

import (
	"database/sql"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// LoadVulnerabilitiesSummary compute vulnerabilities summary
func LoadVulnerabilitiesSummary(db gorp.SqlExecutor, appID int64) (map[string]float64, error) {
	query := `
    SELECT json_object_agg(severity, nb)::TEXT
    FROM (
	    SELECT count(id) AS nb, severity
      FROM application_vulnerability
	    WHERE application_id = $1
	    GROUP BY severity
    ) tmp;
  `

	var summary map[string]float64
	var result sql.NullString
	if err := db.QueryRow(query, appID).Scan(&result); err != nil {
		return nil, sdk.WithStack(err)
	}

	if err := gorpmapping.JSONNullString(result, &summary); err != nil {
		return nil, sdk.WrapError(err, "unable to unmarshal summary")
	}
	return summary, nil
}

// InsertVulnerabilities Insert vulnerabilities
func InsertVulnerabilities(db gorp.SqlExecutor, vs []sdk.Vulnerability, appID int64) error {
	for _, v := range vs {
		v.ApplicationID = appID
		dbVuln := dbApplicationVulnerability(v)
		if err := db.Insert(&dbVuln); err != nil {
			return sdk.WrapError(err, "unable to insert vulnerabilities")
		}
	}
	return nil
}

// DeleteVulnerabilitiesByApplicationIDAndType removes all the vulnerabilities for given application and type.
func DeleteVulnerabilitiesByApplicationIDAndType(db gorp.SqlExecutor, applicationID int64, vulnerabilityType string) error {
	if _, err := db.Exec(`
    DELETE FROM application_vulnerability
    WHERE application_id = $1
    AND type = $2
  `, applicationID, vulnerabilityType); err != nil {
		return sdk.WrapError(err, "unable to remove vulnerabilities with type %s for application with id %d", vulnerabilityType, applicationID)
	}
	return nil
}

// LoadVulnerabilities load vulnerabilities for the given application
func LoadVulnerabilities(db gorp.SqlExecutor, appID int64) ([]sdk.Vulnerability, error) {
	results := make([]dbApplicationVulnerability, 0)
	query := `
    SELECT *
    FROM application_vulnerability
    WHERE application_id = $1
  `
	if _, err := db.Select(&results, query, appID); err != nil {
		if err != sql.ErrNoRows {
			return nil, sdk.WrapError(err, "unable to load latest vulnerabilities for application %d", appID)
		}
		return nil, nil
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
			return sdk.Vulnerability{}, sdk.WrapError(err, "unable to load vulnerability %d for application %d", vulnID, appID)
		}
		return sdk.Vulnerability{}, sdk.WithStack(sdk.ErrNotFound)
	}
	return sdk.Vulnerability(dbVuln), nil
}

// UpdateVulnerability updates a vulnerability
func UpdateVulnerability(db gorp.SqlExecutor, v sdk.Vulnerability) error {
	dbVuln := dbApplicationVulnerability(v)
	if _, err := db.Update(&dbVuln); err != nil {
		return sdk.WrapError(err, "Unable to update vulnerability")
	}
	return nil
}
