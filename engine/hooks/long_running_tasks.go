package hooks

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk/log"
)

func (s *Service) startLongRunningTasks(ctx context.Context) error {
	log.Info("Hooks> Starting long running tasks...")
	c, cancel := context.WithCancel(ctx)
	defer cancel()

	//Load all the tasks
	tasks, err := s.Dao.FindAllLongRunningTasks()
	if err != nil {
		return err
	}

	log.Debug("Hooks> Starting %d tasks", len(tasks))

	//Start the tasks
	for i := range tasks {
		t := &tasks[i]
		if err := s.startLongRunningTask(c, t); err != nil {
			log.Error("hooks.runLongRunningTasks> Unable to start tasks: %v", err)
			return err
		}
	}
	return nil
}

func (s *Service) startLongRunningTask(ctx context.Context, t *LongRunningTask) error {
	log.Info("Hooks> Starting long running task %s", t.UUID)
	switch t.Type {
	case TypeWebHook:
		log.Debug("Hooks> Webhook tasks %s ready", t.UUID)
		return nil
	default:
		return fmt.Errorf("Unsupported task type %s", t.Type)
	}
}
