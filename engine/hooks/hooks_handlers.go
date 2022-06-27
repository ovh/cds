package hooks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (s *Service) registerRepositoryHookHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

		var newHook sdk.RepositoryWebHook
		//Read the body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "unable to read request")
		}
		if err := json.Unmarshal(body, &newHook); err != nil {
			return sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to unmarshal request")
		}

		if len(newHook.Configuration) == 0 {
			return sdk.NewErrorFrom(sdk.ErrInvalidData, "missing hook configuration")
		}

		vcsType, has := newHook.Configuration[sdk.HookConfigVCSType]
		if !has || vcsType.Value == "" {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing vcs type")
		}

		vcsName, has := newHook.Configuration[sdk.HookConfigVCSServer]
		if !has || vcsName.Value == "" {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing vcs name")
		}

		repoName, has := newHook.Configuration[sdk.HookConfigRepoFullName]
		if !has || repoName.Value == "" {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing repository name")
		}

		project, has := newHook.Configuration[sdk.HookConfigProject]
		if !has || project.Value == "" {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "missing project key")
		}

		if err := s.addTaskFromRepositoryHook(newHook); err != nil {
			return sdk.WithStack(err)
		}
		return nil
	}
}

func (s *Service) repositoryHooksHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// Get repository data
		vcsName := r.Header.Get(sdk.SignHeaderVCSName)
		repoName := r.Header.Get(sdk.SignHeaderRepoName)
		vcsType := r.Header.Get(sdk.SignHeaderVCSType)

		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to read body: %v", err)
		}

		// Search for existing hooks
		hookKey := strings.ToLower(cache.Key(EntitiesHookRootKey, vcsType, vcsName, repoName, "*"))
		keys, err := s.Dao.GetAllEntitiesHookKeysByPattern(hookKey)
		if err != nil {
			log.Error(ctx, "unable to check if a hook exist for %s: %v", hookKey, err)
			return err
		}
		if len(keys) == 0 {
			log.Warn(ctx, "Receive hook from %s, but there is no tasks", hookKey)
		}
		for _, k := range keys {
			var uuid string
			if _, err := s.Dao.store.Get(k, &uuid); err != nil {
				log.Error(ctx, "unable to retrieve hook uuid for %s: %v", k, err)
				continue
			}
			hook := s.Dao.FindTask(ctx, uuid)
			if hook == nil {
				return sdk.WrapError(sdk.ErrNotFound, "no hook found on")
			}

			// Enqueue execution
			exec := &sdk.TaskExecution{
				Timestamp:     time.Now().UnixNano(),
				Type:          hook.Type,
				UUID:          hook.UUID,
				Configuration: hook.Configuration,
				Status:        TaskExecutionScheduled,
				WebHook: &sdk.WebHookExecution{
					RequestBody:   body,
					RequestHeader: r.Header,
					RequestURL:    r.URL.RawQuery,
				},
			}
			log.Info(ctx, "Save Entities hook execution for task %v", hook.Configuration)
			if err := s.Dao.SaveTaskExecution(exec); err != nil {
				return err
			}
		}
		return service.WriteJSON(w, nil, http.StatusAccepted)
	}
}

func (s *Service) repositoryWebHookHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		uuid := vars["uuid"]

		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return sdk.NewErrorFrom(sdk.ErrUnknownError, "unable to read body: %v", err)
		}

		hook := s.Dao.FindTask(ctx, uuid)
		if hook == nil {
			return sdk.WrapError(sdk.ErrNotFound, "no hook found on")
		}

		// Enqueue execution
		exec := &sdk.TaskExecution{
			Timestamp:     time.Now().UnixNano(),
			Type:          hook.Type,
			UUID:          hook.UUID,
			Configuration: hook.Configuration,
			Status:        TaskExecutionScheduled,
			WebHook: &sdk.WebHookExecution{
				RequestBody:   body,
				RequestHeader: r.Header,
				RequestURL:    r.URL.RawQuery,
			},
		}
		log.Debug(ctx, "Save execution for task %v", hook.Configuration)
		if err := s.Dao.SaveTaskExecution(exec); err != nil {
			return err
		}

		return service.WriteJSON(w, exec, http.StatusAccepted)
	}
}

