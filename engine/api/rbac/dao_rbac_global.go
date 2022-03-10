package rbac

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func insertRbacGlobal(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rg *sdk.RbacGlobal) error {
	dbRG := rbacGlobal{RbacGlobal: *rg}
	if err := gorpmapping.InsertAndSign(ctx, db, &dbRG); err != nil {
		return err
	}

	for _, rbUser := range rg.RbacUsers {
		if err := insertRbacGlobalUser(ctx, db, dbRG.ID, rbUser.UserID); err != nil {
			return err
		}
	}
	for _, rbGroup := range rg.RbacGroups {
		if err := insertRbacGlobalGroup(ctx, db, dbRG.ID, rbGroup.GroupID); err != nil {
			return err
		}
	}
	*rg = dbRG.RbacGlobal
	return nil
}

func insertRbacGlobalUser(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacGlobalID int64, userID string) error {
	rgu := rbacGlobalUser{
		RbacGlobalID: rbacGlobalID,
		RbacUser: sdk.RbacUser{
			UserID: userID,
		},
	}
	if err := gorpmapping.InsertAndSign(ctx, db, &rgu); err != nil {
		return err
	}
	return nil
}

func insertRbacGlobalGroup(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacGlobalID int64, groupID int64) error {
	rgu := rbacGlobalGroup{
		RbacGlobalID: rbacGlobalID,
		RbacGroup: sdk.RbacGroup{
			GroupID: groupID,
		},
	}
	if err := gorpmapping.InsertAndSign(ctx, db, &rgu); err != nil {
		return err
	}
	return nil
}

func loadRbacRbacGlobalUsersTargeted(ctx context.Context, db gorp.SqlExecutor, rbacGlobal *rbacGlobal) error {
	query := `
		SELECT u.id, u.username
		FROM rbac_global_users rgu
		JOIN authentified_user u ON u.id = rgu.user_id
		WHERE rgu.rbac_global_id = $1
	`
	var users []rbacGlobalUser
	if err := gorpmapping.GetAll(ctx, db, gorpmapping.NewQuery(query).Args(rbacGlobal.ID), &users); err != nil {
		return err
	}
	rbacGlobal.RbacUsers = make([]sdk.RbacUser, 0, len(users))
	for _, u := range users {
		rbacGlobal.RbacUsers = append(rbacGlobal.RbacUsers, u.RbacUser)
	}
	return nil
}

func loadRbacRbacGlobalGroupsTargeted(ctx context.Context, db gorp.SqlExecutor, rbacGlobal *rbacGlobal) error {
	query := `
		SELECT g.id, g.name
		FROM rbac_global_groups rgg
		JOIN "group" g ON g.id = rgg.group_id
		WHERE rgg.rbac_global_id = $1
	`
	var groups []rbacGlobalGroup
	if err := gorpmapping.GetAll(ctx, db, gorpmapping.NewQuery(query).Args(rbacGlobal.ID), &groups); err != nil {
		return err
	}
	rbacGlobal.RbacGroups = make([]sdk.RbacGroup, 0, len(groups))
	for _, g := range groups {
		rbacGlobal.RbacGroups = append(rbacGlobal.RbacGroups, g.RbacGroup)
	}
	return nil
}
