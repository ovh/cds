package group

import (
	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// LoadOptionFunc for group.
type LoadOptionFunc func(gorp.SqlExecutor, ...*sdk.Group) error

// LoadOptions provides all options on group loads functions.
var LoadOptions = struct{}{}
