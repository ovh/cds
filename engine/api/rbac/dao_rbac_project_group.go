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

func loadRbacProjectGroupsByUserID(ctx context.Context, db gorp.SqlExecutor, userID string) ([]rbacProjectGroup, error) {
	groups, err := group.LoadAllByUserID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	groupIDs := make([]int64, 0, len(groups))
	for _, g := range groups {
		groupIDs = append(groupIDs, g.ID)
	}
	return loadRbacProjectGroupsByGroupIDs(ctx, db, groupIDs)
}

func loadRbacProjectGroupsByGroupIDs(ctx context.Context, db gorp.SqlExecutor, groupIDs []int64) ([]rbacProjectGroup, error) {
	q := gorpmapping.NewQuery("SELECT * FROM rbac_project_groups WHERE group_id = ANY ($1)").Args(pq.Int64Array(groupIDs))
	return getAllRBACProjectGroups(ctx, db, q)
}

func loadRBACProjectGroups(ctx context.Context, db gorp.SqlExecutor, rbacProject *rbacProject) error {
	q := gorpmapping.NewQuery("SELECT * FROM rbac_project_groups WHERE rbac_project_id = $1").Args(rbacProject.ID)
	rbacProjectGroups, err := getAllRBACProjectGroups(ctx, db, q)
	if err != nil {
		return err
	}
	rbacProject.RBACProject.RBACGroupsIDs = make([]int64, 0, len(rbacProjectGroups))
	for _, g := range rbacProjectGroups {
		rbacProject.RBACProject.RBACGroupsIDs = append(rbacProject.RBACProject.RBACGroupsIDs, g.RbacProjectGroupID)
	}
	return nil
}

func getAllRBACProjectGroups(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]rbacProjectGroup, error) {
	var rbacGroupIDs []rbacProjectGroup
	if err := gorpmapping.GetAll(ctx, db, q, &rbacGroupIDs); err != nil {
		return nil, err
	}
	for _, rbacGroups := range rbacGroupIDs {
		isValid, err := gorpmapping.CheckSignature(rbacGroups, rbacGroups.Signature)
		if err != nil {
			return nil, sdk.WrapError(err, "error when checking signature for rbac_project_groups %d", rbacGroups.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac.getAllRBACProjectGroups> rbac_project_groups %d data corrupted", rbacGroups.ID)
			continue
		}
	}
	return rbacGroupIDs, nil
}
