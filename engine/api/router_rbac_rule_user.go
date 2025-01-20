package api

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func (api *API) isCurrentUser(_ context.Context, auth *sdk.AuthUserConsumer, _ cache.Store, _ gorp.SqlExecutor, vars map[string]string) error {
	if auth == nil {
		return sdk.WithStack(sdk.ErrForbidden)
	}
	if vars["user"] == auth.AuthConsumerUser.AuthentifiedUser.Username {
		return nil
	}
	return sdk.WithStack(sdk.ErrForbidden)
}
