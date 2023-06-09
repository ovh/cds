package api

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
)

func hasRoleOnWorkflow(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, projectKey string, workflowName string, role string) error {
	if auth == nil {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	hasRole, err := rbac.HasRoleOnWorkflowAndUserID(ctx, db, role, auth.AuthConsumerUser.AuthentifiedUser.ID, projectKey, workflowName)
	if err != nil {
		return err
	}

	ctx = context.WithValue(ctx, cdslog.RbacRole, role)
	log.Info(ctx, "hasRole:%t", hasRole)

	if !hasRole {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	return nil
}

// workflowExecute return nil if the current AuthUserConsumer have the WorkflowRoleExecute on current workflow
func (api *API) workflowExecute(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, vars map[string]string) error {
	projectKey := vars["projectKey"]
  workflowName := vars["workflowName"]
	return hasRoleOnWorkflow(ctx, auth, store, db, projectKey, workflowName, sdk.WorkflowRoleExecute)
}
