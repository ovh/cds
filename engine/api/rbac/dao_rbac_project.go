package rbac

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/engine/gorpmapper"
)

func insertRbacProject(ctx context.Context, db gorpmapper.SqlExecutorWithTx, dbRP *rbacProject) error {
	if err := gorpmapping.InsertAndSign(ctx, db, dbRP); err != nil {
		return err
	}

	for _, rbProjectID := range dbRP.RBACProjectsIDs {
		if err := insertRbacProjectID(ctx, db, dbRP.ID, rbProjectID); err != nil {
			return err
		}
	}
	for _, rbUserID := range dbRP.RBACUsersIDs {
		if err := insertRbacProjectUser(ctx, db, dbRP.ID, rbUserID); err != nil {
			return err
		}
	}
	for _, rbGroupID := range dbRP.RBACGroupsIDs {
		if err := insertRbacProjectGroup(ctx, db, dbRP.ID, rbGroupID); err != nil {
			return err
		}
	}
	return nil
}

func insertRbacProjectID(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacParentID int64, projectID int64) error {
	rgu := rbacProjectID{
		RbacProjectID: rbacParentID,
		ProjectID:     projectID,
	}
	if err := gorpmapping.InsertAndSign(ctx, db, &rgu); err != nil {
		return err
	}
	return nil
}

func insertRbacProjectUser(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacProjectID int64, userID string) error {
	rgu := rbacProjectUser{
		RbacProjectID:     rbacProjectID,
		RbacProjectUserID: userID,
	}
	if err := gorpmapping.InsertAndSign(ctx, db, &rgu); err != nil {
		return err
	}
	return nil
}

func insertRbacProjectGroup(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacProjectID int64, groupID int64) error {
	rgu := rbacProjectGroup{
		RbacProjectID:      rbacProjectID,
		RbacProjectGroupID: groupID,
	}
	if err := gorpmapping.InsertAndSign(ctx, db, &rgu); err != nil {
		return err
	}
	return nil
}

func loadRbacProjectTargeted(ctx context.Context, db gorp.SqlExecutor, rbacProject *rbacProject) error {
	prjs, err := project.LoadProjectByRbacProject(ctx, db, rbacProject.ID)
	if err != nil {
		return err
	}
	rbacProject.RBACProjectKeys = make([]string, 0, len(prjs))
	rbacProject.RBACProjectsIDs = make([]int64, 0, len(prjs))
	for _, pj := range prjs {
		rbacProject.RBACProjectKeys = append(rbacProject.RBACProjectKeys, pj.Key)
		rbacProject.RBACProjectsIDs = append(rbacProject.RBACProjectsIDs, pj.ID)
	}
	return nil
}

func loadRbacRbacProjectUsersTargeted(ctx context.Context, db gorp.SqlExecutor, rbacProject *rbacProject) error {
	users, err := user.LoadUsersByRbacProject(ctx, db, rbacProject.ID)
	if err != nil {
		return err
	}
	rbacProject.RBACUsersName = make([]string, 0, len(users))
	rbacProject.RBACUsersIDs = make([]string, 0, len(users))
	for _, u := range users {
		rbacProject.RBACUsersName = append(rbacProject.RBACUsersName, u.Username)
		rbacProject.RBACUsersIDs = append(rbacProject.RBACUsersIDs, u.ID)
	}
	return nil
}

func loadRbacRbacProjectGroupsTargeted(ctx context.Context, db gorp.SqlExecutor, rbacProject *rbacProject) error {
	groups, err := group.LoadGroupByRbacProject(ctx, db, rbacProject.ID)
	if err != nil {
		return err
	}
	rbacProject.RBACGroupsName = make([]string, 0, len(groups))
	rbacProject.RBACGroupsIDs = make([]int64, 0, len(groups))
	for _, g := range groups {
		rbacProject.RBACGroupsName = append(rbacProject.RBACGroupsName, g.Name)
		rbacProject.RBACGroupsIDs = append(rbacProject.RBACGroupsIDs, g.ID)
	}
	return nil
}
