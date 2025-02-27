package api

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (api *API) isHookService(ctx context.Context, _ map[string]string) error {
	auth := getUserConsumer(ctx)
	if auth == nil {
		return sdk.WithStack(sdk.ErrForbidden)
	}
	if auth.AuthConsumerUser.Service != nil && auth.AuthConsumerUser.Service.Type == sdk.TypeHooks {
		return nil
	}
	return sdk.WithStack(sdk.ErrForbidden)
}