func (s *Service) webhookHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the webhook
		vars := mux.Vars(r)
		uuid := vars["uuid"]

		if uuid == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "Invalid uuid or name")
		}

		//Load the task
		webHook := s.Dao.FindTask(ctx, uuid)
		if webHook == nil {
			return sdk.WrapError(sdk.ErrNotFound, "Unknown uuid")
		}

		//Check method
		confValue := webHook.Config[sdk.WebHookModelConfigMethod]
		if r.Method != confValue.Value {
			return sdk.WrapError(sdk.ErrMethodNotAllowed, "Unsupported method %s : %v", r.Method, webHook.Config)
		}

		//Read the body
		req, err := io.ReadAll(r.Body)
		if err != nil {
			return sdk.WrapError(err, "Unable to read request")
		}

		//Prepare a web hook execution
		exec := &sdk.TaskExecution{
			Timestamp: time.Now().UnixNano(),
			Type:      webHook.Type,
			UUID:      webHook.UUID,
			Config:    webHook.Config,
			Status:    TaskExecutionScheduled,
			WebHook: &sdk.WebHookExecution{
				RequestBody:   req,
				RequestHeader: r.Header,
				RequestURL:    r.URL.RawQuery,
			},
		}

		//Save the web hook execution
		s.Dao.SaveTaskExecution(exec)

		//Return the execution
		return service.WriteJSON(w, exec, http.StatusOK)
	}
}

func (s *Service) startTasksHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := s.startTasks(ctx); err != nil {
			return sdk.WithStack(err)
		}
		return nil
	}
}

func (s *Service) stopTasksHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if err := s.stopTasks(ctx); err != nil {
			return sdk.WithStack(err)
		}
		return nil
	}
}

func (s *Service) startTaskHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		vars := mux.Vars(r)
		uuid := vars["uuid"]
		if uuid == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "Invalid uuid")
		}

		//Load the task
		t := s.Dao.FindTask(ctx, uuid)
		if t == nil {
			return sdk.WrapError(sdk.ErrNotFound, "Unknown uuid")
		}

		//Start the task
		if _, err := s.startTask(ctx, t); err != nil {
			return sdk.WrapError(err, "Start task")
		}
		return nil
	}
}

func (s *Service) stopTaskHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		vars := mux.Vars(r)
		uuid := vars["uuid"]
		if uuid == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "Invalid uuid")
		}

		//Load the task
		t := s.Dao.FindTask(ctx, uuid)
		if t == nil {
			return sdk.WrapError(sdk.ErrNotFound, "Unknown uuid")
		}

		//Stop the task
		if err := s.stopTask(ctx, t); err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "Stop task")
		}
		return nil
	}
}

func (s *Service) postTaskHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//This handler read a sdk.WorkflowNodeHook from the body
		var hook sdk.NodeHook
		if err := service.UnmarshalBody(r, &hook); err != nil {
			return sdk.WithStack(err)
		}
		if err := s.addTask(ctx, &hook); err != nil {
			return sdk.WithStack(err)
		}
		return nil
	}
}

func (s *Service) postAndExecuteTaskHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//This handler read a sdk.WorkflowNodeOutgoingHook from the body
		var nr sdk.WorkflowNodeRun
		if err := service.UnmarshalBody(r, &nr); err != nil {
			return sdk.WrapError(err, "Hooks> postAndExecuteTaskHandler")
		}
		t, e, err := s.addAndExecuteTask(ctx, nr)
		if err != nil {
			return sdk.WrapError(err, "Hooks> postAndExecuteTaskHandler> unable to add Task")
		}
		t.Executions = []sdk.TaskExecution{e}
		return service.WriteJSON(w, t, http.StatusOK)
	}
}

const (
	sortKeyNbExecutionsTotal = "nb_executions_total"
	sortKeyNbExecutionsTodo  = "nb_executions_todo"
)

func (s *Service) getTasksHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		sortParams, err := api.QuerySort(r)
		if err != nil {
			return sdk.NewError(sdk.ErrWrongRequest, err)
		}
		for k := range sortParams {
			if k != sortKeyNbExecutionsTotal && k != sortKeyNbExecutionsTodo {
				return sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("invalid given sort key"))
			}
		}

		tasks, err := s.Dao.FindAllTasks(ctx)
		if err != nil {
			return sdk.WithStack(err)
		}

		execs, err := s.Dao.FindAllTaskExecutionsForTasks(ctx, tasks...)
		if err != nil {
			return sdk.WithStack(err)
		}

		m := make(map[string][]sdk.TaskExecution, len(tasks))
		for _, e := range execs {
			m[e.UUID] = append(m[e.UUID], e)
		}

		for i, t := range tasks {
			var nbTodo int
			for _, e := range m[t.UUID] {
				if e.ProcessingTimestamp == 0 {
					nbTodo++
				}
			}
			tasks[i].NbExecutionsTotal = len(m[t.UUID])
			tasks[i].NbExecutionsTodo = nbTodo
		}

		for k, p := range sortParams {
			switch k {
			case sortKeyNbExecutionsTotal:
				sort.Slice(tasks, func(i, j int) bool {
					return api.SortCompareInt(tasks[i].NbExecutionsTotal, tasks[j].NbExecutionsTotal, p)
				})
			case sortKeyNbExecutionsTodo:
				sort.Slice(tasks, func(i, j int) bool {
					return api.SortCompareInt(tasks[i].NbExecutionsTodo, tasks[j].NbExecutionsTodo, p)
				})
			}
		}

		return service.WriteJSON(w, tasks, http.StatusOK)
	}
}

