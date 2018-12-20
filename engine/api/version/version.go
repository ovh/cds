package version

import (
	"database/sql"
	"strings"

	"github.com/blang/semver"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// Upsert try to insert a new currentCDS version, if it already exist it does nothing
func Upsert(db gorp.SqlExecutor) error {
	var major, minor, patch uint64
	if sdk.VersionCurrent().Version != "" && !strings.HasPrefix(sdk.VersionCurrent().Version, "snapshot") {
		semverVal, err := semver.Parse(sdk.VersionCurrent().Version)
		if err != nil {
			return sdk.WrapError(err, "current version is not semver compatible")
		}
		major = semverVal.Major
		minor = semverVal.Minor
		patch = semverVal.Patch
	}

	query := "INSERT INTO cds_version(release, major, minor, patch) VALUES($1, $2, $3, $4) ON CONFLICT DO NOTHING"
	_, err := db.Exec(query, sdk.VersionCurrent().Version, major, minor, patch)
	return sdk.WithStack(err)
}

// IsFreshInstall return true if it's a fresh installation of CDS and not an upgrade
func IsFreshInstall(db gorp.SqlExecutor) (bool, error) {
	count, err := db.SelectInt("SELECT COUNT(id) FROM cds_version")
	if err != nil {
		return false, sdk.WithStack(err)
	}

	var noUsers bool
	var users []sdk.User
	if _, err := db.Select(&users, `SELECT id FROM "user" LIMIT 2`); err != nil && err == sql.ErrNoRows {
		noUsers = true
	}

	return noUsers && count == 0, nil
}

// MaxVersion return max CDS version already started major, minor, patch
func MaxVersion(db gorp.SqlExecutor) (major, minor, patch uint64, err error) {
	query := "SELECT MAX(major), MAX(minor), MAX(patch) FROM cds_version GROUP BY major,minor ORDER BY (major,minor) DESC LIMIT 1"
	if err = db.QueryRow(query).Scan(&major, &minor, &patch); err != nil {
		if err == sql.ErrNoRows {
			return 0, 0, 0, nil
		}
		return major, minor, patch, err
	}
	return major, minor, patch, nil
}
