package organization

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func InsertOrganizationRegion(ctx context.Context, db gorpmapper.SqlExecutorWithTx, orgaReg *sdk.OrganizationRegion) error {
	orgaReg.ID = sdk.UUID()
	dbData := &dbOrganizationRegion{OrganizationRegion: *orgaReg}
	if err := gorpmapping.InsertAndSign(ctx, db, dbData); err != nil {
		return err
	}
	*orgaReg = dbData.OrganizationRegion
	return nil
}

func DeleteOrganizationRegion(db gorpmapper.SqlExecutorWithTx, organizationID string, regionID string) error {
	_, err := db.Exec("DELETE FROM organization_region WHERE organization_id = $1 AND region_id = $2", organizationID, regionID)
	return sdk.WrapError(err, "cannot remove region %s from organization %s", regionID, organizationID)
}

func getAllOrganizationRegions(ctx context.Context, db gorp.SqlExecutor, query gorpmapping.Query) ([]sdk.OrganizationRegion, error) {
	var res []dbOrganizationRegion
	if err := gorpmapping.GetAll(ctx, db, query, &res); err != nil {
		return nil, err
	}

	orgs := make([]sdk.OrganizationRegion, 0, len(res))
	for _, r := range res {
		isValid, err := gorpmapping.CheckSignature(r, r.Signature)
		if err != nil {
			return nil, err
		}
		if !isValid {
			log.Error(ctx, "organization_region %s / %s data corrupted", r.ID)
			continue
		}
		orgs = append(orgs, r.OrganizationRegion)
	}
	return orgs, nil
}

func LoadRegionIDs(ctx context.Context, db gorp.SqlExecutor, organizationID string) ([]string, error) {
	query := gorpmapping.NewQuery("SELECT * FROM organization_region WHERE organization_id = $1").Args(organizationID)
	orgaRegs, err := getAllOrganizationRegions(ctx, db, query)
	if err != nil {
		return nil, err
	}
	IDs := make([]string, 0, len(orgaRegs))
	for _, o := range orgaRegs {
		IDs = append(IDs, o.RegionID)
	}
	return IDs, nil
}
