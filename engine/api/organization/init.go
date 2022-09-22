package organization

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

func CreateDefaultOrganization(ctx context.Context, db *gorp.DbMap, organizations sdk.StringSlice) error {
	orgasDb, err := LoadAllOrganizations(ctx, db)
	if err != nil {
		return err
	}
	orgaNames := make(sdk.StringSlice, 0, len(orgasDb))
	for _, o := range orgasDb {
		orgaNames = append(orgaNames, o.Name)
	}

	tx, err := db.Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback()
	for _, orga := range organizations {
		if orgaNames.Contains(orga) {
			continue
		}
		newOrga := sdk.Organization{Name: orga}
		if err := Insert(ctx, tx, &newOrga); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
