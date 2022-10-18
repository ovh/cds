package rbac

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func hasGlobalRole(ctx context.Context, auth *sdk.AuthUserConsumer, _ cache.Store, db gorp.SqlExecutor, role string) error {
	if auth == nil {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	hasRole, err := HasGlobalRole(ctx, db, role, auth.AuthConsumerUser.AuthentifiedUser.ID)
	if err != nil {
		return err
	}

	log.RegisterField(LogFieldRole)
	ctx = context.WithValue(ctx, LogFieldRole, role)
	log.Info(ctx, "hasRole:%t", hasRole)

	if !hasRole {
		return sdk.WithStack(sdk.ErrForbidden)
	}
	return nil
}

// PermissionManage return nil if the current AuthUserConsumer have the ProjectRoleManage on current project KEY
func PermissionManage(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, _ map[string]string) error {
	return hasGlobalRole(ctx, auth, store, db, sdk.GlobalRoleManagePermission)
}

func OrganizationManage(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, _ map[string]string) error {
	return hasGlobalRole(ctx, auth, store, db, sdk.GlobalRoleManageOrganization)
}

func RegionManage(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, _ map[string]string) error {
	return hasGlobalRole(ctx, auth, store, db, sdk.GlobalRoleManageRegion)
}

func GlobalHatcheryManage(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, _ map[string]string) error {
	return hasGlobalRole(ctx, auth, store, db, sdk.GlobalRoleManageHatchery)
}
