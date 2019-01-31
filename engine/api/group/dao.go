package group

import (
	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// GetAllByIDs returns all groups by ids.
func GetAllByIDs(db gorp.SqlExecutor, ids []int64) ([]sdk.Group, error) {
	gs := []sdk.Group{}

	if _, err := db.Select(&gs,
		`SELECT * FROM "group" WHERE id = ANY(string_to_array($1, ',')::int[])`,
		gorpmapping.IDsToQueryString(ids),
	); err != nil {
		return nil, sdk.WrapError(err, "cannot get groups")
	}

	return gs, nil
}
