package hooks

import (
	"context"
	"github.com/ovh/cds/sdk/telemetry"

	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

func (s *Service) triggerWorkflowHooks(ctx context.Context, hre *sdk.HookRepositoryEvent) error {
	ctx, next := telemetry.Span(ctx, "s.triggerWorkflowHooks")
	defer next()

	log.Info(ctx, "triggering workflow hooks for event [%s] %s", hre.EventName, hre.GetFullName())
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
