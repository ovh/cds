package rbac

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
)

func hasGlobalRole(ctx context.Context, auth *sdk.AuthUserConsumer, _ cache.Store, db gorp.SqlExecutor, role string) error {
	if auth == nil {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	hasRole, err := HasGlobalRole(ctx, db, role, auth.AuthConsumerUser.AuthentifiedUser.ID)
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
func GlobalPermissionManage(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, _ map[string]string) error {
	return hasGlobalRole(ctx, auth, store, db, sdk.GlobalRoleManagePermission)
}

func GlobalOrganizationManage(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, _ map[string]string) error {
	return hasGlobalRole(ctx, auth, store, db, sdk.GlobalRoleManageOrganization)
}

func GlobalRegionManage(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, _ map[string]string) error {
	return hasGlobalRole(ctx, auth, store, db, sdk.GlobalRoleManageRegion)
}

func GlobalHatcheryManage(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, _ map[string]string) error {
	return hasGlobalRole(ctx, auth, store, db, sdk.GlobalRoleManageHatchery)
}
