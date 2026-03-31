package api

import (
	"context"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/jws"
)

// RegisterLocalService registers a co-located service directly in the database,
// bypassing HTTP and token-based authentication. No consumer is created —
// authentication is handled via context injection by the LocalRoundTripper.
func (api *API) RegisterLocalService(ctx context.Context, data sdk.Service) error {
	tx, err := api.mustDB().Begin()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer tx.Rollback() //nolint

	if err := api.registerLocalServiceTx(ctx, tx, &data); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return sdk.WithStack(err)
	}

	log.Info(ctx, "RegisterLocalService> local service %s(%d) of type %s registered", data.Name, data.ID, data.Type)
	return nil
}

func (api *API) registerLocalServiceTx(ctx context.Context, tx gorpmapper.SqlExecutorWithTx, data *sdk.Service) error {
	if data.Name == "" || data.Type == "" {
		return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing service name or type")
	}

	// Try to find existing service by name and type
	existingSrv, err := services.LoadByNameAndType(ctx, tx, data.Name, data.Type)
	if err != nil && !sdk.ErrorIs(err, sdk.ErrNotFound) {
		return err
	}

	if existingSrv != nil {
		// Service already exists — update it
		existingSrv.Update(*data)
		existingSrv.LastHeartbeat = time.Now()
		if err := services.Update(ctx, tx, existingSrv); err != nil {
			return err
		}
		*data = *existingSrv
		return nil
	}

	// Insert a new service record without a consumer.
	// In local mode, authentication is handled via context injection,
	// so no consumer/session is needed.
	data.LastHeartbeat = time.Now()
	if err := services.Insert(ctx, tx, data); err != nil {
		return sdk.WrapError(err, "cannot insert local service %s", data.Name)
	}

	// For hatchery services, re-attach orphaned workers
	if data.Type == sdk.TypeHatchery {
		if err := worker.ReAttachAllToHatchery(ctx, tx, *data); err != nil {
			return err
		}
	}

	// Insert initial monitoring status
	if err := services.UpsertStatus(ctx, tx, *data, ""); err != nil {
		return sdk.WrapError(err, "cannot upsert status for local service %s", data.Name)
	}

	return nil
}

// APIPublicKey returns the API's public signing key in PEM format.
func (api *API) APIPublicKey() ([]byte, error) {
	return jws.ExportPublicKey(authentication.GetSigningKey())
}

// RegisterLocalHatchery creates the hatchery record, region, RBAC permission,
// and consumer directly in the database for a co-located hatchery.
// Returns the region name and the API's public signing key.
// Idempotent: reuses existing records if they already exist.
func (api *API) RegisterLocalHatchery(ctx context.Context, hatcheryName, regionName, modelType string) (string, []byte, error) {
	db := api.mustDB()

	// --- Region (idempotent) ---
	reg, err := region.LoadRegionByName(ctx, db, regionName)
	if err != nil {
		tx, err := db.Begin()
		if err != nil {
			return "", nil, sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint
		reg = &sdk.Region{Name: regionName}
		if err := region.Insert(ctx, tx, reg); err != nil {
			return "", nil, sdk.WrapError(err, "cannot create region %s", regionName)
		}
		if err := tx.Commit(); err != nil {
			return "", nil, sdk.WithStack(err)
		}
		log.Info(ctx, "RegisterLocalHatchery> region '%s' created", regionName)
	}

	// --- Hatchery (idempotent) ---
	h, err := hatchery.LoadHatcheryByName(ctx, db, hatcheryName)
	if err != nil {
		tx, err := db.Begin()
		if err != nil {
			return "", nil, sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint
		h = &sdk.Hatchery{Name: hatcheryName, ModelType: modelType}
		if err := hatchery.Insert(ctx, tx, h); err != nil {
			return "", nil, sdk.WrapError(err, "cannot create hatchery %s", hatcheryName)
		}
		if err := tx.Commit(); err != nil {
			return "", nil, sdk.WithStack(err)
		}
		log.Info(ctx, "RegisterLocalHatchery> hatchery '%s' created", hatcheryName)
	} else if h.ModelType != modelType {
		tx, err := db.Begin()
		if err != nil {
			return "", nil, sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint
		h.ModelType = modelType
		if err := hatchery.Update(ctx, tx, h); err != nil {
			return "", nil, sdk.WrapError(err, "cannot update hatchery %s model type", hatcheryName)
		}
		if err := tx.Commit(); err != nil {
			return "", nil, sdk.WithStack(err)
		}
		log.Info(ctx, "RegisterLocalHatchery> hatchery '%s' model type updated to '%s'", hatcheryName, modelType)
	}

	// --- Hatchery consumer (idempotent) ---
	_, err = authentication.LoadHatcheryConsumerByName(ctx, db, hatcheryName)
	if err != nil {
		tx, err := db.Begin()
		if err != nil {
			return "", nil, sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint
		if _, err := authentication.NewConsumerHatchery(ctx, tx, *h); err != nil {
			return "", nil, sdk.WrapError(err, "cannot create hatchery consumer")
		}
		if err := tx.Commit(); err != nil {
			return "", nil, sdk.WithStack(err)
		}
		log.Info(ctx, "RegisterLocalHatchery> hatchery consumer created")
	}

	// --- RBAC (idempotent) ---
	rbacName := "perm-hatchery-" + hatcheryName
	if _, err := rbac.LoadRBACByName(ctx, db, rbacName); err != nil {
		tx, err := db.Begin()
		if err != nil {
			return "", nil, sdk.WithStack(err)
		}
		defer tx.Rollback() // nolint
		rb := sdk.RBAC{
			Name: rbacName,
			Hatcheries: []sdk.RBACHatchery{
				{
					Role:       sdk.HatcheryRoleSpawn,
					RegionID:   reg.ID,
					HatcheryID: h.ID,
				},
			},
		}
		if err := rbac.Insert(ctx, tx, &rb); err != nil {
			return "", nil, sdk.WrapError(err, "cannot create RBAC %s", rbacName)
		}
		if err := tx.Commit(); err != nil {
			return "", nil, sdk.WithStack(err)
		}
		log.Info(ctx, "RegisterLocalHatchery> RBAC '%s' created", rbacName)
	}

	// Export API public signing key
	pubKey, err := jws.ExportPublicKey(authentication.GetSigningKey())
	if err != nil {
		return "", nil, sdk.WrapError(err, "cannot export API public key")
	}

	log.Info(ctx, "RegisterLocalHatchery> hatchery '%s' ready on region '%s'", hatcheryName, regionName)
	return regionName, pubKey, nil
}
