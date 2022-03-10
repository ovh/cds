package rbac

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func insertRbacProject(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rp *sdk.RbacProject) error {
	dbRP := rbacProject{RbacProject: *rp}
	if err := gorpmapping.InsertAndSign(ctx, db, &dbRP); err != nil {
		return err
	}

	for _, rbProject := range rp.Projects {
		if err := insertRbacProjectID(ctx, db, dbRP.ID, rbProject.ProjectID); err != nil {
			return err
		}
	}
	for _, rbUser := range rp.RbacUsers {
		if err := insertRbacProjectUser(ctx, db, dbRP.ID, rbUser.UserID); err != nil {
			return err
		}
	}
	for _, rbGroup := range rp.RbacGroups {
		if err := insertRbacProjectGroup(ctx, db, dbRP.ID, rbGroup.GroupID); err != nil {
			return err
		}
	}
	*rp = dbRP.RbacProject
	return nil
}

func insertRbacProjectID(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacParentID int64, projectID int64) error {
	rgu := rbacProjectID{
		RbacProjectIdentifiers: sdk.RbacProjectIdentifiers{
			RbacProjectID: rbacParentID,
			ProjectID:     projectID,
		},
	}
	if err := gorpmapping.InsertAndSign(ctx, db, &rgu); err != nil {
		return err
	}
	return nil
}

func insertRbacProjectUser(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacProjectID int64, userID string) error {
	rgu := rbacProjectUser{
		RbacProjectID: rbacProjectID,
		RbacUser: sdk.RbacUser{
			UserID: userID,
		},
	}
	if err := gorpmapping.InsertAndSign(ctx, db, &rgu); err != nil {
		return err
	}
	return nil
}

func insertRbacProjectGroup(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacProjectID int64, groupID int64) error {
	rgu := rbacProjectGroup{
		RbacProjectID: rbacProjectID,
		RbacGroup: sdk.RbacGroup{
			GroupID: groupID,
		},
	}
	if err := gorpmapping.InsertAndSign(ctx, db, &rgu); err != nil {
		return err
	}
	return nil
}

func loadRbacProjectTargeted(ctx context.Context, db gorp.SqlExecutor, rbacProject *rbacProject) error {
	query := `
		SELECT p.id, p.name, p.description
		FROM rbac_project_ids rpi
		JOIN project p ON p.id = rpi.project_id
		WHERE rpi.rbac_project_id = $1`
	var rbacProjects []rbacProjectID
	if err := gorpmapping.GetAll(ctx, db, gorpmapping.NewQuery(query).Args(rbacProject.ID), &rbacProjects); err != nil {
		return err
	}
	rbacProject.Projects = make([]sdk.RbacProjectIdentifiers, 0, len(rbacProjects))
	for _, rp := range rbacProjects {
		rbacProject.Projects = append(rbacProject.Projects, rp.RbacProjectIdentifiers)
	}
	return nil
}

func loadRbacRbacProjectUsersTargeted(ctx context.Context, db gorp.SqlExecutor, rbacProject *rbacProject) error {
	query := `
		SELECT u.id, u.username
		FROM rbac_project_users rpu
		JOIN authentified_user u ON u.id = rpu.user_id
		WHERE rpu.rbac_project_id = $1
	`
	var users []rbacProjectUser
	if err := gorpmapping.GetAll(ctx, db, gorpmapping.NewQuery(query).Args(rbacProject.ID), &users); err != nil {
		return err
	}
	rbacProject.RbacUsers = make([]sdk.RbacUser, 0, len(users))
	for _, u := range users {
		rbacProject.RbacUsers = append(rbacProject.RbacUsers, u.RbacUser)
	}
	return nil
}

func loadRbacRbacProjectGroupsTargeted(ctx context.Context, db gorp.SqlExecutor, rbacProject *rbacProject) error {
	query := `
		SELECT g.id, g.name
		FROM rbac_project_groups rpg
		JOIN "group" g ON g.id = rpg.group_id
		WHERE rpg.rbac_project_id = $1
	`
	var groups []rbacProjectGroup
	if err := gorpmapping.GetAll(ctx, db, gorpmapping.NewQuery(query).Args(rbacProject.ID), &groups); err != nil {
		return err
	}
	rbacProject.RbacGroups = make([]sdk.RbacGroup, 0, len(groups))
	for _, g := range groups {
		rbacProject.RbacGroups = append(rbacProject.RbacGroups, g.RbacGroup)
	}
	return nil
}
