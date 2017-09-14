package hooks

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/sdk/log"
)

// runTasks should run as a long-running goroutine
func (s *Service) runTasks(ctx context.Context) error {
	if err := s.synchronizeTasks(); err != nil {
		log.Error("Hook> Unable to synchronize tasks: %v", err)
	}

	if err := s.startLongRunningTasks(ctx); err != nil {
		log.Error("Hook> EXit long running tasks: %v", err)
	}

	if err := s.startScheduledTasks(ctx); err != nil {
		log.Error("Hook> Exit scheduled tasks: %v", err)
	}
	<-ctx.Done()
	return ctx.Err()
}

func (s *Service) synchronizeTasks() error {
	log.Info("Hooks> Tasks synchronized")

	//Get all hooks from CDS, and synchronize the tasks in cache
	hooks, err := s.cds.WorkflowAllHooksList()
	if err != nil {
		return sdk.WrapError(err, "synchronizeTasks> Unable to get hooks")
	}

	for _, h := range hooks {
		t, err := s.hookToTask(h)
		if err != nil {
			log.Error("Hook> Unable to synchronize task +%v: %v", h, err)
			continue
		}
		_ = t
	}

	return nil
}

func (s *Service) hookToTask(h sdk.WorkflowNodeHook) (interface{}, error) {
	if h.WorkflowHookModel.Type != sdk.WorkflowHookModelBuiltin {
		return nil, fmt.Errorf("Unsupported hook type: %s", h.WorkflowHookModel.Type)
	}

	return nil, nil
}
