package hooks

import (
	"context"
	"fmt"
	"time"

	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/tat/api/cache"

	"github.com/ovh/cds/sdk/log"
)

//This are all the types
const (
	TypeWebHook = "Webhook"
)

var (
	longRunningRootKey = cache.Key("hooks", "tasks", "long_running")
)

// runTasks should run as a long-running goroutine
func (s *Service) runTasks(ctx context.Context) error {
	if err := s.synchronizeTasks(); err != nil {
		log.Error("Hook> Unable to synchronize tasks: %v", err)
	}

	if err := s.startLongRunningTasks(ctx); err != nil {
		log.Error("Hook> Exit long running tasks: %v", err)
		return err
	}

	if err := s.startScheduledTasks(ctx); err != nil {
		log.Error("Hook> Exit scheduled tasks: %v", err)
		return err
	}
	<-ctx.Done()
	return ctx.Err()
}

func (s *Service) synchronizeTasks() error {
	t0 := time.Now()
	defer func() {
		log.Info("Hooks> All tasks has been resynchronized (%.3fs)", time.Since(t0).Seconds())
	}()

	//Get all hooks from CDS, and synchronize the tasks in cache
	hooks, err := s.cds.WorkflowAllHooksList()
	if err != nil {
		return sdk.WrapError(err, "synchronizeTasks> Unable to get hooks")
	}

	log.Info("Hooks> Synchronizing (%d) tasks from CDS API (%s)", len(hooks), s.Cfg.API.HTTP.URL)

	for _, h := range hooks {
		t, err := s.hookToTask(h)
		if err != nil {
			log.Error("Hook> Unable to synchronize task +%v: %v", h, err)
			continue
		}
		if lrTask, ok := t.(LongRunningTask); ok {
			s.Dao.SaveLongRunningTask(&lrTask)
			continue
		}
		//TODO save scheduled tasks
	}

	return nil
}

func (s *Service) hookToTask(h sdk.WorkflowNodeHook) (interface{}, error) {
	if h.WorkflowHookModel.Type != sdk.WorkflowHookModelBuiltin {
		return nil, fmt.Errorf("Unsupported hook type: %s", h.WorkflowHookModel.Type)
	}

	switch h.WorkflowHookModel.Name {
	case workflow.WebHookModel.Name:
		return LongRunningTask{
			UUID:   h.UUID,
			Type:   TypeWebHook,
			Config: h.Config,
		}, nil
	}

	return nil, fmt.Errorf("Unsupported hook: %s", h.WorkflowHookModel.Name)
}
