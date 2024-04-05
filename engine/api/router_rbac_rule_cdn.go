package api

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func (api *API) isCDNService(_ context.Context, auth *sdk.AuthUserConsumer, _ cache.Store, _ gorp.SqlExecutor, _ map[string]string) error {
	if auth == nil {
		return sdk.WithStack(sdk.ErrForbidden)
	}
	if auth.AuthConsumerUser.Service != nil && auth.AuthConsumerUser.Service.Type == sdk.TypeCDN {
		return nil
	}
	return sdk.WithStack(sdk.ErrForbidden)
}
