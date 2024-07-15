package api

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
)

func hasRoleOnVariableSet(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, projectKey string, variableset string, role string) error {
	if auth == nil {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	hasRole, err := rbac.HasRoleOnVariableSetAndUserID(ctx, db, role, auth.AuthConsumerUser.AuthentifiedUser.ID, projectKey, variableset)
	if err != nil {
		return err
	}

	ctx = context.WithValue(ctx, cdslog.RbacRole, role)
	if !hasRole {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	return nil
}

// workflowTrigger return nil if the current AuthUserConsumer have the WorkflowRoleTrigger on current workflow
func (api *API) variableSetItemManage(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, vars map[string]string) error {
	projectKey := vars["projectKey"]
	vsName := vars["variableSetName"]

	return hasRoleOnVariableSet(ctx, auth, store, db, projectKey, vsName, sdk.VariableSetRoleManageItem)
}

func (api *API) variableSetItemRead(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, vars map[string]string) error {
	projectKey := vars["projectKey"]
	vsName := vars["variableSetName"]

	return hasRoleOnVariableSet(ctx, auth, store, db, projectKey, vsName, sdk.VariableSetRoleUse)
}
