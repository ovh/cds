package sanity

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// LoadAllWarnings loads all warnings existing in CDS
func LoadAllWarnings(db gorp.SqlExecutor, al string) ([]sdk.Warning, error) {
	// TODO
	return nil, nil
}

// LoadUserWarnings loads all warnings related to Jobs user has access to
func LoadUserWarnings(db gorp.SqlExecutor, al string, userID int64) ([]sdk.Warning, error) {
	// TODO
	return nil, nil
}
