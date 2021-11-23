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

func getOrganizations(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]Organization, error) {
	os := []Organization{}

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

func getOrganization(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) (*Organization, error) {
	var org = Organization{}
	found, err := gorpmapping.Get(ctx, db, q, &org)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get authentified user organization")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	isValid, err := gorpmapping.CheckSignature(org, org.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "authentified user organization %d data corrupted", org.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	return &org, nil
}

func LoadOrganizationsByUserIDs(ctx context.Context, db gorp.SqlExecutor, userIDs []string) ([]Organization, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM authentified_user_organization
    WHERE authentified_user_id = ANY($1)
  `).Args(pq.StringArray(userIDs))
	return getOrganizations(ctx, db, query)
}

func LoadOrganizationByUserID(ctx context.Context, db gorp.SqlExecutor, userID string) (*Organization, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM authentified_user_organization
    WHERE authentified_user_id = $1
  `).Args(userID)
	return getOrganization(ctx, db, query)
}

func InsertOrganization(ctx context.Context, db gorpmapper.SqlExecutorWithTx, o *Organization) error {
	return sdk.WrapError(gorpmapping.InsertAndSign(ctx, db, o), "unable to insert authentified user organization")
}
