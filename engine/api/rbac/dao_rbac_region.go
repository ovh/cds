package rbac

import (
	"context"
	"encoding/json"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
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

func getAllRBACRegions(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]sdk.RBACRegion, error) {
	var rbacRegions []rbacRegion
	if err := gorpmapping.GetAll(ctx, db, q, &rbacRegions); err != nil {
		return nil, err
	}

	regionsFiltered := make([]sdk.RBACRegion, 0, len(rbacRegions))
	for _, rbacRegionData := range rbacRegions {
		isValid, err := gorpmapping.CheckSignature(rbacRegionData, rbacRegionData.Signature)
		if err != nil {
			return nil, sdk.WrapError(err, "error when checking signature for rbac_region %d", rbacRegionData.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac.getAllRBACRegions> rbac_region %d data corrupted", rbacRegionData.ID)
			continue
		}
		regionsFiltered = append(regionsFiltered, rbacRegionData.RBACRegion)
	}
	return regionsFiltered, nil
}

func LoadRegionIDsByRoleAndVCSUSer(ctx context.Context, db gorp.SqlExecutor, role string, user sdk.RBACVCSUser) ([]sdk.RBACRegion, error) {
	btes, _ := json.Marshal([]sdk.RBACVCSUser{user})
	q := gorpmapping.NewQuery(`SELECT * from rbac_region WHERE role = $1 AND vcs_users::JSONB @> $2`).Args(role, string(btes))
	return getAllRBACRegions(ctx, db, q)
}

func LoadRegionIDsByRoleAndUserID(ctx context.Context, db gorp.SqlExecutor, role string, userID string) ([]sdk.RBACRegion, error) {
	ctx, next := telemetry.Span(ctx, "LoadRegionIDsByRoleAndUserID")
	defer next()
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

func loadRBACRegionsByRoleAndIDs(ctx context.Context, db gorp.SqlExecutor, role string, rbacRegionIDs []int64) ([]sdk.RBACRegion, error) {
	q := gorpmapping.NewQuery(`SELECT * from rbac_region WHERE role = $1 AND id = ANY($2)`).Args(role, pq.Int64Array(rbacRegionIDs))
	return getAllRBACRegions(ctx, db, q)
}

func loadRBACRegionOnAllUsers(ctx context.Context, db gorp.SqlExecutor, role string) ([]sdk.RBACRegion, error) {
	q := gorpmapping.NewQuery("SELECT * from rbac_region WHERE role = $1 AND all_users = true").Args(role)
	return getAllRBACRegions(ctx, db, q)
}

func LoadRBACByRegionID(ctx context.Context, db gorp.SqlExecutor, regionID string) ([]sdk.RBAC, error) {
	query := gorpmapping.NewQuery(`SELECT * FROM rbac_region WHERE region_id = $1`).Args(regionID)
	rbRegions, err := getAllRBACRegions(ctx, db, query)
	if err != nil {
		return nil, err
	}
	rbacIDs := make(sdk.StringSlice, 0)
	for _, rg := range rbRegions {
		rbacIDs = append(rbacIDs, rg.RbacID)
	}

	ids, err := loadRBacIDsByHatcheryRegionID(ctx, db, regionID)
	if err != nil {
		return nil, err
	}
	rbacIDs = append(rbacIDs, ids...)
	rbacIDs.Unique()
	return LoadRBACByIDs(ctx, db, rbacIDs, LoadOptions.All)

}

func HasRoleOnRegion(ctx context.Context, db gorp.SqlExecutor, role string, regionID string, userID string, userOrgID string) (bool, error) {
	// Get all region that can match user id
	rRegion, err := LoadRegionIDsByRoleAndUserID(ctx, db, role, userID)
	if err != nil {
		return false, err
	}

	// Check region and user organization
	for _, rr := range rRegion {
		if rr.RegionID == regionID {
			if err := LoadRBACRegionOrganizations(ctx, db, &rr); err != nil {
				return false, err
			}
			for _, rbacOrga := range rr.RBACOrganizationIDs {
				if rbacOrga == userOrgID {
					return true, nil
				}
			}
		}
	}
	return false, nil
}
