package rbac

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/organization"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/telemetry"
)

func insertRBACRegion(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacRegion *rbacRegion) error {
	if err := gorpmapping.InsertAndSign(ctx, db, rbacRegion); err != nil {
		return err
	}

	for _, orgaID := range rbacRegion.RBACOrganizationIDs {
		if err := insertRBACRegionOrganization(ctx, db, rbacRegion.ID, orgaID); err != nil {
			return err
		}
	}
	for _, rbUserID := range rbacRegion.RBACUsersIDs {
		if err := insertRBACRegionUser(ctx, db, rbacRegion.ID, rbUserID); err != nil {
			return err
		}
	}
	for _, rbGroupID := range rbacRegion.RBACGroupsIDs {
		if err := insertRBACRegionGroup(ctx, db, rbacRegion.ID, rbGroupID); err != nil {
			return err
		}
	}
	return nil
}

func getAllRBACRegions(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]rbacRegion, error) {
	var rbacRegions []rbacRegion
	if err := gorpmapping.GetAll(ctx, db, q, &rbacRegions); err != nil {
		return nil, err
	}

	regionsFiltered := make([]rbacRegion, 0, len(rbacRegions))
	for _, rbacRegionData := range rbacRegions {
		isValid, err := gorpmapping.CheckSignature(rbacRegionData, rbacRegionData.Signature)
		if err != nil {
			return nil, sdk.WrapError(err, "error when checking signature for rbac_region %d", rbacRegionData.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac.getAllRBACRegions> rbac_region %d data corrupted", rbacRegionData.ID)
			continue
		}
		regionsFiltered = append(regionsFiltered, rbacRegionData)
	}
	return regionsFiltered, nil
}

func HasRoleOnRegionAndUserID(ctx context.Context, db gorp.SqlExecutor, role string, authentifiedUser *sdk.AuthentifiedUser, regionIdentifier string) (bool, error) {
	ctx, next := telemetry.Span(ctx, "rbac.HasRoleOnRegionAndUserID")
	defer next()

	// Get all region permissions with the given user
	rRegion, err := LoadRegionIDsByRoleAndUserID(ctx, db, role, authentifiedUser.ID)
	if err != nil {
		return false, err
	}

	// Load user organization to get its ID
	org, err := organization.LoadOrganizationByName(ctx, db, authentifiedUser.Organization)
	if err != nil {
		return false, err
	}

	// Load region ID if needed
	regionID := regionIdentifier
	if !sdk.IsValidUUID(regionID) {
		reg, err := region.LoadRegionByName(ctx, db, regionIdentifier)
		if err != nil {
			return false, err
		}
		regionID = reg.ID
	}

	// Check region and organization
	for _, rr := range rRegion {
		if rr.RegionID == regionID {
			if err := loadRBACRegionOrganizations(ctx, db, &rr); err != nil {
				return false, err
			}
			for _, rbacOrga := range rr.RBACOrganizationIDs {
				if rbacOrga == org.ID {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func LoadRegionIDsByRoleAndUserID(ctx context.Context, db gorp.SqlExecutor, role string, userID string) ([]rbacRegion, error) {
	// Get rbac_region_groups

	rbacRegionGroups, err := loadRBACRegionGroupsByUserID(ctx, db, userID)
	if err != nil {
		return nil, err
	}

	// Get rbac_project_users
	rbacRegionUsers, err := loadRBACRegionUsersByUserID(ctx, db, userID)
	if err != nil {
		return nil, err
	}

	// Deduplicate rbac_region.id
	rbacRegionIDs := make(sdk.Int64Slice, 0)
	for _, rrg := range rbacRegionGroups {
		rbacRegionIDs = append(rbacRegionIDs, rrg.RbacRegionID)
	}
	for _, rru := range rbacRegionUsers {
		rbacRegionIDs = append(rbacRegionIDs, rru.RbacRegionID)
	}
	rbacRegionIDs.Unique()

	// Get rbac_region
	rbacRegions, err := loadRBACRegionsByRoleAndIDs(ctx, db, role, rbacRegionIDs)
	if err != nil {
		return nil, err
	}

	// Load also rbac_region with all users allowed
	rbacRegionsAllUsers, err := loadRBACRegionOnAllUsers(ctx, db, role)
	if err != nil {
		return nil, err
	}
	rbacRegions = append(rbacRegions, rbacRegionsAllUsers...)
	return rbacRegions, nil
}

func loadRBACRegionsByRoleAndIDs(ctx context.Context, db gorp.SqlExecutor, role string, rbacRegionIDs []int64) ([]rbacRegion, error) {
	q := gorpmapping.NewQuery(`SELECT * from rbac_region WHERE role = $1 AND id = ANY($2)`).Args(role, pq.Int64Array(rbacRegionIDs))
	return getAllRBACRegions(ctx, db, q)
}

func loadRBACRegionOnAllUsers(ctx context.Context, db gorp.SqlExecutor, role string) ([]rbacRegion, error) {
	q := gorpmapping.NewQuery("SELECT * from rbac_region WHERE role = $1 AND all_users = true").Args(role)
	return getAllRBACRegions(ctx, db, q)
}
