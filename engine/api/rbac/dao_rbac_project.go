package rbac

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

func insertRBACProject(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacProject *rbacProject) error {
	if err := gorpmapping.InsertAndSign(ctx, db, rbacProject); err != nil {
		return err
	}

	for _, projectKey := range rbacProject.RBACProjectKeys {
		if err := insertRBACProjectKey(ctx, db, rbacProject.ID, projectKey); err != nil {
			return err
		}
	}
	for _, rbUserID := range rbacProject.RBACUsersIDs {
		if err := insertRBACProjectUser(ctx, db, rbacProject.ID, rbUserID); err != nil {
			return err
		}
	}
	for _, rbGroupID := range rbacProject.RBACGroupsIDs {
		if err := insertRBACProjectGroup(ctx, db, rbacProject.ID, rbGroupID); err != nil {
			return err
		}
	}
	return nil
}

func insertRBACProjectKey(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacParentID int64, projectKey string) error {
	rpk := rbacProjectKey{
		RbacProjectID: rbacParentID,
		ProjectKey:    projectKey,
	}
	if err := gorpmapping.InsertAndSign(ctx, db, &rpk); err != nil {
		return err
	}
	return nil
}

func insertRBACProjectUser(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacProjectID int64, userID string) error {
	rgu := rbacProjectUser{
		RbacProjectID:     rbacProjectID,
		RbacProjectUserID: userID,
	}
	if err := gorpmapping.InsertAndSign(ctx, db, &rgu); err != nil {
		return err
	}
	return nil
}

func insertRBACProjectGroup(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacProjectID int64, groupID int64) error {
	rgu := rbacProjectGroup{
		RbacProjectID:      rbacProjectID,
		RbacProjectGroupID: groupID,
	}
	if err := gorpmapping.InsertAndSign(ctx, db, &rgu); err != nil {
		return err
	}
	return nil
}

func getAllRBACProjects(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]rbacProject, error) {
	var rbacProjects []rbacProject
	if err := gorpmapping.GetAll(ctx, db, q, &rbacProjects); err != nil {
		return nil, err
	}

	projectsFiltered := make([]rbacProject, 0, len(rbacProjects))
	for _, projectDatas := range rbacProjects {
		isValid, err := gorpmapping.CheckSignature(projectDatas, projectDatas.Signature)
		if err != nil {
			return nil, sdk.WrapError(err, "error when checking signature for rbac_project %d", projectDatas.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac.getAllRBACProjects> rbac_project %d data corrupted", projectDatas.ID)
			continue
		}
		projectsFiltered = append(projectsFiltered, projectDatas)
	}
	return projectsFiltered, nil
}

func loadRBACProjectsByRoleAndIDs(ctx context.Context, db gorp.SqlExecutor, role string, rbacProjectIDs []int64) ([]rbacProject, error) {
	q := gorpmapping.NewQuery(`SELECT * from rbac_project WHERE role = $1 AND id = ANY($2)`).Args(role, pq.Int64Array(rbacProjectIDs))
	return getAllRBACProjects(ctx, db, q)
}

func loadRBACProjectByRoleAndPublic(ctx context.Context, db gorp.SqlExecutor, role string) ([]rbacProject, error) {
	q := gorpmapping.NewQuery(`SELECT * from rbac_project WHERE role = $1 AND all_users = true`).Args(role)
	return getAllRBACProjects(ctx, db, q)
}
