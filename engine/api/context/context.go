package context

import (
	"github.com/ovh/cds/sdk"
)

// Context gather information about http call origin
type Context struct {
	Agent  sdk.Agent
	User   *sdk.User
	Worker sdk.Worker
}
