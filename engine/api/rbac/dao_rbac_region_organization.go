package rbac

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func insertRBACRegionOrganization(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacRegionID int64, orgaID string) error {
	rpk := rbacRegionOrganization{
		RbacRegionID:       rbacRegionID,
		RbacOrganizationID: orgaID,
	}
	if err := gorpmapping.InsertAndSign(ctx, db, &rpk); err != nil {
		return err
	}
	return nil
}

func LoadRBACRegionOrganizations(ctx context.Context, db gorp.SqlExecutor, rbacRegion *sdk.RBACRegion) error {
	q := gorpmapping.NewQuery("SELECT * FROM rbac_region_organizations WHERE rbac_region_id = $1").Args(rbacRegion.ID)
	rbacOrganizationIDS, err := getAllRBACRegionOrganizations(ctx, db, q)
	if err != nil {
		return err
	}
	rbacRegion.RBACOrganizationIDs = make([]string, 0, len(rbacOrganizationIDS))
	for _, rbacOrg := range rbacOrganizationIDS {
		rbacRegion.RBACOrganizationIDs = append(rbacRegion.RBACOrganizationIDs, rbacOrg.RbacOrganizationID)
	}
	return nil
}

func loadRBACRegionOrganizationsByRbacRegionID(ctx context.Context, db gorp.SqlExecutor, rbacRegionID int64) ([]rbacRegionOrganization, error) {
	q := gorpmapping.NewQuery("SELECT * FROM rbac_region_organizations WHERE rbac_region_id = $1").Args(rbacRegionID)
	rbacOrganizations, err := getAllRBACRegionOrganizations(ctx, db, q)
	if err != nil {
		return nil, err
	}
	return rbacOrganizations, nil
}

func getAllRBACRegionOrganizations(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]rbacRegionOrganization, error) {
	var rbacRegionOrganizations []rbacRegionOrganization
	if err := gorpmapping.GetAll(ctx, db, q, &rbacRegionOrganizations); err != nil {
		return nil, err
	}

	organizationsFiltered := make([]rbacRegionOrganization, 0, len(rbacRegionOrganizations))
	for _, rbacOrg := range rbacRegionOrganizations {
		isValid, err := gorpmapping.CheckSignature(rbacOrg, rbacOrg.Signature)
		if err != nil {
			return nil, sdk.WrapError(err, "error when checking signature for rbac_region_organizations %d", rbacOrg.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac.getAllRBACRegionOrganizations> rbac_region_organizations %d data corrupted", rbacOrg.ID)
			continue
		}
		organizationsFiltered = append(organizationsFiltered, rbacOrg)
	}
	return organizationsFiltered, nil
}
