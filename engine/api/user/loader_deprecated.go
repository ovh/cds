package user

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

type LoadDeprecatedUserOptionFunc func(gorp.SqlExecutor, ...*sdk.User) error
