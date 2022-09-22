package organization

import (
	"context"
	"github.com/lib/pq"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func Insert(ctx context.Context, db gorpmapper.SqlExecutorWithTx, orga *sdk.Organization) error {
	orga.ID = sdk.UUID()
	dbData := &dbOrganization{Organization: *orga}
	if err := gorpmapping.InsertAndSign(ctx, db, dbData); err != nil {
		return err
	}
	*orga = dbData.Organization
	return nil
}

func Delete(db gorpmapper.SqlExecutorWithTx, organizationID string) error {
	_, err := db.Exec("DELETE FROM organization WHERE id = $1", organizationID)
	return sdk.WrapError(err, "cannot delete organization %s", organizationID)
}

func getOrganization(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) (*sdk.Organization, error) {
	var dbOrg dbOrganization
	found, err := gorpmapping.Get(ctx, db, query, &dbOrg)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	isValid, err := gorpmapping.CheckSignature(dbOrg, dbOrg.Signature)
	if err != nil {
		return nil, err
	}
	if !isValid {
		log.Error(ctx, "organization %s / %s data corrupted", dbOrg.ID, dbOrg.Name)
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}
	return &dbOrg.Organization, nil
}

func getAllOrganizations(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.Organization, error) {
	var res []dbOrganization
	if err := gorpmapping.GetAll(ctx, db, query, &res); err != nil {
		return nil, err
	}

	orgs := make([]sdk.Organization, 0, len(res))
	for _, r := range res {
		isValid, err := gorpmapping.CheckSignature(r, r.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "organization %d / %s data corrupted", r.ID, r.Name)
			continue
		}
		orgs = append(orgs, r.Organization)
	}
	return orgs, nil
}

func LoadAllOrganizations(ctx context.Context, db gorp.SqlExecutor) ([]sdk.Organization, error) {
	query := gorpmapping.NewQuery(`SELECT organization.* FROM organization`)
	return getAllOrganizations(ctx, db, query)
}

func LoadOrganizationByName(ctx context.Context, db gorp.SqlExecutor, name string) (*sdk.Organization, error) {
	query := gorpmapping.NewQuery(`SELECT organization.* FROM organization WHERE organization.name = $1`).Args(name)
	return getOrganization(ctx, db, query)
}

func LoadOrganizationByID(ctx context.Context, db gorp.SqlExecutor, ID string) (*sdk.Organization, error) {
	query := gorpmapping.NewQuery(`SELECT organization.* FROM organization WHERE organization.id = $1`).Args(ID)
	return getOrganization(ctx, db, query)
}

func LoadOrganizationByIDs(ctx context.Context, db gorp.SqlExecutor, IDs []string) ([]sdk.Organization, error) {
	query := gorpmapping.NewQuery(`SELECT organization.* FROM organization WHERE organization.id = ANY($1)`).Args(pq.StringArray(IDs))
	return getAllOrganizations(ctx, db, query)
}
