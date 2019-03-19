package authentication

import (
	"context"
	"net/http"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

type Driver interface {
	CheckAuthentication(ctx context.Context, db gorp.SqlExecutor, r *http.Request) (*sdk.AuthentifiedUser, error)
}
