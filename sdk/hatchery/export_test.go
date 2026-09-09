package hatchery

import (
	"context"

	"github.com/ovh/cds/sdk"
)

// CanRunJobWithModelV2ForTest exposes canRunJobWithModelV2 to the external test
// package, which cannot build the unexported request itself. The mocks import this
// package, so the tests using them cannot live in it.
func CanRunJobWithModelV2ForTest(ctx context.Context, h InterfaceWithModels, requirements []sdk.Requirement, workerModelV2 string) (*sdk.Model, *sdk.WorkerStarterWorkerModel, error) {
	return canRunJobWithModelV2(ctx, h, workerStarterRequest{
		id:           "job-id",
		requirements: requirements,
	}, workerModelV2)
}
