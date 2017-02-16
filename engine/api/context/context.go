package context

import (
	"github.com/ovh/cds/sdk"
)

// Ctx gather information about http call origin
type Ctx struct {
	Agent    sdk.Agent
	User     *sdk.User
	Worker   *sdk.Worker
	Hatchery *sdk.Hatchery
}
