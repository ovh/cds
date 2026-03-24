package api

import (
	"context"
	"time"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/worker"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
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
