package api

import (
	"context"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

// jobRunList only the hatchery can list job runs
func (api *API) jobRunList(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, vars map[string]string) error {
	if getHatcheryConsumer(ctx) != nil || auth != nil {
		return nil
	}
	return sdk.WithStack(sdk.ErrForbidden)
}

// jobRunRead only hatchery can read a job run for now
func (api *API) jobRunRead(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, vars map[string]string) error {
	if getHatcheryConsumer(ctx) != nil {
		return nil
	}
	return sdk.WithStack(sdk.ErrForbidden)
}

// jobRunUpdate only hatchery and worker can update a job run
func (api *API) jobRunUpdate(ctx context.Context, auth *sdk.AuthUserConsumer, store cache.Store, db gorp.SqlExecutor, vars map[string]string) error {
	if getHatcheryConsumer(ctx) != nil {
		return nil
	}
	return sdk.WithStack(sdk.ErrForbidden)
}
