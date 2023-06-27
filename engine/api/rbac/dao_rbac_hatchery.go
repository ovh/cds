package rbac

import (
  "context"

  "github.com/go-gorp/gorp"
  "github.com/ovh/cds/sdk"
  "github.com/rockbears/log"

  "github.com/ovh/cds/engine/api/database/gorpmapping"
  "github.com/ovh/cds/engine/gorpmapper"
  "github.com/ovh/cds/sdk/telemetry"
)

func insertRBACHatchery(ctx context.Context, db gorpmapper.SqlExecutorWithTx, rbacHatchery *rbacHatchery) error {
  if err := gorpmapping.InsertAndSign(ctx, db, rbacHatchery); err != nil {
    return err
  }
  return nil
}

func getAllRBACHatchery(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) ([]rbacHatchery, error) {
  var rbacHatcheries []rbacHatchery
  if err := gorpmapping.GetAll(ctx, db, q, &rbacHatcheries); err != nil {
    return nil, err
  }

  hatcheriesFiltered := make([]rbacHatchery, 0, len(rbacHatcheries))
  for _, rbacHatch := range rbacHatcheries {
    isValid, err := gorpmapping.CheckSignature(rbacHatch, rbacHatch.Signature)
    if err != nil {
      return nil, sdk.WrapError(err, "error when checking signature for rbac_hatchery %d", rbacHatch.ID)
    }
    if !isValid {
      log.Error(ctx, "rbac.getAllRBACHatchery> rbac_hatchery %d data corrupted", rbacHatch.ID)
      continue
    }
    hatcheriesFiltered = append(hatcheriesFiltered, rbacHatch)
  }
  return hatcheriesFiltered, nil
}

func getRBACHatchery(ctx context.Context, db gorp.SqlExecutor, q gorpmapping.Query) (*rbacHatchery, error) {
  var rbacHatch rbacHatchery
  found, err := gorpmapping.Get(ctx, db, q, &rbacHatch)
  if err != nil {
    return nil, err
  }
  if !found {
    return nil, sdk.ErrNotFound
  }

  isValid, err := gorpmapping.CheckSignature(rbacHatch, rbacHatch.Signature)
  if err != nil {
    return nil, sdk.WrapError(err, "error when checking signature for rbac_hatchery %d", rbacHatch.ID)
  }
  if !isValid {
    log.Error(ctx, "rbac.getRBACHatchery> rbac_hatchery %d data corrupted", rbacHatch.ID)
    return nil, sdk.ErrNotFound
  }

  return &rbacHatch, nil
}

func LoadRBACByHatcheryID(ctx context.Context, db gorp.SqlExecutor, hatcheryID string) (*sdk.RBAC, error) {
  ctx, next := telemetry.Span(ctx, "hatchery.LoadRBACByHatcheryID")
  defer next()
  query := gorpmapping.NewQuery(`SELECT * FROM rbac_hatchery WHERE hatchery_id = $1`).Args(hatcheryID)
  rbHatchery, err := getRBACHatchery(ctx, db, query)
  if err != nil {
    return nil, err
  }
  return LoadRBACByID(ctx, db, rbHatchery.RbacID, LoadOptions.LoadRBACHatchery)

}

func loadRBacIDsByHatcheryRegionID(ctx context.Context, db gorp.SqlExecutor, regionID string) ([]string, error) {
  query := gorpmapping.NewQuery(`SELECT * FROM rbac_hatchery WHERE region_id = $1`).Args(regionID)
  rbHatcheries, err := getAllRBACHatchery(ctx, db, query)
  if err != nil {
    return nil, err
  }
  rbacIDs := make(sdk.StringSlice, 0)
  for _, rh := range rbHatcheries {
    rbacIDs = append(rbacIDs, rh.RbacID)
  }
  rbacIDs.Unique()
  return rbacIDs, nil
}
