package hooks

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) webhookHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the webhook
		vars := mux.Vars(r)
		uuid := vars["uuid"]

		if uuid == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "Invalid uuid or name")
		}

		//Load the task
		webHook := s.Dao.FindTask(uuid)
		if webHook == nil {
			return sdk.WrapError(sdk.ErrNotFound, "Unknown uuid")
		}

		//Check method
		confValue := webHook.Config[sdk.WebHookModelConfigMethod]
		if r.Method != confValue.Value {
			return sdk.WrapError(sdk.ErrMethodNotAllowed, "Unsupported method %s : %v", r.Method, webHook.Config)
		}

		//Read the body
		req, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return sdk.WrapError(err, "Unable to read request")
		}

		//Prepare a web hook execution
		exec := &sdk.TaskExecution{
			Timestamp: time.Now().UnixNano(),
			Type:      webHook.Type,
			UUID:      webHook.UUID,
			Config:    webHook.Config,
			WebHook: &sdk.WebHookExecution{
				RequestBody:   req,
				RequestHeader: r.Header,
				RequestURL:    r.URL.RawQuery,
			},
		}

		//Save the web hook execution
		s.Dao.SaveTaskExecution(exec)

		//Push the webhook execution in the queue, so it will be executed
		s.Dao.EnqueueTaskExecution(exec)

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
		if err := s.stopTasks(); err != nil {
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
		t := s.Dao.FindTask(uuid)
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
		t := s.Dao.FindTask(uuid)
		if t == nil {
			return sdk.WrapError(sdk.ErrNotFound, "Unknown uuid")
		}

		//Stop the task
		if err := s.stopTask(t); err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "Stop task")
		}
		return nil
	}
}

func (s *Service) postTaskHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//This handler read a sdk.WorkflowNodeHook from the body
		hook := &sdk.WorkflowNodeHook{}
		if err := service.UnmarshalBody(r, hook); err != nil {
			return sdk.WithStack(err)
		}
		if err := s.addTask(ctx, hook); err != nil {
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

		tasks, err := s.Dao.FindAllTasks()
		if err != nil {
			return sdk.WithStack(err)
		}

		execs, err := s.Dao.FindAllTaskExecutionsForTasks(tasks...)
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
		t := s.Dao.FindTask(uuid)
		if t == nil {
			return sdk.WrapError(sdk.ErrNotFound, "Hooks> putTaskHandler> unknown uuid")
		}

		//Stop the task
		if err := s.stopTask(t); err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "Hooks> putTaskHandler> stop task")
		}

		//Save it
		s.Dao.SaveTask(t)

		//Start the task
		if _, err := s.startTask(ctx, t); err != nil {
			return sdk.WrapError(err, "Unable start task %+v", t)
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
		t := s.Dao.FindTask(uuid)
		if t != nil {
			return service.WriteJSON(w, t, http.StatusOK)
		}

		execs, err := s.Dao.FindAllTaskExecutions(t)
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
		t := s.Dao.FindTask(uuid)
		if t != nil {
			return service.WriteJSON(w, t, http.StatusOK)
		}

		//Stop the task
		if err := s.stopTask(t); err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "Hooks> putTaskHandler> stop task")
		}

		//Delete the task
		s.Dao.DeleteTask(t)

		return nil
	}
}

func (s *Service) getTaskExecutionsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		vars := mux.Vars(r)
		uuid := vars["uuid"]

		//Load the task
		t := s.Dao.FindTask(uuid)
		if t == nil {
			return service.WriteJSON(w, t, http.StatusOK)
		}

		//Load the executions
		execs, err := s.Dao.FindAllTaskExecutions(t)
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
		t := s.Dao.FindTask(uuid)
		if t == nil {
			return service.WriteJSON(w, t, http.StatusOK)
		}

		//Stop the task
		if err := s.stopTask(t); err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "Hooks> deleteAllTaskExecutionsHandler> stop task")
		}

		//Load the executions
		execs, err := s.Dao.FindAllTaskExecutions(t)
		if err != nil {
			return sdk.WrapError(err, "Unable to find task executions for %s", uuid)
		}
		for i := range execs {
			s.Dao.DeleteTaskExecution(&execs[i])
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
		hooks := map[string]sdk.WorkflowNodeHook{}
		if err := service.UnmarshalBody(r, &hooks); err != nil {
			return sdk.WithStack(err)
		}

		for uuid := range hooks {
			//Load the task
			t := s.Dao.FindTask(uuid)
			if t == nil {
				continue
			}

			//Stop the task
			if err := s.stopTask(t); err != nil {
				return sdk.WrapError(sdk.ErrNotFound, "Stop task %s", err)
			}
			//Delete the task
			s.Dao.DeleteTask(t)
		}

		return nil
	}
}

func (s *Service) postTaskBulkHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//This handler read a sdk.WorkflowNodeHook from the body
		hooks := map[string]sdk.WorkflowNodeHook{}
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