func (s *Service) putTaskHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		vars := mux.Vars(r)
		uuid := vars["uuid"]
		if uuid == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "Hooks> putTaskHandler> invalid uuid")
		}

		//Load the task
		t := s.Dao.FindTask(ctx, uuid)
		if t == nil {
			return sdk.WrapError(sdk.ErrNotFound, "Hooks> putTaskHandler> unknown uuid")
		}

		//Stop the task
		if err := s.stopTask(ctx, t); err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "Hooks> putTaskHandler> stop task")
		}

		//Save it
		if err := s.Dao.SaveTask(t); err != nil {
			return sdk.WrapError(err, "Unable to save task %v", t)
		}

		//Start the task
		if _, err := s.startTask(ctx, t); err != nil {
			return sdk.WrapError(err, "Unable start task %v", t)
		}

		return nil
	}
}

func (s *Service) getTaskHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		vars := mux.Vars(r)
		uuid := vars["uuid"]

		//Load the task
		t := s.Dao.FindTask(ctx, uuid)
		if t == nil {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		execs, err := s.Dao.FindAllTaskExecutions(ctx, t)
		if err != nil {
			return sdk.WrapError(err, "Unable to load executions")
		}

		t.Executions = execs

		return service.WriteJSON(w, t, http.StatusOK)
	}
}

func (s *Service) deleteTaskHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		vars := mux.Vars(r)
		uuid := vars["uuid"]

		//Load the task
		t := s.Dao.FindTask(ctx, uuid)
		if t == nil {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		//Stop the task
		if err := s.stopTask(ctx, t); err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "Hooks> putTaskHandler> stop task")
		}

		return s.deleteTask(ctx, t)
	}
}

func (s *Service) getTaskExecutionsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		vars := mux.Vars(r)
		uuid := vars["uuid"]

		//Load the task
		t := s.Dao.FindTask(ctx, uuid)
		if t == nil {
			return sdk.WithStack(sdk.ErrNotFound)
		}

		//Load the executions
		execs, err := s.Dao.FindAllTaskExecutions(ctx, t)
		if err != nil {
			return sdk.WrapError(err, "Unable to find task executions for %s", uuid)
		}
		t.Executions = execs

		sort.Slice(t.Executions, func(i, j int) bool {
			return t.Executions[i].Timestamp > t.Executions[j].Timestamp
		})

		return service.WriteJSON(w, t, http.StatusOK)
	}
}

func (s *Service) deleteAllTaskExecutionsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		vars := mux.Vars(r)
		uuid := vars["uuid"]

		//Load the task
		t := s.Dao.FindTask(ctx, uuid)
		if t == nil {
			return service.WriteJSON(w, t, http.StatusOK)
		}

		//Stop the task
		if err := s.stopTask(ctx, t); err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "Hooks> deleteAllTaskExecutionsHandler> stop task")
		}

		//Load the executions
		execs, err := s.Dao.FindAllTaskExecutions(ctx, t)
		if err != nil {
			return sdk.WrapError(err, "Unable to find task executions for %s", uuid)
		}
		for i := range execs {
			if err := s.Dao.DeleteTaskExecution(&execs[i]); err != nil {
				return err
			}
		}

		//Start the task
		if _, err := s.startTask(ctx, t); err != nil {
			return sdk.WrapError(err, "Unable start task %+v", t)
		}

		return service.WriteJSON(w, t, http.StatusOK)
	}
}

func (s *Service) deleteTaskBulkHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		hooks := map[string]sdk.NodeHook{}
		if err := service.UnmarshalBody(r, &hooks); err != nil {
			return sdk.WithStack(err)
		}

		for uuid := range hooks {
			//Load the task
			t := s.Dao.FindTask(ctx, uuid)
			if t == nil {
				continue
			}

			//Stop the task
			if err := s.stopTask(ctx, t); err != nil {
				return sdk.WrapError(sdk.ErrNotFound, "Stop task %s", err)
			}

			if err := s.deleteTask(ctx, t); err != nil {
				return err
			}

		}

		return nil
	}
}

