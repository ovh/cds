package version

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// Upsert try to insert a new currentCDS version, if it already exist it does nothing
func Upsert(db gorp.SqlExecutor) error {
	_, err := db.Exec("INSERT INTO cds_version(release) VALUES($1) ON CONFLICT DO NOTHING", sdk.VersionCurrent().Version)
	return sdk.WithStack(err)
}

// IsFreshInstall return true if it's a fresh installation of CDS and not an upgrade
func IsFreshInstall(db gorp.SqlExecutor) (bool, error) {
	count, err := db.SelectInt("SELECT COUNT(id) FROM cds_version")
	if err != nil {
		return false, sdk.WithStack(err)
	}
	return count == 0, nil
}
