package hooks

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (s *Service) triggerGetGitInfo(ctx context.Context, hre *sdk.HookRepositoryEvent) error {

	// Continue to next step
	hre.Status = sdk.HookEventStatusWorkflow
	if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
		return err
	}
	return s.triggerWorkflows(ctx, hre)
}
