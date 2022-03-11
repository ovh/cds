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

	for _, rbProjectID := range dbRP.RbacProjectsIDs {
		if err := insertRbacProjectID(ctx, db, dbRP.ID, rbProjectID); err != nil {
			return err
		}
	}
	for _, rbUserID := range dbRP.RbacUsersIDs {
		if err := insertRbacProjectUser(ctx, db, dbRP.ID, rbUserID); err != nil {
			return err
		}
	}
	for _, rbGroupID := range dbRP.RbacGroupsIDs {
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
	rbacProject.RbacProjectKeys = make([]string, 0, len(prjs))
	rbacProject.RbacProjectsIDs = make([]int64, 0, len(prjs))
	for _, pj := range prjs {
		rbacProject.RbacProjectKeys = append(rbacProject.RbacProjectKeys, pj.Key)
		rbacProject.RbacProjectsIDs = append(rbacProject.RbacProjectsIDs, pj.ID)
	}
	return nil
}

func loadRbacRbacProjectUsersTargeted(ctx context.Context, db gorp.SqlExecutor, rbacProject *rbacProject) error {
	users, err := user.LoadUsersByRbacProject(ctx, db, rbacProject.ID)
	if err != nil {
		return err
	}
	rbacProject.RbacUsersName = make([]string, 0, len(users))
	rbacProject.RbacUsersIDs = make([]string, 0, len(users))
	for _, u := range users {
		rbacProject.RbacUsersName = append(rbacProject.RbacUsersName, u.Username)
		rbacProject.RbacUsersIDs = append(rbacProject.RbacUsersIDs, u.ID)
	}
	return nil
}

func loadRbacRbacProjectGroupsTargeted(ctx context.Context, db gorp.SqlExecutor, rbacProject *rbacProject) error {
	groups, err := group.LoadGroupByRbacProject(ctx, db, rbacProject.ID)
	if err != nil {
		return err
	}
	rbacProject.RbacGroupsName = make([]string, 0, len(groups))
	rbacProject.RbacGroupsIDs = make([]int64, 0, len(groups))
	for _, g := range groups {
		rbacProject.RbacGroupsName = append(rbacProject.RbacGroupsName, g.Name)
		rbacProject.RbacGroupsIDs = append(rbacProject.RbacGroupsIDs, g.ID)
	}
	return nil
}
