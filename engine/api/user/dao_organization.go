package user

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func getUserOrganizations(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]UserOrganization, error) {
	os := []UserOrganization{}

	if err := gorpmapping.GetAll(ctx, db, q, &os); err != nil {
		return nil, sdk.WrapError(err, "cannot get user organizations")
	}

	for i := range os {
		isValid, err := gorpmapping.CheckSignature(os[i], os[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "authentified user organization %d data corrupted", os[i].ID)
			continue
		}
	}

	return os, nil
}

func LoadAllUserOrganizationsByUserIDs(ctx context.Context, db gorp.SqlExecutor, userIDs []string) ([]UserOrganization, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM authentified_user_organization
    WHERE authentified_user_id = ANY($1)
  `).Args(pq.StringArray(userIDs))
	return getUserOrganizations(ctx, db, query)
}

func InsertUserOrganization(ctx context.Context, db gorpmapper.SqlExecutorWithTx, o *UserOrganization) error {
	o.ID = sdk.UUID()
	return sdk.WrapError(gorpmapping.InsertAndSign(ctx, db, o), "unable to insert authentified user organization")
}
