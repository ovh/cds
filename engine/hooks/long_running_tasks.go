package hooks

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ovh/cds/sdk"

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

func (s *Service) doWebHook(t *LongRunningTask, r *http.Request) error {
	log.Info("Hooks> Executing tasks %s", t.UUID)

	if t.Config["method"] != r.Method {
		log.Debug("Hooks> config %+v doesn't match", t.Config)
		return sdk.ErrWebhookConfigDoesNotMatch
	}

	return nil
}
