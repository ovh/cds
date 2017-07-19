package businesscontext

import (
	"github.com/ovh/cds/sdk"
)

// Ctx gather information about http call origin
type Ctx struct {
	Agent       string
	User        *sdk.User
	Worker      *sdk.Worker
	Hatchery    *sdk.Hatchery
	Project     *sdk.Project
	Application *sdk.Application
	Pipeline    *sdk.Pipeline
}
