package api

import (
	"context"
	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

// jobRunList only the hatchery can list job runs
func (api *API) jobRunList(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, vars map[string]string) error {
	hatchConsumer := getHatcheryConsumer(ctx)
	switch {
	case hatchConsumer != nil:
		return hatcheryHasRoleOnRegion(ctx, db, hatchConsumer.AuthConsumerHatchery.HatcheryID, vars["regionName"], sdk.HatcheryRoleSpawn)
	}
	// TODO manage users
	return sdk.WithStack(sdk.ErrForbidden)
}

// jobRunRead only hatchery can read a job run for now
func (api *API) jobRunRead(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, vars map[string]string) error {
	hatchConsumer := getHatcheryConsumer(ctx)
	switch {
	case hatchConsumer != nil:
		return hatcheryHasRoleOnRegion(ctx, db, hatchConsumer.AuthConsumerHatchery.HatcheryID, vars["regionName"], sdk.HatcheryRoleSpawn)
	}
	// TODO manage worker
	return sdk.WithStack(sdk.ErrForbidden)
}

// jobRunUpdate only hatchery and worker can update a job run
func (api *API) jobRunUpdate(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, vars map[string]string) error {
	hatchConsumer := getHatcheryConsumer(ctx)
	switch {
	case hatchConsumer != nil:
		return hatcheryHasRoleOnRegion(ctx, db, hatchConsumer.AuthConsumerHatchery.HatcheryID, vars["regionName"], sdk.HatcheryRoleSpawn)
	}
	// TODO manage worker
	return sdk.WithStack(sdk.ErrForbidden)
}
