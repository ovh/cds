package api

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func (api *API) workerModelRead(ctx context.Context, c *sdk.AuthUserConsumer, cache cache.Store, db gorp.SqlExecutor, vars map[string]string) error {
	if getHatcheryConsumer(ctx) != nil {
		return nil
	}
	return api.projectRead(ctx, c, cache, db, vars)
}
