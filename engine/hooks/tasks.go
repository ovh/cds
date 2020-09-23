package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/gorhill/cronexpr"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//This are all the types
const (
	TypeRepoManagerWebHook = "RepoWebHook"
	TypeWebHook            = "Webhook"
	TypeScheduler          = "Scheduler"
	TypeRepoPoller         = "RepoPoller"
	TypeBranchDeletion     = "BranchDeletion"
	TypeKafka              = "Kafka"
	TypeGerrit             = "Gerrit"
	TypeRabbitMQ           = "RabbitMQ"
	TypeWorkflowHook       = "Workflow"
	TypeOutgoingWebHook    = "OutgoingWebhook"
	TypeOutgoingWorkflow   = "OutgoingWorkflow"

	GithubHeader         = "X-Github-Event"
	GitlabHeader         = "X-Gitlab-Event"
	BitbucketHeader      = "X-Event-Key"
	BitbucketCloudHeader = "X-Event-Key_Cloud" // Fake header, do not use to fetch header, just to return custom header

	ConfigNumber    = "Number"
	ConfigSubNumber = "SubNumber"
	ConfigHookID    = "HookID"
	ConfigHookRunID = "HookRunID"
)

var (
	rootKey           = cache.Key("hooks", "tasks")
	executionRootKey  = cache.Key("hooks", "tasks", "executions")
	schedulerQueueKey = cache.Key("hooks", "scheduler", "queue")
	gerritRepoKey     = cache.Key("hooks", "gerrit", "repo")
	gerritRepoHooks   = make(map[string]bool)
)

// runTasks should run as a long-running goroutine
func (s *Service) runTasks(ctx context.Context) error {
	if err := s.synchronizeTasks(ctx); err != nil {
		log.Error(ctx, "Hook> Unable to synchronize tasks: %v", err)
	}

	if err := s.startTasks(ctx); err != nil {
		log.Error(ctx, "Hook> Exit running tasks: %v", err)
		return err
	}
	<-ctx.Done()
	return ctx.Err()
}

func (s *Service) synchronizeTasks(ctx context.Context) error {
	t0 := time.Now()
	defer func() {
		log.Info(ctx, "Hooks> All tasks has been resynchronized (%.3fs)", time.Since(t0).Seconds())
	}()

	log.Info(ctx, "Hooks> Synchronizing tasks from CDS API (%s)", s.Cfg.API.HTTP.URL)

	//Get all hooks from CDS, and synchronize the tasks in cache
	hooks, err := s.Client.WorkflowAllHooksList()
	if err != nil {
		return sdk.WrapError(err, "Unable to get hooks")
	}

	allOldTasks, err := s.Dao.FindAllTasks(ctx)
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
				log.Debug("Hook> Synchronizing %s task %s", h.HookModelName, t.UUID)
				break
			}
		}
		if !found && t.Type != TypeOutgoingWebHook && t.Type != TypeOutgoingWorkflow {
			if err := s.deleteTask(ctx, t); err != nil {
				log.Error(ctx, "Hook> Error on task %s delete on synchronization: %v", t.UUID, err)
			} else {
				log.Info(ctx, "Hook> Task %s deleted on synchronization", t.UUID)
			}
		}
	}

	for _, h := range hooks {
		confProj := h.Config[sdk.HookConfigProject]
		confWorkflow := h.Config[sdk.HookConfigWorkflow]
		if confProj.Value == "" || confWorkflow.Value == "" {
			log.Error(ctx, "Hook> Unable to synchronize task %+v: %v", h, err)
			continue
		}
		t, err := s.hookToTask(&h)
		if err != nil {
			log.Error(ctx, "Hook> Unable to transform hook to task %+v: %v", h, err)
			continue
		}
		if err := s.Dao.SaveTask(t); err != nil {
			log.Error(ctx, "Hook> Unable to save task %+v: %v", h, err)
			continue
		}
	}

	// Start listening to gerrit event stream
	vcsConfig, err := s.Client.VCSConfiguration()
	if err != nil {
		return sdk.WrapError(err, "unable to get vcs configuration")
	}

	for k, v := range vcsConfig {
		if v.Type == "gerrit" && v.Username != "" && v.Password != "" && v.SSHPort != 0 {
			s.initGerritStreamEvent(ctx, k, vcsConfig)
		}
	}

	return nil
}

func (s *Service) initGerritStreamEvent(ctx context.Context, vcsName string, vcsConfig map[string]sdk.VCSConfiguration) {
	// Create channel to store gerrit event
	gerritEventChan := make(chan GerritEvent, 20)
	// Listen to gerrit event stream
	s.GoRoutines.Run(ctx, "gerrit.EventStream."+vcsName, func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				if ctx.Err() != nil {
					log.Error(ctx, "hook:initGerritStreamEvent: %v", ctx.Err())
				}
				return
			default:
				if err := ListenGerritStreamEvent(ctx, s.Cache, s.GoRoutines, vcsConfig[vcsName], gerritEventChan); err != nil {
					log.Error(ctx, "hook:initGerritStreamEvent: failed listening gerrit event stream: %v", err)
				}
				time.Sleep(10 * time.Second)
			}
		}

	})
	// Listen to gerrit event stream
	s.GoRoutines.Run(ctx, "gerrit.EventStreamCompute."+vcsName, func(ctx context.Context) {
		s.ComputeGerritStreamEvent(ctx, vcsName, gerritEventChan)
	})
	// Save the fact that we are listen the event stream for this gerrit
	gerritRepoHooks[vcsName] = true
}

