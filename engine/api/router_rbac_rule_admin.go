package api

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (api *API) isAdmin(ctx context.Context, _ map[string]string) error {
	if isAdmin(ctx) {
		return nil
	}
	return sdk.WithStack(sdk.ErrForbidden)
}
