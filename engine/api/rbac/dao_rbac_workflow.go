package rbac

import (
	"context"
	"fmt"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/glob"
	"github.com/ovh/cds/sdk/telemetry"
)

func insertRBACWorkflow(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacWorkflow *rbacWorkflow) error {
	if err := gorpmapping.InsertAndSign(ctx, db, rbacWorkflow); err != nil {
		return err
	}

	for _, rbUserID := range rbacWorkflow.RBACUsersIDs {
		if err := insertRBACWorkflowUser(ctx, db, rbacWorkflow.ID, rbUserID); err != nil {
			return err
		}
	}
	for _, rbGroupID := range rbacWorkflow.RBACGroupsIDs {
		if err := insertRBACWorkflowGroup(ctx, db, rbacWorkflow.ID, rbGroupID); err != nil {
			return err
		}
	}
	return nil
}

func insertRBACWorkflowUser(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacWorkflowID int64, userID string) error {
	rgu := rbacWorkflowUser{
		RbacWorkflowID:     rbacWorkflowID,
		RbacWorkflowUserID: userID,
	}
	if err := gorpmapping.InsertAndSign(ctx, db, &rgu); err != nil {
		return err
	}
	return nil
}

func insertRBACWorkflowGroup(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacWorkflowID int64, groupID int64) error {
	rgu := rbacWorkflowGroup{
		RbacWorkflowID:      rbacWorkflowID,
		RbacWorkflowGroupID: groupID,
	}
	if err := gorpmapping.InsertAndSign(ctx, db, &rgu); err != nil {
		return err
	}
	return nil
}

func getAllRBACWorkflows(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]rbacWorkflow, error) {
	var rbacWorkflows []rbacWorkflow
	if err := gorpmapping.GetAll(ctx, db, q, &rbacWorkflows); err != nil {
		return nil, err
	}

	worflowsFiltered := make([]rbacWorkflow, 0, len(rbacWorkflows))
	for _, wDatas := range rbacWorkflows {
		isValid, err := gorpmapping.CheckSignature(wDatas, wDatas.Signature)
		if err != nil {
			return nil, sdk.WrapError(err, "error when checking signature for rbac_workflow %d", wDatas.ID)
		}
		if !isValid {
			log.Error(ctx, "rbac.getAllRBACWorkflows> rbac_workflow %d data corrupted", wDatas.ID)
			continue
		}
		worflowsFiltered = append(worflowsFiltered, wDatas)
	}
	return worflowsFiltered, nil
}

func HasRoleOnWorkflowAndVCSUsername(ctx context.Context, db gorp.SqlExecutor, role string, VCSUser sdk.RBACVCSUser, projectKey string, vcs, repo, workflowName string) (bool, error) {
	workflowNamePerm := fmt.Sprintf("%s/%s/%s", vcs, repo, workflowName)

	workflows, allWorkflowAllowed, err := LoadAllWorkflowsAllowedForVCSUSer(ctx, db, role, projectKey, VCSUser)
	if err != nil {
		return false, err
	}
	log.Info(ctx, "HasRoleOnWorkflowAndVCSUsername> granted workflows for %s/%s: %+v , allWorkflowAllowed=%v", VCSUser.VCSServer, VCSUser.VCSUsername, workflows, allWorkflowAllowed)

	if allWorkflowAllowed {
		return true, nil
	}
	for _, item := range workflows {
		g := glob.New(item)
		r, err := g.MatchString(workflowNamePerm)
		if err != nil {
			return false, err
		}
		if r != nil {
			return true, nil
		}
	}
	return false, nil
}

func HasRoleOnWorkflowAndUserID(ctx context.Context, db gorp.SqlExecutor, role string, userID string, projectKey string, vcs, repo, workflowName string) (bool, error) {
	ctx, next := telemetry.Span(ctx, "rbac.HasRoleOnWorkflowAndUserID")
	defer next()

	workflowNamePerm := fmt.Sprintf("%s/%s/%s", vcs, repo, workflowName)

	workflows, allWorkflowAllowed, err := LoadAllWorkflowsAllowed(ctx, db, role, projectKey, userID)
	if err != nil {
		return false, err
	}
	if allWorkflowAllowed {
		return true, nil
	}
	for _, item := range workflows {
		g := glob.New(item)
		r, err := g.MatchString(workflowNamePerm)
		if err != nil {
			return false, err
		}
		if r != nil {
			return true, nil
		}
	}
	return false, nil
}

