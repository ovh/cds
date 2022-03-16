package rbac

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func insertRbacGlobal(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rg *rbacGlobal) error {
	if err := gorpmapping.InsertAndSign(ctx, db, rg); err != nil {
		return err
	}

	for _, userID := range rg.RBACUsersIDs {
		if err := insertRbacGlobalUser(ctx, db, rg.ID, userID); err != nil {
			return err
		}
	}
	for _, groupID := range rg.RBACGroupsIDs {
		if err := insertRbacGlobalGroup(ctx, db, rg.ID, groupID); err != nil {
			return err
		}
	}
	return nil
}

func insertRbacGlobalUser(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacGlobalID int64, userID string) error {
	rgu := rbacGlobalUser{
		RbacGlobalID:     rbacGlobalID,
		RbacGlobalUserID: userID,
	}
	if err := gorpmapping.InsertAndSign(ctx, db, &rgu); err != nil {
		return err
	}
	return nil
}

func insertRbacGlobalGroup(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacGlobalID int64, groupID int64) error {
	rgu := rbacGlobalGroup{
		RbacGlobalID:      rbacGlobalID,
		RbacGlobalGroupID: groupID,
	}
	if err := gorpmapping.InsertAndSign(ctx, db, &rgu); err != nil {
		return err
	}
	return nil
}

func getAllRBACGlobalUsers(ctx context.Context, db gorp.SqlExecutor, rbacGlobal *rbacGlobal) error {
	q := gorpmapping.NewQuery("SELECT * FROM  rbac_global_users WHERE rbac_global_id = $1").Args(rbacGlobal.ID)
	var rbacUserIDS []rbacGlobalUser
	if err := gorpmapping.GetAll(ctx, db, q, &rbacUserIDS); err != nil {
		return err
	}
	rbacGlobal.RBACGlobal.RBACUsersIDs = make([]string, 0, len(rbacUserIDS))
	for _, rbacUsers := range rbacUserIDS {
		isValid, err := gorpmapping.CheckSignature(rbacUsers, rbacUsers.Signature)
		if err != nil {
			return sdk.WrapError(err, "error when checking signature for rbac_global_users %d", rbacUsers.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac.getAllRBACGlobalUsers> rbac_global_user %d data corrupted", rbacUsers.ID)
			continue
		}
		rbacGlobal.RBACGlobal.RBACUsersIDs = append(rbacGlobal.RBACGlobal.RBACUsersIDs, rbacUsers.RbacGlobalUserID)
	}
	return nil
}

func getAllRBACGlobalGroups(ctx context.Context, db gorp.SqlExecutor, rbacGlobal *rbacGlobal) error {
	q := gorpmapping.NewQuery("SELECT * FROM rbac_global_groups WHERE rbac_global_id = $1").Args(rbacGlobal.ID)
	var rbacGroupIDs []rbacGlobalGroup
	if err := gorpmapping.GetAll(ctx, db, q, &rbacGroupIDs); err != nil {
		return err
	}
	rbacGlobal.RBACGlobal.RBACGroupsIDs = make([]int64, 0, len(rbacGroupIDs))
	for _, rbacGroups := range rbacGroupIDs {
		isValid, err := gorpmapping.CheckSignature(rbacGroups, rbacGroups.Signature)
		if err != nil {
			return sdk.WrapError(err, "error when checking signature for rbac_global_groups %d", rbacGroups.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac.getAllRBACGlobalUsers> rbac_global_groups %d data corrupted", rbacGroups.ID)
			continue
		}
		rbacGlobal.RBACGlobal.RBACGroupsIDs = append(rbacGlobal.RBACGlobal.RBACGroupsIDs, rbacGroups.RbacGlobalGroupID)
	}
	return nil
}
