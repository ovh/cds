package hooks

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gorhill/cronexpr"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//This are all the types
const (
	TypeRepoManagerWebHook = "RepoWebHook"
	TypeWebHook            = "Webhook"
	TypeScheduler          = "Scheduler"
	TypeRepoPoller         = "RepoPoller"
	TypeKafka              = "Kafka"
	TypeRabbitMQ           = "RabbitMQ"
	TypeWorkflowHook       = "Workflow"
	TypeOutgoingWebHook    = "OutgoingWebhook"
	TypeOutgoingWorkflow   = "OutgoingWorkflow"

	GithubHeader    = "X-Github-Event"
	GitlabHeader    = "X-Gitlab-Event"
	BitbucketHeader = "X-Event-Key"

	ConfigNumber    = "Number"
	ConfigSubNumber = "SubNumber"
	ConfigHookID    = "HookID"
	ConfigHookRunID = "HookRunID"
)

var (
	rootKey           = cache.Key("hooks", "tasks")
	executionRootKey  = cache.Key("hooks", "tasks", "executions")
	schedulerQueueKey = cache.Key("hooks", "scheduler", "queue")
)

// runTasks should run as a long-running goroutine
func (s *Service) runTasks(ctx context.Context) error {
	if err := s.synchronizeTasks(); err != nil {
		log.Error("Hook> Unable to synchronize tasks: %v", err)
	}

	if err := s.startTasks(ctx); err != nil {
		log.Error("Hook> Exit running tasks: %v", err)
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

	log.Info("Hooks> Synchronizing tasks from CDS API (%s)", s.Cfg.API.HTTP.URL)

	//Get all hooks from CDS, and synchronize the tasks in cache
	hooks, err := s.Client.WorkflowAllHooksList()
	if err != nil {
		return sdk.WrapError(err, "Unable to get hooks")
	}

	allOldTasks, err := s.Dao.FindAllTasks()
	if err != nil {
		return sdk.WrapError(err, "Unable to get allOldTasks")
	}

	//Delete all old task which are not referenced in CDS API anymore
	for i := range allOldTasks {
		t := &allOldTasks[i]
		var found bool
		for _, h := range hooks {
			if h.UUID == t.UUID {
				found = true
				break
			}
		}
		if !found && t.Type != TypeOutgoingWebHook && t.Type != TypeOutgoingWorkflow {
			s.Dao.DeleteTask(t)
			log.Info("Hook> Task %s deleted on synchronization", t.UUID)
		}
	}

	for _, h := range hooks {
		confProj := h.Config[sdk.HookConfigProject]
		confWorkflow := h.Config[sdk.HookConfigWorkflow]
		if confProj.Value == "" || confWorkflow.Value == "" {
			log.Error("Hook> Unable to synchronize task %+v: %v", h, err)
			continue
		}
		t, err := s.hookToTask(&h)
		if err != nil {
			log.Error("Hook> Unable to synchronize task %+v: %v", h, err)
			continue
		}
		s.Dao.SaveTask(t)
	}

	return nil
}

func (s *Service) hookToTask(h *sdk.WorkflowNodeHook) (*sdk.Task, error) {
	if h.WorkflowHookModel.Type != sdk.WorkflowHookModelBuiltin {
		return nil, fmt.Errorf("Unsupported hook type: %s", h.WorkflowHookModel.Type)
	}

	switch h.WorkflowHookModel.Name {
	case sdk.KafkaHookModelName:
		return &sdk.Task{
			UUID:   h.UUID,
			Type:   TypeKafka,
			Config: h.Config,
		}, nil
	case sdk.RabbitMQHookModelName:
		return &sdk.Task{
			UUID:   h.UUID,
			Type:   TypeRabbitMQ,
			Config: h.Config,
		}, nil
	case sdk.WebHookModelName:
		h.Config["webHookURL"] = sdk.WorkflowNodeHookConfigValue{
			Value:        fmt.Sprintf("%s/webhook/%s", s.Cfg.URLPublic, h.UUID),
			Configurable: false,
		}
		return &sdk.Task{
			UUID:   h.UUID,
			Type:   TypeWebHook,
			Config: h.Config,
		}, nil
	case sdk.RepositoryWebHookModelName:
		h.Config["webHookURL"] = sdk.WorkflowNodeHookConfigValue{
			Value:        fmt.Sprintf("%s/webhook/%s", s.Cfg.URLPublic, h.UUID),
			Configurable: false,
		}
		return &sdk.Task{
			UUID:   h.UUID,
			Type:   TypeRepoManagerWebHook,
			Config: h.Config,
		}, nil
	case sdk.SchedulerModelName:
		return &sdk.Task{
			UUID:   h.UUID,
			Type:   TypeScheduler,
			Config: h.Config,
		}, nil
	case sdk.GitPollerModelName:
		return &sdk.Task{
			UUID:   h.UUID,
			Type:   TypeRepoPoller,
			Config: h.Config,
		}, nil
	case sdk.WorkflowModelName:
		return &sdk.Task{
			UUID: h.UUID,
			Type: TypeWorkflowHook,
		}, nil
	}

	return nil, fmt.Errorf("Unsupported hook: %s", h.WorkflowHookModel.Name)
}

func (s *Service) startTasks(ctx context.Context) error {
	c, cancel := context.WithCancel(ctx)
	defer cancel()

	//Load all the tasks
	tasks, err := s.Dao.FindAllTasks()
	if err != nil {
		return sdk.WrapError(err, "Unable to find all tasks")
	}

	//Start the tasks
	for i := range tasks {
		t := &tasks[i]
		if t.Type == TypeOutgoingWebHook || t.Type == TypeOutgoingWorkflow {
			continue
		}
		if _, err := s.startTask(c, t); err != nil {
			log.Error("Hooks> runLongRunningTasks> Unable to start task: %v", err)
			continue
		}
	}
	return nil
}

func (s *Service) stopTasks() error {
	//Load all the tasks
	tasks, err := s.Dao.FindAllTasks()
	if err != nil {
		return sdk.WrapError(err, "Unable to find all tasks")
	}

	//Start the tasks
	for i := range tasks {
		t := &tasks[i]
		if err := s.stopTask(t); err != nil {
			log.Error("Hooks> stopTasks> Unable to stop task: %v", err)
			continue
		}
	}
	return nil
}

func (s *Service) startTask(ctx context.Context, t *sdk.Task) (*sdk.TaskExecution, error) {
	t.Stopped = false
	s.Dao.SaveTask(t)

	switch t.Type {
	case TypeWebHook, TypeRepoManagerWebHook, TypeWorkflowHook:
		return nil, nil
	case TypeScheduler, TypeRepoPoller:
		return nil, s.prepareNextScheduledTaskExecution(t)
	case TypeKafka:
		return nil, s.startKafkaHook(t)
	case TypeRabbitMQ:
		return nil, s.startRabbitMQHook(t)
	case TypeOutgoingWebHook:
		return s.startOutgoingWebHookTask(t)
	case TypeOutgoingWorkflow:
		return s.startOutgoingWorkflowTask(t)
	default:
		return nil, fmt.Errorf("Unsupported task type %s", t.Type)
	}
}

func (s *Service) prepareNextScheduledTaskExecution(t *sdk.Task) error {
	if t.Stopped {
		return nil
	}

	//Load the last execution of this task
	execs, err := s.Dao.FindAllTaskExecutions(t)
	if err != nil {
		return sdk.WrapError(err, "unable to load last executions")
	}

	//The last execution has not been executed, let it go
	if len(execs) > 0 && execs[len(execs)-1].ProcessingTimestamp == 0 {
		log.Debug("Hooks> Scheduled tasks %s:%d ready. Next execution scheduled on %v", t.UUID, execs[len(execs)-1].Timestamp, time.Unix(0, execs[len(execs)-1].Timestamp))
		return nil
	}

	//Load the location for the timezone
	confTimezone := t.Config[sdk.SchedulerModelTimezone]
	loc, err := time.LoadLocation(confTimezone.Value)
	if err != nil {
		return sdk.WrapError(err, "unable to parse timezone: %v", t.Config[sdk.SchedulerModelTimezone])
	}

	var exec *sdk.TaskExecution
	var nextSchedule time.Time
	switch t.Type {
	case TypeScheduler:
		//Parse the cron expr
		confCron := t.Config[sdk.SchedulerModelCron]
		cronExpr, err := cronexpr.Parse(confCron.Value)
		if err != nil {
			return sdk.WrapError(err, "unable to parse cron expression: %v", t.Config[sdk.SchedulerModelCron])
		}

		//Compute a new date
		t0 := time.Now().In(loc)
		nextSchedule = cronExpr.Next(t0)

	case TypeRepoPoller:
		// Default value of next scheduling
		nextSchedule = time.Now().Add(time.Minute)
		if val, ok := t.Config["next_execution"]; ok {
			nextExec, errT := strconv.ParseInt(val.Value, 10, 64)
			if errT == nil {
				nextSchedule = time.Unix(nextExec, 0)
			}
		}
	}

	//Craft a new execution
	exec = &sdk.TaskExecution{
		Timestamp: nextSchedule.UnixNano(),
		Status:    TaskExecutionScheduled,
		Type:      t.Type,
		UUID:      t.UUID,
		Config:    t.Config,
		ScheduledTask: &sdk.ScheduledTaskExecution{
			DateScheduledExecution: fmt.Sprintf("%v", nextSchedule),
		},
	}

	s.Dao.SaveTaskExecution(exec)
	//We don't push in queue, we will the scheduler to run it

	log.Debug("Hooks> Scheduled tasks %v:%d ready. Next execution scheduled on %v, len:%d", t.UUID, exec.Timestamp, time.Unix(0, exec.Timestamp), len(execs))

	return nil
}

func (s *Service) stopTask(t *sdk.Task) error {
	log.Info("Hooks> Stopping task %s", t.UUID)
	t.Stopped = true
	s.Dao.SaveTask(t)

	switch t.Type {
	case TypeWebHook, TypeScheduler, TypeRepoManagerWebHook, TypeRepoPoller, TypeKafka, TypeWorkflowHook:
		log.Debug("Hooks> Tasks %s has been stopped", t.UUID)
		return nil
	default:
		return fmt.Errorf("Unsupported task type %s", t.Type)
	}
}

// doTask return a boolean that means the task should be restarted of not
func (s *Service) doTask(ctx context.Context, t *sdk.Task, e *sdk.TaskExecution) (bool, error) {
	if t.Stopped {
		return false, nil
	}

	var hs []sdk.WorkflowNodeRunHookEvent
	var h *sdk.WorkflowNodeRunHookEvent
	var err error

	var doRestart = false
	switch {
	case e.WebHook != nil && e.Type == TypeOutgoingWebHook:
		err = s.doOutgoingWebHookExecution(e)
	case e.Type == TypeOutgoingWorkflow:
		err = s.doOutgoingWorkflowExecution(e)
	case e.WebHook != nil && (e.Type == TypeWebHook || e.Type == TypeRepoManagerWebHook):
		hs, err = s.doWebHookExecution(e)
	case e.ScheduledTask != nil && e.Type == TypeScheduler:
		h, err = s.doScheduledTaskExecution(e)
		doRestart = true
	case e.ScheduledTask != nil && e.Type == TypeRepoPoller:
		//Populate next execution
		hs, err = s.doPollerTaskExecution(t, e)
	case e.Kafka != nil && e.Type == TypeKafka:
		h, err = s.doKafkaTaskExecution(e)
	case e.RabbitMQ != nil && e.Type == TypeRabbitMQ:
		h, err = s.doRabbitMQTaskExecution(e)
	default:
		err = fmt.Errorf("Unsupported task type %s", e.Type)
	}

	if err != nil {
		return doRestart, err
	}
	if h != nil {
		hs = append(hs, *h)
	}
	if hs == nil || len(hs) == 0 {
		return doRestart, nil
	}

	// Call CDS API
	confProj := t.Config[sdk.HookConfigProject]
	confWorkflow := t.Config[sdk.HookConfigWorkflow]
	var globalErr error
	for _, hEvent := range hs {
		run, err := s.Client.WorkflowRunFromHook(confProj.Value, confWorkflow.Value, hEvent)
		if err != nil {
			globalErr = err
			log.Error("Hooks> Unable to run workflow %s", err)
		} else {
			//Save the run number
			e.WorkflowRun = run.Number
			log.Debug("Hooks> workflow %s/%s#%d has been triggered", confProj.Value, confWorkflow.Value, run.Number)
		}
	}

	if globalErr != nil {
		return doRestart, sdk.WrapError(globalErr, "Unable to run workflow")
	}

	return doRestart, nil
}
