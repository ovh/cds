package api

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (api *API) isWorker(ctx context.Context, _ map[string]string) error {
	hc := getHatcheryConsumer(ctx)
	work := getWorker(ctx)
	if hc != nil && work != nil {
		return nil
	}
	return sdk.WithStack(sdk.ErrForbidden)
}

func (api *API) workerGet(ctx context.Context, _ map[string]string) error {
	if isCDN(ctx) {
		return nil
	}
	return sdk.WithStack(sdk.ErrForbidden)
}

func (api *API) workerList(ctx context.Context, _ map[string]string) error {
	hc := getHatcheryConsumer(ctx)
	if isAdmin(ctx) || hc != nil {
		return nil
	}
	return sdk.WithStack(sdk.ErrForbidden)
}
