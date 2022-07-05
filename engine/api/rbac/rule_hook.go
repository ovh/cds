package rbac

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func IsHookService(_ context.Context, auth *sdk.AuthConsumer, _ cache.Store, _ gorp.SqlExecutor, _ map[string]string) error {
	if auth.Service != nil && auth.Service.Type == sdk.TypeHooks {
		return nil
	}
	return sdk.WithStack(sdk.ErrForbidden)
}
