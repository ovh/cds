package rbac

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func loadRBACWorkflowGroups(ctx context.Context, db gorp.SqlExecutor, rbacWorkflow *rbacWorkflow) error {
	q := gorpmapping.NewQuery("SELECT * FROM rbac_workflow_groups WHERE rbac_workflow_id = $1").Args(rbacWorkflow.ID)
	rbacWorkflowGroups, err := getAllRBACWorkflowGroups(ctx, db, q)
	if err != nil {
		return err
	}
	rbacWorkflow.RBACWorkflow.RBACGroupsIDs = make([]int64, 0, len(rbacWorkflowGroups))
	for _, g := range rbacWorkflowGroups {
		rbacWorkflow.RBACWorkflow.RBACGroupsIDs = append(rbacWorkflow.RBACWorkflow.RBACGroupsIDs, g.RbacWorkflowGroupID)
	}
	return nil
}

func getAllRBACWorkflowGroups(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]rbacWorkflowGroup, error) {
	var rbacGroupIDs []rbacWorkflowGroup
	if err := gorpmapping.GetAll(ctx, db, q, &rbacGroupIDs); err != nil {
		return nil, err
	}

	groupsFiltered := make([]rbacWorkflowGroup, 0, len(rbacGroupIDs))
	for _, rbacGroups := range rbacGroupIDs {
		isValid, err := gorpmapping.CheckSignature(rbacGroups, rbacGroups.Signature)
		if err != nil {
			return nil, sdk.WrapError(err, "error when checking signature for rbac_workflow_groups %d", rbacGroups.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac.getAllRBACWorkflowGroups> rbac_workflow_groups %d data corrupted", rbacGroups.ID)
			continue
		}
		groupsFiltered = append(groupsFiltered, rbacGroups)
	}
	return groupsFiltered, nil
}