func (s *Service) postTaskBulkHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//This handler read a sdk.WorkflowNodeHook from the body
		hooks := map[string]sdk.NodeHook{}
		if err := service.UnmarshalBody(r, &hooks); err != nil {
			return sdk.WithStack(err)
		}

		for _, hook := range hooks {
			err := s.updateTask(ctx, &hook)
			if err == errNoTask {
				if err := s.addTask(ctx, &hook); err != nil {
					return sdk.WithStack(err)
				}
			} else if err != nil {
				return sdk.WithStack(err)
			}
		}
		return service.WriteJSON(w, hooks, http.StatusOK)
	}
}

func (s *Service) addTask(ctx context.Context, h *sdk.NodeHook) error {
	//Parse the hook as a task
	t, err := s.hookToTask(h)
	if err != nil {
		return sdk.WrapError(err, "Unable to parse hook")
	}

	//Save the task
	if err := s.Dao.SaveTask(t); err != nil {
		return sdk.WrapError(err, "unable to addTask %v", t)
	}

	//Start the task
	if _, err := s.startTask(ctx, t); err != nil {
		return sdk.WrapError(err, "Unable start task %v", t)
	}
	return nil
}

func (s *Service) addTaskFromRepositoryHook(h sdk.RepositoryWebHook) error {
	t, err := s.repositoryHookToTask(h)
	if err != nil {
		return err
	}

	if err := s.Dao.SaveRepoWebHook(t); err != nil {
		return err
	}
	return nil
}

func (s *Service) addAndExecuteTask(ctx context.Context, nr sdk.WorkflowNodeRun) (sdk.Task, sdk.TaskExecution, error) {
	// Parse the hook as a task
	t, err := s.nodeRunToTask(nr)
	if err != nil {
		return t, sdk.TaskExecution{}, sdk.WrapError(err, "Hooks> addAndExecuteTask> Unable to parse node run (%+v)", nr)
	}
	// Save the task
	if err := s.Dao.SaveTask(&t); err != nil {
		return t, sdk.TaskExecution{}, sdk.WrapError(err, "unable to save task %v", t)
	}

	// Start the task
	e, err := s.startTask(ctx, &t)
	if err != nil {
		return t, sdk.TaskExecution{}, sdk.WrapError(err, "Unable start task %+v", t)
	}

	return t, *e, nil
}

var errNoTask = errors.New("task not found")

func (s *Service) updateTask(ctx context.Context, h *sdk.NodeHook) error {
	//Parse the hook as a task
	t, err := s.hookToTask(h)
	if err != nil {
		return sdk.WrapError(err, "Unable to parse hook")
	}

	task := s.Dao.FindTask(ctx, t.UUID)
	if task == nil {
		return errNoTask
	}

	task.Config = t.Config
	_ = s.stopTask(ctx, t)
	execs, _ := s.Dao.FindAllTaskExecutions(ctx, t)
	for _, e := range execs {
		if e.Status == TaskExecutionScheduled {
			if err := s.Dao.DeleteTaskExecution(&e); err != nil {
				log.Error(ctx, "unable to delete previous task execution: %v", err)
			}
		}
	}
	if _, err := s.startTask(ctx, t); err != nil {
		return sdk.WrapError(err, "Unable start task %+v", t)
	}
	// Save the task
	if err := s.Dao.SaveTask(t); err != nil {
		return sdk.WrapError(err, "unable to save task %v", t)
	}
	return nil
}

func (s *Service) deleteTask(ctx context.Context, t *sdk.Task) error {
	switch t.Type {
	case TypeGerrit:
		s.stopGerritHookTask(t)
	case TypeEntitiesHook:
		entitiesHookKey := cache.Key(EntitiesHookRootKey,
			t.Configuration[sdk.HookConfigVCSType].Value,
			t.Configuration[sdk.HookConfigVCSServer].Value,
			t.Configuration[sdk.HookConfigRepoFullName].Value,
			t.Configuration[sdk.HookConfigTypeProject].Value)
		if err := s.Dao.store.Delete(entitiesHookKey); err != nil {
			return err
		}
	}

	//Delete the task
	return s.Dao.DeleteTask(ctx, t)
}

