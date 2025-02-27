package api

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (api *API) pluginRead(ctx context.Context, _ map[string]string) error {
	// Old worker
	if isWorker(ctx) || getUserConsumer(ctx) != nil {
		return nil
	}
	// New worker
	if getWorker(ctx) != nil {
		return nil
	}

	return sdk.WithStack(sdk.ErrForbidden)
}
