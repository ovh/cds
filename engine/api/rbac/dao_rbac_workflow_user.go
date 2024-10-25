package rbac

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func loadRBACWorkflowUsers(ctx context.Context, db gorp.SqlExecutor, rbacWorkflow *rbacWorkflow) error {
	q := gorpmapping.NewQuery("SELECT * FROM rbac_workflow_users WHERE rbac_workflow_id = $1").Args(rbacWorkflow.ID)
	rbacUserIDS, err := getAllRBACWorkflowUsers(ctx, db, q)
	if err != nil {
		return err
	}
	rbacWorkflow.RBACWorkflow.RBACUsersIDs = make([]string, 0, len(rbacUserIDS))
	for _, rbacUsers := range rbacUserIDS {
		rbacWorkflow.RBACWorkflow.RBACUsersIDs = append(rbacWorkflow.RBACWorkflow.RBACUsersIDs, rbacUsers.RbacWorkflowUserID)
	}
	return nil
}

func getAllRBACWorkflowUsers(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]rbacWorkflowUser, error) {
	var rbacWorkflowUsers []rbacWorkflowUser
	if err := gorpmapping.GetAll(ctx, db, q, &rbacWorkflowUsers); err != nil {
		return nil, err
	}

	usersFiltered := make([]rbacWorkflowUser, 0, len(rbacWorkflowUsers))
	for _, rbacUsers := range rbacWorkflowUsers {
		isValid, err := gorpmapping.CheckSignature(rbacUsers, rbacUsers.Signature)
		if err != nil {
			return nil, sdk.WrapError(err, "error when checking signature for rbac_workflow_users %d", rbacUsers.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac.getAllRBACWorkflowUsers> rbac_workflow_users %d data corrupted", rbacUsers.ID)
			continue
		}
		usersFiltered = append(usersFiltered, rbacUsers)
	}
	return usersFiltered, nil
}
