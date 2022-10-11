package rbac

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func insertRBACRegionGroup(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacRegionID int64, groupID int64) error {
	rgu := rbacRegionGroup{
		RbacRegionID: rbacRegionID,
		RbacGroupID:  groupID,
	}
	if err := gorpmapping.InsertAndSign(ctx, db, &rgu); err != nil {
		return err
	}
	return nil
}

func loadRBACRegionGroupsByUserID(ctx context.Context, db gorp.SqlExecutor, userID string) ([]rbacRegionGroup, error) {
	groups, err := group.LoadAllByUserID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	groupIDs := make([]int64, 0, len(groups))
	for _, g := range groups {
		groupIDs = append(groupIDs, g.ID)
	}
	return loadRBACRegionGroupsByGroupIDs(ctx, db, groupIDs)
}

func loadRBACRegionGroupsByGroupIDs(ctx context.Context, db gorp.SqlExecutor, groupIDs []int64) ([]rbacRegionGroup, error) {
	q := gorpmapping.NewQuery("SELECT * FROM rbac_region_groups WHERE group_id = ANY ($1)").Args(pq.Int64Array(groupIDs))
	return getAllRBACRegionGroups(ctx, db, q)
}

func loadRBACRegionGroups(ctx context.Context, db gorp.SqlExecutor, rbacRegion *rbacRegion) error {
	q := gorpmapping.NewQuery("SELECT * FROM rbac_region_groups WHERE rbac_region_id = $1").Args(rbacRegion.ID)
	rbacRegionGroups, err := getAllRBACRegionGroups(ctx, db, q)
	if err != nil {
		return err
	}
	rbacRegion.RBACRegion.RBACGroupsIDs = make([]int64, 0, len(rbacRegionGroups))
	for _, g := range rbacRegionGroups {
		rbacRegion.RBACRegion.RBACGroupsIDs = append(rbacRegion.RBACRegion.RBACGroupsIDs, g.RbacGroupID)
	}
	return nil
}

func getAllRBACRegionGroups(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]rbacRegionGroup, error) {
	var rbacGroupIDs []rbacRegionGroup
	if err := gorpmapping.GetAll(ctx, db, q, &rbacGroupIDs); err != nil {
		return nil, err
	}

	groupsFiltered := make([]rbacRegionGroup, 0, len(rbacGroupIDs))
	for _, rbacGroups := range rbacGroupIDs {
		isValid, err := gorpmapping.CheckSignature(rbacGroups, rbacGroups.Signature)
		if err != nil {
			return nil, sdk.WrapError(err, "error when checking signature for rbac_region_groups %d", rbacGroups.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac.getAllRBACRegionGroups> rbac_region_groups %d data corrupted", rbacGroups.ID)
			continue
		}
		groupsFiltered = append(groupsFiltered, rbacGroups)
	}
	return groupsFiltered, nil
}