// Status returns sdk.MonitoringStatus, implements interface service.Service
func (s *Service) Status(ctx context.Context) *sdk.MonitoringStatus {
	m := s.NewMonitoringStatus()

	if s.Dao.store == nil {
		return m
	}

	// hook queue in status
	status := sdk.MonitoringStatusOK
	size, errQ := s.Dao.QueueLen()
	if errQ != nil {
		log.Error(ctx, "Status> Unable to retrieve queue len: %v", errQ)
	}

	if size >= 100 {
		status = sdk.MonitoringStatusAlert
	} else if size >= 10 {
		status = sdk.MonitoringStatusWarn
	}
	m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Queue", Value: fmt.Sprintf("%d", size), Status: status})

	// hook balance in status
	in, out := s.Dao.TaskExecutionsBalance()

	status = sdk.MonitoringStatusOK
	if float64(in) > float64(out) {
		status = sdk.MonitoringStatusWarn
	}
	m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Balance", Value: fmt.Sprintf("%d/%d", in, out), Status: status})

	var nbHooksKafkaTotal int64

	tasks, err := s.Dao.FindAllTasks(ctx)
	if err != nil {
		log.Error(ctx, "Status> Unable to find all tasks: %v", err)
	}

	for _, t := range tasks {
		if t.Type == TypeKafka {
			nbHooksKafkaTotal++
		}

		if t.Stopped {
			m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Task Stopped", Value: t.UUID, Status: sdk.MonitoringStatusWarn})
		}
	}

	m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Hook Kafka", Value: fmt.Sprintf("%d", nbHooksKafkaTotal), Status: status})

	statusConsumer := sdk.MonitoringStatusOK
	if nbKafkaConsumers > nbHooksKafkaTotal {
		statusConsumer = sdk.MonitoringStatusWarn
	}
	m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Hook Kafka Consumers", Value: fmt.Sprintf("%d", nbKafkaConsumers), Status: statusConsumer})

	return m
}

func (s *Service) statusHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var status = http.StatusOK
		return service.WriteJSON(w, s.Status(ctx), status)
	}
}

func (s *Service) getTaskExecutionHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		vars := mux.Vars(r)
		uuid := vars["uuid"]
		timestamp := vars["timestamp"]

		//Load the task
		t := s.Dao.FindTask(ctx, uuid)
		if t == nil {
			return service.WriteJSON(w, t, http.StatusOK)
		}

		//Load the executions
		execs, err := s.Dao.FindAllTaskExecutions(ctx, t)
		if err != nil {
			return sdk.WrapError(err, "Unable to find task executions for %s", uuid)
		}

		for _, e := range execs {
			if strconv.FormatInt(e.Timestamp, 10) == timestamp {
				return service.WriteJSON(w, e, http.StatusOK)
			}
		}

		return sdk.WithStack(sdk.ErrNotFound)
	}
}

func (s *Service) postStopTaskExecutionHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		vars := mux.Vars(r)
		uuid := vars["uuid"]
		timestamp := vars["timestamp"]

		//Load the task
		t := s.Dao.FindTask(ctx, uuid)
		if t == nil {
			return service.WriteJSON(w, t, http.StatusOK)
		}

		//Load the executions
		execs, err := s.Dao.FindAllTaskExecutions(ctx, t)
		if err != nil {
			return sdk.WrapError(err, "Unable to find task executions for %s", uuid)
		}

		for i := range execs {
			e := &execs[i]
			if (strconv.FormatInt(e.Timestamp, 10) == timestamp && e.Status == TaskExecutionDoing) || e.Status == TaskExecutionScheduled || e.Status == TaskExecutionEnqueued {
				e.Status = TaskExecutionDone
				e.LastError = TaskExecutionDone
				e.NbErrors = s.Cfg.RetryError + 1
				s.Dao.SaveTaskExecution(e)
				log.Info(ctx, "Hooks> postStopTaskExecutionHandler> task executions %s:%v has been stoppped", uuid, timestamp)
				return nil
			}
		}

		return nil
	}
}

func (s *Service) postMaintenanceHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		enableS := api.FormString(r, "enable")
		enable, err := strconv.ParseBool(enableS)
		if err != nil {
			return sdk.WrapError(err, "unable to parse maintenance params")
		}

		if err := s.Dao.store.SetWithTTL(MaintenanceHookKey, enable, 0); err != nil {
			return sdk.WrapError(err, "unable to save maintenance state")
		}
		if err := s.Dao.store.Publish(ctx, MaintenanceHookQueue, fmt.Sprintf("%v", enable)); err != nil {
			return sdk.WrapError(err, "unable to publish maintenance state")
		}
		s.Maintenance = enable
		return nil
	}
}
