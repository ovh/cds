package authentication

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// LoadSessionOptionFunc for auth session.
type LoadSessionOptionFunc func(context.Context, gorp.SqlExecutor, ...*sdk.AuthSession) error
