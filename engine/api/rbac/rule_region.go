package rbac

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	cdslog "github.com/ovh/cds/sdk/log"
)

func hasRoleOnRegion(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, regionIdentifier string, role string) error {
	if auth == nil {
		return sdk.WithStack(sdk.ErrForbidden)
	}

	hasRole, err := HasRoleOnRegionAndUserID(ctx, db, role, auth.AuthConsumerUser.AuthentifiedUser, regionIdentifier)
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

// RegionManage return nil if the current AuthConsumer have the RegionManage on current region ID
func RegionManage(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, vars map[string]string) error {
	regionIdentifier := vars["regionIdentifier"]
	return hasRoleOnRegion(ctx, auth, store, db, regionIdentifier, sdk.RegionRoleManage)
}

// RegionRead return nil if the current AuthConsumer have the ProjectRoleRead on current project KEY
func RegionRead(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, vars map[string]string) error {
	regionIdentifier := vars["regionIdentifier"]
	return hasRoleOnRegion(ctx, auth, store, db, regionIdentifier, sdk.RegionRoleRead)
}
