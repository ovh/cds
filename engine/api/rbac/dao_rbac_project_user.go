package rbac

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func loadRbacProjectUsersByUserID(ctx context.Context, db gorp.SqlExecutor, userID string) ([]rbacProjectUser, error) {
	q := gorpmapping.NewQuery("SELECT * FROM rbac_project_users WHERE user_id = $1").Args(userID)
	return getAllRBACProjectUsers(ctx, db, q)
}

func loadRBACProjectUsers(ctx context.Context, db gorp.SqlExecutor, rbacProject *rbacProject) error {
	q := gorpmapping.NewQuery("SELECT * FROM rbac_project_users WHERE rbac_project_id = $1").Args(rbacProject.ID)
	rbacUserIDS, err := getAllRBACProjectUsers(ctx, db, q)
	if err != nil {
		return err
	}
	rbacProject.RBACProject.RBACUsersIDs = make([]string, 0, len(rbacUserIDS))
	for _, rbacUsers := range rbacUserIDS {
		rbacProject.RBACProject.RBACUsersIDs = append(rbacProject.RBACProject.RBACUsersIDs, rbacUsers.RbacProjectUserID)
	}
	return nil
}

func getAllRBACProjectUsers(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]rbacProjectUser, error) {
	var rbacProjectUsers []rbacProjectUser
	if err := gorpmapping.GetAll(ctx, db, q, &rbacProjectUsers); err != nil {
		return nil, err
	}
	for _, rbacUsers := range rbacProjectUsers {
		isValid, err := gorpmapping.CheckSignature(rbacUsers, rbacUsers.Signature)
		if err != nil {
			return nil, sdk.WrapError(err, "error when checking signature for rbac_project_users %d", rbacUsers.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac.getAllRBACGlobalUsers> rbac_project_user %d data corrupted", rbacUsers.ID)
			continue
		}
	}
	return rbacProjectUsers, nil
}
