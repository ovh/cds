package api

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func (api *API) isAdmin(ctx context.Context, _ *sdk.AuthUserConsumer, _ cache.Store, _ gorp.SqlExecutor, _ map[string]string) error {
	if isAdmin(ctx) {
		return nil
	}
	return sdk.WithStack(sdk.ErrForbidden)
}
