package hooks

import (
	"context"

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
	return nil
}
