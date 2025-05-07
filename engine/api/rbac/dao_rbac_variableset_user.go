package rbac

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func loadRBACVariableSetUsers(ctx context.Context, db gorp.SqlExecutor, rbacVS *rbacVariableSet) error {
	q := gorpmapping.NewQuery("SELECT * FROM rbac_variableset_users WHERE rbac_variableset_id = $1").Args(rbacVS.ID)
	rbacUserIDS, err := getAllRBACVariableSetUsers(ctx, db, q)
	if err != nil {
		return err
	}
	rbacVS.RBACVariableSet.RBACUsersIDs = make([]string, 0, len(rbacUserIDS))
	for _, rbacUsers := range rbacUserIDS {
		rbacVS.RBACVariableSet.RBACUsersIDs = append(rbacVS.RBACVariableSet.RBACUsersIDs, rbacUsers.RbacVariableSetUserID)
	}
	return nil
}

func getAllRBACVariableSetUsers(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]rbacVariableSetUser, error) {
	var rbacVSUsers []rbacVariableSetUser
	if err := gorpmapping.GetAll(ctx, db, q, &rbacVSUsers); err != nil {
		return nil, err
	}

	usersFiltered := make([]rbacVariableSetUser, 0, len(rbacVSUsers))
	for _, rbacUsers := range rbacVSUsers {
		isValid, err := gorpmapping.CheckSignature(rbacUsers, rbacUsers.Signature)
		if err != nil {
			return nil, sdk.WrapError(err, "error when checking signature for rbac_variableset_users %d", rbacUsers.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac.getAllRBACVariableSetUsers> rbac_variableset_users %d data corrupted", rbacUsers.ID)
			continue
		}
		usersFiltered = append(usersFiltered, rbacUsers)
	}
	return usersFiltered, nil
}
