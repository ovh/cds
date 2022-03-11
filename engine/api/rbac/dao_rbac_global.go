package rbac

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/gorpmapper"
)

func insertRbacGlobal(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rg *rbacGlobal) error {
	if err := gorpmapping.InsertAndSign(ctx, db, rg); err != nil {
		return err
	}

	for _, userID := range rg.RbacUsersIDs {
		if err := insertRbacGlobalUser(ctx, db, rg.ID, userID); err != nil {
			return err
		}
	}
	for _, groupID := range rg.RbacGroupsIDs {
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

func loadRbacRbacGlobalUsersTargeted(ctx context.Context, db gorp.SqlExecutor, rbacGlobal *rbacGlobal) error {
	users, err := user.LoadUsersByRbacGlobal(ctx, db, rbacGlobal.ID)
	if err != nil {
		return err
	}
	rbacGlobal.RbacUsersName = make([]string, 0, len(users))
	rbacGlobal.RbacUsersIDs = make([]string, 0, len(users))
	for _, u := range users {
		rbacGlobal.RbacUsersName = append(rbacGlobal.RbacUsersName, u.Username)
		rbacGlobal.RbacUsersIDs = append(rbacGlobal.RbacUsersIDs, u.ID)
	}
	return nil
}

func loadRbacRbacGlobalGroupsTargeted(ctx context.Context, db gorp.SqlExecutor, rbacGlobal *rbacGlobal) error {
	groups, err := group.LoadGroupByRbacGlobal(ctx, db, rbacGlobal.ID)
	if err != nil {
		return err
	}
	rbacGlobal.RbacGroupsName = make([]string, 0, len(groups))
	rbacGlobal.RbacGroupsIDs = make([]int64, 0, len(groups))
	for _, g := range groups {
		rbacGlobal.RbacGroupsName = append(rbacGlobal.RbacGroupsName, g.Name)
		rbacGlobal.RbacGroupsIDs = append(rbacGlobal.RbacGroupsIDs, g.ID)
	}
	return nil
}