func (s *Service) addTask(ctx context.Context, h *sdk.WorkflowNodeHook) error {
	//Parse the hook as a task
	t, err := s.hookToTask(h)
	if err != nil {
		return sdk.WrapError(err, "Unable to parse hook")
	}

	//Save the task
	s.Dao.SaveTask(t)

	//Start the task
	if _, err := s.startTask(ctx, t); err != nil {
		return sdk.WrapError(err, "Unable start task %+v", t)
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
	s.Dao.SaveTask(&t)

	// Start the task
	e, err := s.startTask(ctx, &t)
	if err != nil {
		return t, sdk.TaskExecution{}, sdk.WrapError(err, "Unable start task %+v", t)
	}

	return t, *e, nil
}

var errNoTask = errors.New("task not found")

func (s *Service) updateTask(ctx context.Context, h *sdk.WorkflowNodeHook) error {
	//Parse the hook as a task
	t, err := s.hookToTask(h)
	if err != nil {
		return sdk.WrapError(err, "Unable to parse hook")
	}

	task := s.Dao.FindTask(t.UUID)
	if task == nil {
		return errNoTask
	}

	task.Config = t.Config
	_ = s.stopTask(t)
	execs, _ := s.Dao.FindAllTaskExecutions(t)
	for _, e := range execs {
		if e.Status == TaskExecutionScheduled {
			s.Dao.DeleteTaskExecution(&e)
		}
	}
	if _, err := s.startTask(ctx, t); err != nil {
		return sdk.WrapError(err, "Unable start task %+v", t)
	}
	// Save the task
	s.Dao.SaveTask(t)
	return nil
}

func (s *Service) deleteTask(ctx context.Context, h *sdk.WorkflowNodeHook) error {
	//Parse the hook as a task
	t, err := s.hookToTask(h)
	if err != nil {
		return sdk.WrapError(err, "Unable to parse hook")
	}

	switch t.Type {
	case TypeGerrit:
		s.stopGerritHookTask(t)
	}

	//Delete the task
	s.Dao.DeleteTask(t)

	return nil
}

// Status returns sdk.MonitoringStatus, implements interface service.Service
func (s *Service) Status() sdk.MonitoringStatus {
	m := s.CommonMonitoring()

	if s.Dao.store == nil {
		return m
	}

	// hook queue in status
	status := sdk.MonitoringStatusOK
	size := s.Dao.QueueLen()
	if size >= 100 {
		status = sdk.MonitoringStatusAlert
	} else if size >= 10 {
		status = sdk.MonitoringStatusWarn
	}
	m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Queue", Value: fmt.Sprintf("%d", size), Status: status})

	var nbHooksKafkaTotal int64

	tasks, err := s.Dao.FindAllTasks()
	if err != nil {
		log.Error("Status> Unable to find all tasks: %v", err)
	}

	for _, t := range tasks {
		if t.Type == TypeKafka {
			nbHooksKafkaTotal++
		}

		if t.Stopped {
			m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Task Stopped", Value: t.UUID, Status: sdk.MonitoringStatusWarn})
		}

		execs, err := s.Dao.FindAllTaskExecutions(&t)
		if err != nil {
			log.Error("Status> Unable to find all task executions (%s): %v", t.UUID, err)
			continue
		}

		var nbTodo, nbTotal int
		for _, e := range execs {
			if e.ProcessingTimestamp == 0 {
				nbTodo++
			}
		}
		nbTotal = len(execs)

		if nbTodo >= 20 {
			status = sdk.MonitoringStatusAlert
		} else if nbTodo > 10 {
			status = sdk.MonitoringStatusWarn
		}

		if nbTodo > 10 {
			m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Execs Todo " + t.UUID, Value: fmt.Sprintf("%d", nbTodo), Status: status})
		}

		if nbTotal >= s.Cfg.ExecutionHistory*10 {
			status = sdk.MonitoringStatusAlert
		} else if nbTotal >= s.Cfg.ExecutionHistory*5 {
			status = sdk.MonitoringStatusWarn
		}

		if nbTotal >= s.Cfg.ExecutionHistory*2 {
			m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Execs Total " + t.UUID, Value: fmt.Sprintf("%d", nbTotal), Status: status})
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
		return service.WriteJSON(w, s.Status(), status)
	}
}

func (s *Service) getTaskExecutionHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		vars := mux.Vars(r)
		uuid := vars["uuid"]
		timestamp := vars["timestamp"]

		//Load the task
		t := s.Dao.FindTask(uuid)
		if t == nil {
			return service.WriteJSON(w, t, http.StatusOK)
		}

		//Load the executions
		execs, err := s.Dao.FindAllTaskExecutions(t)
		if err != nil {
			return sdk.WrapError(err, "Unable to find task executions for %s", uuid)
		}

		for _, e := range execs {
			if strconv.FormatInt(e.Timestamp, 10) == timestamp {
				return service.WriteJSON(w, e, http.StatusOK)
			}
		}

		return sdk.ErrNotFound
	}
}

func (s *Service) postStopTaskExecutionHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		vars := mux.Vars(r)
		uuid := vars["uuid"]
		timestamp := vars["timestamp"]

		//Load the task
		t := s.Dao.FindTask(uuid)
		if t == nil {
			return service.WriteJSON(w, t, http.StatusOK)
		}

		//Load the executions
		execs, err := s.Dao.FindAllTaskExecutions(t)
		if err != nil {
			return sdk.WrapError(err, "Unable to find task executions for %s", uuid)
		}

		for i := range execs {
			e := &execs[i]
			if strconv.FormatInt(e.Timestamp, 10) == timestamp && e.Status == TaskExecutionDoing || e.Status == TaskExecutionScheduled {
				e.Status = TaskExecutionDone
				e.LastError = TaskExecutionDone
				e.NbErrors = s.Cfg.RetryError + 1
				s.Dao.SaveTaskExecution(e)
				log.Info("Hooks> postStopTaskExecutionHandler> task executions %s:%v has been stoppped", uuid, timestamp)
				return nil
			}
		}

		return nil
	}
}
