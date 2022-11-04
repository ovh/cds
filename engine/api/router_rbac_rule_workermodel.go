package api

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func (api *API) WorkerModelRead(ctx context.Context, c *sdk.AuthUserConsumer, cache cache.Store, db gorp.SqlExecutor, vars map[string]string) error {
	if getHatcheryConsumer(ctx) != nil {
		return nil
	}
	return rbac.ProjectRead(ctx, c, cache, db, vars)
}
