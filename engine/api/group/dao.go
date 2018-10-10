package group

import (
	"fmt"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
)

// GetAll returns all groups for given criteria.
func GetAll(db *gorp.DbMap, c Criteria) ([]sdk.Group, error) {
	gs := []sdk.Group{}

	if _, err := db.Select(&gs, fmt.Sprintf(`SELECT * FROM "group" WHERE %s`, c.where()), c.args()); err != nil {
		return nil, sdk.WrapError(err, "Cannot get groups")
	}

	return gs, nil
}
