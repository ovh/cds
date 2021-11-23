package group

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func getOrganizations(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]Organization, error) {
	os := []Organization{}

	if err := gorpmapping.GetAll(ctx, db, q, &os); err != nil {
		return nil, sdk.WrapError(err, "cannot get group organizations")
	}

	for i := range os {
		isValid, err := gorpmapping.CheckSignature(os[i], os[i].Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "group organization %d data corrupted", os[i].ID)
			continue
		}
	}

	return os, nil
}

func getOrganization(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) (*Organization, error) {
	var org = Organization{}
	found, err := gorpmapping.Get(ctx, db, q, &org)
	if err != nil {
		return nil, sdk.WrapError(err, "cannot get group organization")
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	isValid, err := gorpmapping.CheckSignature(org, org.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "group organization %d data corrupted", org.ID)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	return &org, nil
}

func LoadOrganizationsByGroupIDs(ctx context.Context, db gorp.SqlExecutor, groupIDs []int64) ([]Organization, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM group_organization
    WHERE group_id = ANY(string_to_array($1, ',')::int[])
  `).Args(gorpmapping.IDsToQueryString(groupIDs))
	return getOrganizations(ctx, db, query)
}

func LoadOrganizationByGroupID(ctx context.Context, db gorp.SqlExecutor, groupID int64) (*Organization, error) {
	query := gorpmapping.NewQuery(`
    SELECT *
    FROM group_organization
    WHERE group_id = $1
  `).Args(groupID)
	return getOrganization(ctx, db, query)
}

func InsertOrganization(ctx context.Context, db gorpmapper.SqlExecutorWithTx, o *Organization) error {
	return sdk.WrapError(gorpmapping.InsertAndSign(ctx, db, o), "unable to insert group organization")
}

func UpdateOrganization(ctx context.Context, db gorpmapper.SqlExecutorWithTx, o *Organization) error {
	return sdk.WrapError(gorpmapping.UpdateAndSign(ctx, db, o), "unable to update group organization")
}
