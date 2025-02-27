package api

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (api *API) isCurrentUser(ctx context.Context, vars map[string]string) error {
	c := getUserConsumer(ctx)
	if c == nil {
		return sdk.WithStack(sdk.ErrForbidden)
	}
	if vars["user"] == c.AuthConsumerUser.AuthentifiedUser.Username {
		return nil
	}
	return sdk.WithStack(sdk.ErrForbidden)
}
