package api

import (
	"context"
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func (api *API) isWorker(ctx context.Context, _ *sdk.AuthUserConsumer, _ cache.Store, _ gorp.SqlExecutor, _ map[string]string) error {
	hc := getHatcheryConsumer(ctx)
	work := getWorker(ctx)
	if hc != nil && work != nil {
		return nil
	}
	return sdk.WithStack(sdk.ErrForbidden)
}

func (api *API) workerGet(ctx context.Context, _ *sdk.AuthUserConsumer, _ cache.Store, _ gorp.SqlExecutor, _ map[string]string) error {
	if isCDN(ctx) {
		return nil
	}
	return sdk.WithStack(sdk.ErrForbidden)
}
