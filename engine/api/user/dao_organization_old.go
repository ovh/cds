package user

import (
	"context"
	"github.com/ovh/cds/engine/gorpmapper"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func getOldOrganizations(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]OrganizationOld, error) {
	os := []OrganizationOld{}

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

func LoadOldOrganizationsByUserIDs(ctx context.Context, db gorp.SqlExecutor, userIDs []string) ([]OrganizationOld, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM authentified_user_organization_old
    WHERE authentified_user_id = ANY($1)
  `).Args(pq.StringArray(userIDs))
	return getOldOrganizations(ctx, db, query)
}

func InsertOldUserOrganisation(ctx context.Context, db gorpmapper.SqlExecutorWithTx, userID, orgaName string) error {
	uo := OrganizationOld{
		AuthentifiedUserID: userID,
		Organization:       orgaName,
	}
	return sdk.WithStack(gorpmapping.InsertAndSign(ctx, db, &uo))
}
