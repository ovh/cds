package application

import (
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