func (s *Service) hookToTask(h *sdk.NodeHook) (*sdk.Task, error) {
	switch h.HookModelName {
	case sdk.GerritHookModelName:
		return &sdk.Task{
			UUID:   h.UUID,
			Type:   TypeGerrit,
			Config: h.Config,
		}, nil
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
	return nil, fmt.Errorf("Unsupported hook: %s", h.HookModelName)
}

func (s *Service) startTasks(ctx context.Context) error {
	c, cancel := context.WithCancel(ctx)
	defer cancel()

	//Load all the tasks
	tasks, err := s.Dao.FindAllTasks(ctx)
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
			log.Error(ctx, "Hooks> runLongRunningTasks> Unable to start task: %v", err)
			continue
		}
	}
	return nil
}

func (s *Service) stopTasks(ctx context.Context) error {
	//Load all the tasks
	tasks, err := s.Dao.FindAllTasks(ctx)
	if err != nil {
		return sdk.WrapError(err, "Unable to find all tasks")
	}

	//Start the tasks
	for i := range tasks {
		t := &tasks[i]
		if err := s.stopTask(ctx, t); err != nil {
			log.Error(ctx, "Hooks> stopTasks> Unable to stop task: %v", err)
			continue
		}
	}
	return nil
}

func (s *Service) startTask(ctx context.Context, t *sdk.Task) (*sdk.TaskExecution, error) {
	t.Stopped = false
	if err := s.Dao.SaveTask(t); err != nil {
		return nil, sdk.WrapError(err, "unable to save task")
	}

	switch t.Type {
	case TypeWebHook, TypeRepoManagerWebHook, TypeWorkflowHook:
		return nil, nil
	case TypeScheduler, TypeRepoPoller, TypeBranchDeletion:
		return nil, s.prepareNextScheduledTaskExecution(ctx, t)
	case TypeKafka:
		return nil, s.startKafkaHook(ctx, t)
	case TypeRabbitMQ:
		return nil, s.startRabbitMQHook(ctx, t)
	case TypeOutgoingWebHook:
		return s.startOutgoingWebHookTask(t)
	case TypeOutgoingWorkflow:
		return s.startOutgoingWorkflowTask(t)
	case TypeGerrit:
		return nil, s.startGerritHookTask(t)
	default:
		return nil, fmt.Errorf("Unsupported task type %s", t.Type)
	}
}

func (s *Service) prepareNextScheduledTaskExecution(ctx context.Context, t *sdk.Task) error {
	if t.Stopped {
		return nil
	}

	//Load the last execution of this task
	execs, err := s.Dao.FindAllTaskExecutions(ctx, t)
	if err != nil {
		return sdk.WrapError(err, "unable to load last executions")
	}

	//The last execution has not been executed, let it go
	if len(execs) > 0 && execs[len(execs)-1].ProcessingTimestamp == 0 {
		log.Debug("Hooks> Scheduled task %s:%d ready. Next execution already scheduled on %v", t.UUID, execs[len(execs)-1].Timestamp, time.Unix(0, execs[len(execs)-1].Timestamp))
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

	case TypeBranchDeletion:
		now := time.Now()
		nextSchedule = now.Add(24 * time.Hour)
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

	log.Debug("Hooks> Scheduled task %v:%d ready. Next execution scheduled on %v, len:%d", t.UUID, exec.Timestamp, time.Unix(0, exec.Timestamp), len(execs))

	return nil
}

func (s *Service) stopTask(ctx context.Context, t *sdk.Task) error {
	log.Info(ctx, "Hooks> Stopping task %s", t.UUID)
	t.Stopped = true
	if err := s.Dao.SaveTask(t); err != nil {
		return sdk.WrapError(err, "unable to save task %v", t)
	}

	switch t.Type {
	case TypeWebHook, TypeScheduler, TypeRepoManagerWebHook, TypeRepoPoller, TypeKafka, TypeWorkflowHook:
		log.Debug("Hooks> Tasks %s has been stopped", t.UUID)
		return nil
	case TypeGerrit:
		s.stopGerritHookTask(t)
		log.Debug("Hooks> Gerrit Task %s has been stopped", t.UUID)
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
	case e.GerritEvent != nil:
		h, err = s.doGerritExecution(e)
	case e.WebHook != nil && e.Type == TypeOutgoingWebHook:
		err = s.doOutgoingWebHookExecution(ctx, e)
	case e.Type == TypeOutgoingWorkflow:
		err = s.doOutgoingWorkflowExecution(ctx, e)
	case e.WebHook != nil && (e.Type == TypeWebHook || e.Type == TypeRepoManagerWebHook):
		hs, err = s.doWebHookExecution(ctx, e)
	case e.ScheduledTask != nil && e.Type == TypeScheduler:
		h, err = s.doScheduledTaskExecution(ctx, e)
		doRestart = true
	case e.ScheduledTask != nil && e.Type == TypeRepoPoller:
		//Populate next execution
		hs, err = s.doPollerTaskExecution(ctx, t, e)
		doRestart = true
	case e.ScheduledTask != nil && e.Type == TypeBranchDeletion:
		_, err = s.doBranchDeletionTaskExecution(e)
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
			log.Warning(ctx, "Hooks> %s > unable to run workflow %s/%s : %v", t.UUID, confProj.Value, confWorkflow.Value, err)
		} else {
			//Save the run number
			e.WorkflowRun = run.Number
			log.Debug("Hooks> workflow %s/%s#%d has been triggered", confProj.Value, confWorkflow.Value, run.Number)
		}
	}

	if globalErr != nil {
		return doRestart, globalErr
	}

	return doRestart, nil
}

func getPayloadStringVariable(ctx context.Context, payload map[string]interface{}, msg interface{}) {
	payloadStr, err := json.Marshal(msg)
	if err != nil {
		log.Error(ctx, "Unable to marshal payload: %v", err)
	}
	payload[PAYLOAD] = string(payloadStr)
}
