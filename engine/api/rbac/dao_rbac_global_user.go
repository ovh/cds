package rbac

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func loadRbacGlobalUsersByUserID(ctx context.Context, db gorp.SqlExecutor, userID string) ([]rbacGlobalUser, error) {
	q := gorpmapping.NewQuery("SELECT * FROM rbac_global_users WHERE user_id = $1").Args(userID)
	return getAllRBACGlobalUsers(ctx, db, q)
}

func loadRBACGlobalUsers(ctx context.Context, db gorp.SqlExecutor, rbacGlobal *rbacGlobal) error {
	q := gorpmapping.NewQuery("SELECT * FROM rbac_global_users WHERE rbac_global_id = $1").Args(rbacGlobal.ID)
	rbacUserIDS, err := getAllRBACGlobalUsers(ctx, db, q)
	if err != nil {
		return err
	}
	rbacGlobal.RBACUsersIDs = make([]string, 0, len(rbacUserIDS))
	for _, rbacUsers := range rbacUserIDS {
		rbacGlobal.RBACUsersIDs = append(rbacGlobal.RBACUsersIDs, rbacUsers.RbacGlobalUserID)
	}
	return nil
}

func getAllRBACGlobalUsers(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]rbacGlobalUser, error) {
	var rbacGlobalUsers []rbacGlobalUser
	if err := gorpmapping.GetAll(ctx, db, q, &rbacGlobalUsers); err != nil {
		return nil, err
	}

	usersFiltered := make([]rbacGlobalUser, 0, len(rbacGlobalUsers))
	for _, rbacUsers := range rbacGlobalUsers {
		isValid, err := gorpmapping.CheckSignature(rbacUsers, rbacUsers.Signature)
		if err != nil {
			return nil, sdk.WrapError(err, "error when checking signature for rbac_global_users %d", rbacUsers.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac.getAllRBACGlobalUsers> rbac_global_users %d data corrupted", rbacUsers.ID)
			continue
		}
		usersFiltered = append(usersFiltered, rbacUsers)
	}
	return usersFiltered, nil
}
