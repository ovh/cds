package context

import (
	"github.com/ovh/cds/sdk"
)

// Context gather information about http call origin
type Context struct {
	User     *sdk.User
	WorkerID string
}
