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

func loadRBACVariableSetGroups(ctx context.Context, db gorp.SqlExecutor, rbacVariableSet *rbacVariableSet) error {
	q := gorpmapping.NewQuery("SELECT * FROM rbac_variableset_groups WHERE rbac_variableset_id = $1").Args(rbacVariableSet.ID)
	rbacVariableSetGroups, err := getAllRBACVariableSetGroups(ctx, db, q)
	if err != nil {
		return err
	}
	rbacVariableSet.RBACVariableSet.RBACGroupsIDs = make([]int64, 0, len(rbacVariableSetGroups))
	for _, g := range rbacVariableSetGroups {
		rbacVariableSet.RBACVariableSet.RBACGroupsIDs = append(rbacVariableSet.RBACVariableSet.RBACGroupsIDs, g.RbacVariableSetGroupID)
	}
	return nil
}

func getAllRBACVariableSetGroups(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]rbacVariableSetGroup, error) {
	var rbacGroupIDs []rbacVariableSetGroup
	if err := gorpmapping.GetAll(ctx, db, q, &rbacGroupIDs); err != nil {
		return nil, err
	}

	groupsFiltered := make([]rbacVariableSetGroup, 0, len(rbacGroupIDs))
	for _, rbacGroups := range rbacGroupIDs {
		isValid, err := gorpmapping.CheckSignature(rbacGroups, rbacGroups.Signature)
		if err != nil {
			return nil, sdk.WrapError(err, "error when checking signature for rbac_variableset_groups %d", rbacGroups.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac.getAllRBACVariableSetGroups> rbac_variableset_groups %d data corrupted", rbacGroups.ID)
			continue
		}
		groupsFiltered = append(groupsFiltered, rbacGroups)
	}
	return groupsFiltered, nil
}

func loadRBACVariableSetGroupsByGroupIDs(ctx context.Context, db gorp.SqlExecutor, groupIDs []int64) ([]rbacVariableSetGroup, error) {
	q := gorpmapping.NewQuery("SELECT * FROM rbac_variableset_groups WHERE group_id = ANY ($1)").Args(pq.Int64Array(groupIDs))
	return getAllRBACVariableSetGroups(ctx, db, q)
}

func loadRBACVariableSetGroupsByUserID(ctx context.Context, db gorp.SqlExecutor, userID string) ([]rbacVariableSetGroup, error) {
	groups, err := group.LoadAllByUserID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	groupIDs := make([]int64, 0, len(groups))
	for _, g := range groups {
		groupIDs = append(groupIDs, g.ID)
	}
	return loadRBACVariableSetGroupsByGroupIDs(ctx, db, groupIDs)
}
