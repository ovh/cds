package hooks

import (
	"context"

	"github.com/ovh/cds/sdk"
)

func (s *Service) triggerWorkflowHooks(ctx context.Context, hre *sdk.HookRepositoryEvent) error {
	// TODO trigger workflow hook
	hre.Status = sdk.HookEventStatusDone
	if err := s.Dao.SaveRepositoryEvent(ctx, hre); err != nil {
		return err
	}
	if err := s.Dao.RemoveRepositoryEventFromInProgressList(ctx, *hre); err != nil {
		return err
	}
	return nil
}