func LoadAllWorkflowsAllowedForVCSUSer(ctx context.Context, db gorp.SqlExecutor, role string, projectKey string, user sdk.RBACVCSUser) (sdk.StringSlice, bool, error) {
	workflows := sdk.StringSlice{}

	rbacWorkflows, err := loadRBACWorkflowsByProjectAndRole(ctx, db, projectKey, role)
	if err != nil {
		return nil, false, err
	}

	for _, rw := range rbacWorkflows {
		if rw.AllUsers {
			if rw.AllWorkflows {
				return nil, true, nil
			}
			workflows = append(workflows, rw.RBACWorkflowsNames...)
			continue
		}
		for _, rbacVCSUser := range rw.RBACVCSUsers {
			log.Info(ctx, "LoadAllWorkflowsAllowedForVCSUSer> checking %s/%s against %s/%s", user.VCSServer, user.VCSUsername, rbacVCSUser.VCSServer, rbacVCSUser.VCSUsername)
			if rbacVCSUser.VCSServer == user.VCSServer && rbacVCSUser.VCSUsername == user.VCSUsername {
				if rw.AllWorkflows {
					log.Info(ctx, "LoadAllWorkflowsAllowedForVCSUSer> %s/%s is allowed on all workflows of project %s", projectKey)
					return nil, true, nil
				}
				log.Info(ctx, "LoadAllWorkflowsAllowedForVCSUSer> %s/%s is allowed on workflows %+v of project %s", rw.RBACWorkflowsNames, projectKey)
				workflows = append(workflows, rw.RBACWorkflowsNames...)
				break
			}
		}
	}

	return workflows, false, nil
}

func LoadAllWorkflowsAllowed(ctx context.Context, db gorp.SqlExecutor, role string, projectKey string, userID string) (sdk.StringSlice, bool, error) {
	workflows := sdk.StringSlice{}

	groups, err := group.LoadAllByUserID(ctx, db, userID)
	if err != nil {
		return nil, false, err
	}
	groupIDs := make(sdk.Int64Slice, 0, len(groups))
	for _, g := range groups {
		groupIDs = append(groupIDs, g.ID)
	}

	rbacWorkflows, err := loadRBACWorkflowsByProjectAndRole(ctx, db, projectKey, role)
	if err != nil {
		return nil, false, err
	}

	for _, rw := range rbacWorkflows {
		if rw.AllUsers {
			if rw.AllWorkflows {
				return nil, true, nil
			}
			workflows = append(workflows, rw.RBACWorkflowsNames...)
			continue
		}
		for _, rbacUserID := range rw.RBACUsersIDs {
			if rbacUserID == userID {
				if rw.AllWorkflows {
					return nil, true, nil
				}
				workflows = append(workflows, rw.RBACWorkflowsNames...)
				continue
			}
		}
		for _, groupID := range rw.RBACGroupsIDs {
			if groupIDs.Contains(groupID) {
				if rw.AllWorkflows {
					return nil, true, nil
				}
				workflows = append(workflows, rw.RBACWorkflowsNames...)
				continue
			}
		}

	}
	return workflows, false, nil
}

func loadRBACWorkflowsByProjectAndRole(ctx context.Context, db gorp.SqlExecutor, projectKey string, role string) ([]rbacWorkflow, error) {
	query := gorpmapping.NewQuery(`SELECT * FROM rbac_workflow WHERE project_key = $1 AND role = $2`).Args(projectKey, role)
	rbacWorkflows, err := getAllRBACWorkflows(ctx, db, query)
	if err != nil {
		return nil, err
	}
	for i := range rbacWorkflows {
		rw := &rbacWorkflows[i]
		if !rw.AllUsers {
			if err := loadRBACWorkflowGroups(ctx, db, rw); err != nil {
				return nil, err
			}
			if err := loadRBACWorkflowUsers(ctx, db, rw); err != nil {
				return nil, err
			}
		}
	}
	return rbacWorkflows, nil
}
