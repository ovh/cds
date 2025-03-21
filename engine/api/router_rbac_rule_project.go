package api

import (
	"context"

	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/featureflipping"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
)

func (api *API) hasRoleOnProject(ctx context.Context, vars map[string]string, role string) error {
	projectKey := vars["projectKey"]

	c := getUserConsumer(ctx)
	if c == nil {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	if supportMFA(ctx) && !isMFA(ctx) {
		_, requireMFA := featureflipping.IsEnabled(ctx, gorpmapping.Mapper, api.mustDBWithCtx(ctx), sdk.FeatureMFARequired, map[string]string{
			"project_key": projectKey,
		})
		if requireMFA {
			return sdk.WithStack(sdk.ErrMFARequired)
		}
	}

	hasRole, err := rbac.HasRoleOnProjectAndUserID(ctx, api.mustDBWithCtx(ctx), role, c.AuthConsumerUser.AuthentifiedUser.ID, projectKey)
	if err != nil {
		return err
	}

	ctx = context.WithValue(ctx, cdslog.RbacRole, role)
	if hasRole {
		return nil
	}

	return sdk.WithStack(sdk.ErrForbidden)
}

// ProjectManage return nil if the current AuthUserConsumer have the ProjectRoleManage on current project KEY
func (api *API) projectManage(ctx context.Context, vars map[string]string) error {
	return api.hasRoleOnProject(ctx, vars, sdk.ProjectRoleManage)
}

// projectManageNotification return nil if the current AuthUserConsumer have the role ProjectRoleManageNotification on current project KEY
func (api *API) projectManageNotification(ctx context.Context, vars map[string]string) error {
	return api.hasRoleOnProject(ctx, vars, sdk.ProjectRoleManageNotification)
}

// projectManageVariableSet return nil if the current AuthUserConsumer have the role ProjectRoleManageVariableSet on current project KEY
func (api *API) projectManageVariableSet(ctx context.Context, vars map[string]string) error {
	return api.hasRoleOnProject(ctx, vars, sdk.ProjectRoleManageVariableSet)
}

// ProjectRead return nil if the current AuthUserConsumer have the ProjectRoleRead on current project KEY
func (api *API) projectRead(ctx context.Context, vars map[string]string) error {
	entityType := vars["entityType"]
	hatch := getHatcheryConsumer(ctx)

	// hatchery can get every worker model
	if hatch != nil && entityType == sdk.EntityTypeWorkerModel {
		return nil
	}

	return api.hasRoleOnProject(ctx, vars, sdk.ProjectRoleRead)
}

func (api *API) analysisRead(ctx context.Context, vars map[string]string) error {
	if isHooks(ctx) {
		return nil
	}
	return api.projectRead(ctx, vars)
}
