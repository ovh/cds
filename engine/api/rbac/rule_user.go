package rbac

import (
	"context"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

func IsCurrentUser(_ context.Context, auth *sdk.AuthConsumer, _ cache.Store, _ gorp.SqlExecutor, vars map[string]string) error {
	if vars["user"] == auth.AuthentifiedUser.Username {
		return nil
	}
	return sdk.WithStack(sdk.ErrForbidden)
}
