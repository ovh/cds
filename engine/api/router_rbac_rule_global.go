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

func hasGlobalRole(ctx context.Context, auth *sdk.AuthUserConsumer, _ cache.Store, db gorp.SqlExecutor, role string) error {
	if auth == nil {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	hasRole, err := rbac.HasGlobalRole(ctx, db, role, auth.AuthConsumerUser.AuthentifiedUser.ID)
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

// GlobalPermissionManage return nil if the current AuthConsumer have the ProjectRoleManage on current project KEY
func (api *API) globalPermissionManage(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, _ map[string]string) error {
	return hasGlobalRole(ctx, auth, store, db, sdk.GlobalRoleManagePermission)
}

func (api *API) globalOrganizationManage(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, _ map[string]string) error {
	return hasGlobalRole(ctx, auth, store, db, sdk.GlobalRoleManageOrganization)
}

func (api *API) globalRegionManage(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, _ map[string]string) error {
	return hasGlobalRole(ctx, auth, store, db, sdk.GlobalRoleManageRegion)
}

func (api *API) globalHatcheryManage(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, _ map[string]string) error {
	return hasGlobalRole(ctx, auth, store, db, sdk.GlobalRoleManageHatchery)
}
