package hooks

import (
	"context"
	"fmt"

	"github.com/ovh/cds/sdk/log"
)

//This are all the types
const (
	TypeWebHook = "webhook"

	longRunningTasksKey = "cds:hooks:longRunningTasks"
)

func (s *Service) startLongRunningTasks(ctx context.Context) error {
	log.Info("Hooks> Starting long running tasks...")
	c, cancel := context.WithCancel(ctx)
	defer cancel()

	//Load all the tasks
	nbTasks := s.Cache.SetCard(longRunningTasksKey)
	tasks := make([]*LongRunningTask, nbTasks)
	for _, t := range tasks {
		*t = LongRunningTask{}
	}
	if err := s.Cache.SetScan(longRunningTasksKey, interfaceSlice(tasks)...); err != nil {
		return err
	}

	//Start the tasks
	for _, t := range tasks {
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
