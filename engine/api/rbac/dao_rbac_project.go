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

func insertRbacProject(ctx context.Context, db gorpmapper.SqlExecutorWithTx, dbRP *rbacProject) error {
	if err := gorpmapping.InsertAndSign(ctx, db, dbRP); err != nil {
		return err
	}

	for _, rbProjectID := range dbRP.RBACProjectsIDs {
		if err := insertRbacProjectIdentifiers(ctx, db, dbRP.ID, rbProjectID); err != nil {
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

func insertRbacProjectIdentifiers(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacParentID int64, projectID int64) error {
	identifier := rbacProjectIdentifiers{
		RbacProjectID: rbacParentID,
		ProjectID:     projectID,
	}
	if err := gorpmapping.InsertAndSign(ctx, db, &identifier); err != nil {
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

func getAllRbacProjects(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]rbacProject, error) {
	var rbacProjects []rbacProject
	if err := gorpmapping.GetAll(ctx, db, q, &rbacProjects); err != nil {
		return nil, err
	}
	for _, projectDatas := range rbacProjects {
		isValid, err := gorpmapping.CheckSignature(projectDatas, projectDatas.Signature)
		if err != nil {
			return nil, sdk.WrapError(err, "error when checking signature for rbac_project %d", projectDatas.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac.getAllRBACGlobalUsers> rbac_project %d data corrupted", projectDatas.ID)
			continue
		}
	}
	return rbacProjects, nil
}

func loadRbacProjectsByRoleAndIDs(ctx context.Context, db gorp.SqlExecutor, role string, rbacProjectIDs []int64) ([]rbacProject, error) {
	q := gorpmapping.NewQuery(`SELECT * from rbac_project WHERE role = $1 AND id = ANY($2)`).Args(role, pq.Int64Array(rbacProjectIDs))
	return getAllRbacProjects(ctx, db, q)
}
