package api

import (
	"context"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func hatcheryHasRoleOnRegion(ctx context.Context, db gorp.SqlExecutor, hatcheryID string, regionName string, role string) error {
	perm, err := rbac.LoadRBACHatcheryByHatcheryID(ctx, db, hatcheryID)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNoAction) {
			return sdk.WithStack(sdk.ErrForbidden)
		}
		return err
	}
	reg, err := region.LoadRegionByName(ctx, db, regionName)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNoAction) {
			return sdk.WithStack(sdk.ErrForbidden)
		}
		return err
	}
	if perm.RegionID == reg.ID && perm.Role == role {
		return nil
	}
	return sdk.WithStack(sdk.ErrForbidden)
}

func (api *API) isHatchery(ctx context.Context, _ *sdk.AuthUserConsumer, _ cache.Store, _ gorp.SqlExecutor, _ map[string]string) error {
  if getHatcheryConsumer(ctx) != nil && getWorker(ctx) == nil {
		return nil
	}
	return sdk.WithStack(sdk.ErrForbidden)
}
