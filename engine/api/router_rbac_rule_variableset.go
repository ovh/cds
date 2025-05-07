package api

import (
	"context"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/featureflipping"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
)

func (api *API) hasRoleOnVariableSet(ctx context.Context, vars map[string]string, role string) error {
	projectKey := vars["projectKey"]
	variableSetName := vars["variableSetName"]

	if supportMFA(ctx) && !isMFA(ctx) {
		_, requireMFA := featureflipping.IsEnabled(ctx, gorpmapping.Mapper, api.mustDBWithCtx(ctx), sdk.FeatureMFARequired, map[string]string{
			"project_key": projectKey,
		})
		if requireMFA {
			return sdk.WithStack(sdk.ErrMFARequired)
		}
	}

	auth := getUserConsumer(ctx)
	if auth == nil {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	hasRole, err := rbac.HasRoleOnVariableSetAndUserID(ctx, api.mustDBWithCtx(ctx), role, auth.AuthConsumerUser.AuthentifiedUser.ID, projectKey, variableSetName)
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
func (api *API) variableSetItemManage(ctx context.Context, vars map[string]string) error {
	return api.hasRoleOnVariableSet(ctx, vars, sdk.VariableSetRoleManageItem)
}

func (api *API) variableSetItemRead(ctx context.Context, vars map[string]string) error {
	return api.hasRoleOnVariableSet(ctx, vars, sdk.VariableSetRoleUse)
}
