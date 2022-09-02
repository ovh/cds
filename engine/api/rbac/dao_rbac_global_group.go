package rbac

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/sdk"
)

func loadRbacGlobalGroupsByUserID(ctx context.Context, db gorp.SqlExecutor, userID string) ([]rbacGlobalGroup, error) {
	groups, err := group.LoadAllByUserID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	groupIDs := make([]int64, 0, len(groups))
	for _, g := range groups {
		groupIDs = append(groupIDs, g.ID)
	}
	return loadRbacGlobalGroupsByGroupIDs(ctx, db, groupIDs)
}

func loadRbacGlobalGroupsByGroupIDs(ctx context.Context, db gorp.SqlExecutor, groupIDs []int64) ([]rbacGlobalGroup, error) {
	q := gorpmapping.NewQuery("SELECT * FROM rbac_global_groups WHERE group_id = ANY ($1)").Args(pq.Int64Array(groupIDs))
	return getAllRBACGlobalGroups(ctx, db, q)
}

func getAllRBACGlobalGroups(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]rbacGlobalGroup, error) {
	var rbacGroups []rbacGlobalGroup
	if err := gorpmapping.GetAll(ctx, db, q, &rbacGroups); err != nil {
		return nil, err
	}
	for _, rg := range rbacGroups {
		isValid, err := gorpmapping.CheckSignature(rg, rg.Signature)
		if err != nil {
			return nil, sdk.WrapError(err, "error when checking signature for rbac_global_groups %d", rg.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac.getAllRBACGlobalGroups> rbac_global_groups %d data corrupted", rg.ID)
			continue
		}
	}
	return rbacGroups, nil
}

func loadRBACGlobalGroups(ctx context.Context, db gorp.SqlExecutor, rbacGlobal *rbacGlobal) error {
	q := gorpmapping.NewQuery("SELECT * FROM rbac_global_groups WHERE rbac_global_id = $1").Args(rbacGlobal.ID)
	rbacGlobalGroups, err := getAllRBACGlobalGroups(ctx, db, q)
	if err != nil {
		return err
	}
	for _, rgg := range rbacGlobalGroups {
		rbacGlobal.RBACGroupsIDs = append(rbacGlobal.RBACGroupsIDs, rgg.RbacGlobalGroupID)
	}
	return nil
}
