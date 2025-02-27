package api

import (
	"context"

	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
)

func (api *API) hasGlobalRole(ctx context.Context, role string) error {
	auth := getUserConsumer(ctx)
	if auth == nil {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	hasRole, err := rbac.HasGlobalRole(ctx, api.mustDBWithCtx(ctx), role, auth.AuthConsumerUser.AuthentifiedUser.ID)
	if err != nil {
		return err
	}

	ctx = context.WithValue(ctx, cdslog.RbacRole, role)
	if !hasRole {
		return sdk.WithStack(sdk.ErrForbidden)
	}
	return nil
}

// GlobalPermissionManage return nil if the current AuthConsumer have the ProjectRoleManage on current project KEY
func (api *API) globalPermissionManage(ctx context.Context, _ map[string]string) error {
	return api.hasGlobalRole(ctx, sdk.GlobalRoleManagePermission)
}

func (api *API) globalOrganizationManage(ctx context.Context, _ map[string]string) error {
	return api.hasGlobalRole(ctx, sdk.GlobalRoleManageOrganization)
}

func (api *API) globalRegionManage(ctx context.Context, _ map[string]string) error {
	return api.hasGlobalRole(ctx, sdk.GlobalRoleManageRegion)
}

func (api *API) globalHatcheryManage(ctx context.Context, _ map[string]string) error {
	return api.hasGlobalRole(ctx, sdk.GlobalRoleManageHatchery)
}

func (api *API) globalPluginManage(ctx context.Context, _ map[string]string) error {
	return api.hasGlobalRole(ctx, sdk.GlobalRoleManagePlugin)
}
