package hooks

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) webhookHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the webhook
		vars := mux.Vars(r)
		uuid := vars["uuid"]

		if uuid == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "Hook> webhookHandler> invalid uuid or name")
		}

		//Load the task
		webHook := s.Dao.FindTask(uuid)
		if webHook == nil {
			return sdk.WrapError(sdk.ErrNotFound, "Hook> webhookHandler> unknown uuid")
		}

		//Check method
		confValue := webHook.Config[sdk.WebHookModelConfigMethod]
		if r.Method != confValue.Value {
			return sdk.WrapError(sdk.ErrMethodNotAllowed, "Hook> webhookHandler> Unsupported method %s : %v", r.Method, webHook.Config)
		}

		//Read the body
		req, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return sdk.WrapError(err, "Hook> webhookHandler> unable to read request")
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
		return api.WriteJSON(w, exec, http.StatusOK)
	}
}

func (s *Service) postTaskHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//This handler read a sdk.WorkflowNodeHook from the body
		hook := &sdk.WorkflowNodeHook{}
		if err := api.UnmarshalBody(r, hook); err != nil {
			return sdk.WrapError(err, "Hooks> postTaskHandler")
		}
		if err := s.addTask(ctx, hook); err != nil {
			return sdk.WrapError(err, "Hooks> postTaskHandler")
		}
		return nil
	}
}

func (s *Service) getTasksHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		tasks, err := s.Dao.FindAllTasks()
		if err != nil {
			return sdk.WrapError(err, "Hooks> getTasksHandler")
		}
		for i := range tasks {
			execs, err := s.Dao.FindAllTaskExecutions(&tasks[i])
			if err != nil {
				log.Error("getTasksHandler> Unable to find all task executions (%s): %v", tasks[i].UUID, err)
				continue
			}

			var nbTodo int
			for _, e := range execs {
				if e.ProcessingTimestamp != 0 {
					nbTodo++
				}
			}
			tasks[i].NbExecutionsTotal = len(execs)
			tasks[i].NbExecutionsTodo = nbTodo
		}
		return api.WriteJSON(w, tasks, http.StatusOK)
	}
}

func (s *Service) putTaskHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		vars := mux.Vars(r)
		uuid := vars["uuid"]
		if uuid == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "Hook> putTaskHandler> invalid uuid")
		}

		//Load the task
		t := s.Dao.FindTask(uuid)
		if t == nil {
			return sdk.WrapError(sdk.ErrNotFound, "Hook> putTaskHandler> unknown uuid")
		}

		//Stop the task
		if err := s.stopTask(t); err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "Hook> putTaskHandler> stop task")
		}

		//Save it
		s.Dao.SaveTask(t)

		//Start the task
		if err := s.startTask(ctx, t); err != nil {
			return sdk.WrapError(err, "Hooks> putTaskHandler> Unable start task %+v", t)
		}

		return nil
	}
}

func (s *Service) getTaskHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		vars := mux.Vars(r)
		uuid := vars["uuid"]

		//Load the task
		t := s.Dao.FindTask(uuid)
		if t != nil {
			return api.WriteJSON(w, t, http.StatusOK)
		}

		execs, err := s.Dao.FindAllTaskExecutions(t)
		if err != nil {
			return sdk.WrapError(err, "Hooks> getTaskHandler> Unable to load executions")
		}

		t.Executions = execs

		return api.WriteJSON(w, t, http.StatusOK)
	}
}

func (s *Service) deleteTaskHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		vars := mux.Vars(r)
		uuid := vars["uuid"]

		//Load the task
		t := s.Dao.FindTask(uuid)
		if t != nil {
			return api.WriteJSON(w, t, http.StatusOK)
		}

		//Stop the task
		if err := s.stopTask(t); err != nil {
			return sdk.WrapError(sdk.ErrNotFound, "Hook> putTaskHandler> stop task")
		}

		//Delete the task
		s.Dao.DeleteTask(t)

		return nil
	}
}

func (s *Service) getTaskExecutionsHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		vars := mux.Vars(r)
		uuid := vars["uuid"]

		//Load the task
		t := s.Dao.FindTask(uuid)
		if t == nil {
			return api.WriteJSON(w, t, http.StatusOK)
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

		return api.WriteJSON(w, t, http.StatusOK)
	}
}

func (s *Service) deleteAllTaskExecutionsHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the task from the URL
		vars := mux.Vars(r)
		uuid := vars["uuid"]

		//Load the task
		t := s.Dao.FindTask(uuid)
		if t == nil {
			return api.WriteJSON(w, t, http.StatusOK)
		}

		//Load the executions
		execs, err := s.Dao.FindAllTaskExecutions(t)
		if err != nil {
			return sdk.WrapError(err, "Unable to find task executions for %s", uuid)
		}
		for i := range execs {
			s.Dao.DeleteTaskExecution(&execs[i])
		}

		return api.WriteJSON(w, t, http.StatusOK)
	}
}

func (s *Service) deleteTaskBulkHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		hooks := map[string]sdk.WorkflowNodeHook{}
		if err := api.UnmarshalBody(r, &hooks); err != nil {
			return sdk.WrapError(err, "Hooks> postTaskBulkHandler")
		}

		for uuid := range hooks {
			//Load the task
			t := s.Dao.FindTask(uuid)
			if t == nil {
				continue
			}

			//Stop the task
			if err := s.stopTask(t); err != nil {
				return sdk.WrapError(sdk.ErrNotFound, "Hook> putTaskHandler> stop task %s", err)
			}
			//Delete the task
			s.Dao.DeleteTask(t)
		}

		return nil
	}
}

func (s *Service) postTaskBulkHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//This handler read a sdk.WorkflowNodeHook from the body
		hooks := map[string]sdk.WorkflowNodeHook{}
		if err := api.UnmarshalBody(r, &hooks); err != nil {
			return sdk.WrapError(err, "Hooks> postTaskBulkHandler")
		}

		for _, hook := range hooks {
			if err := s.addTask(ctx, &hook); err != nil {
				return sdk.WrapError(err, "Hooks> postTaskBulkHandler")
			}
		}
		return api.WriteJSON(w, hooks, http.StatusOK)
	}
}

func (s *Service) addTask(ctx context.Context, h *sdk.WorkflowNodeHook) error {
	//Parse the hook as a task
	t, err := s.hookToTask(h)
	if err != nil {
		return sdk.WrapError(err, "Hooks> addTask> Unable to parse hook")
	}

	//Save the task
	s.Dao.SaveTask(t)

	//Start the task
	if err := s.startTask(ctx, t); err != nil {
		return sdk.WrapError(err, "Hooks> addTask> Unable start task %+v", t)
	}
	return nil
}

// Status returns sdk.MonitoringStatus, implements interface service.Service
func (s *Service) Status() sdk.MonitoringStatus {
	m := s.CommonMonitoring()

	if s.Dao.store != nil {
		// hook queue in status
		status := sdk.MonitoringStatusOK
		size := s.Dao.QueueLen()
		if size >= 100 {
			status = sdk.MonitoringStatusAlert
		} else if size >= 10 {
			status = sdk.MonitoringStatusWarn
		}
		m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Queue", Value: fmt.Sprintf("%d", size), Status: status})

		tasks, err := s.Dao.FindAllTasks()
		if err != nil {
			log.Error("Status> Unable to find all tasks: %v", err)
		}
		for _, t := range tasks {
			execs, err := s.Dao.FindAllTaskExecutions(&t)
			if err != nil {
				log.Error("Status> Unable to find all task executions (%s): %v", t.UUID, err)
				continue
			}

			var nbTodo, nbTotal int
			for _, e := range execs {
				if e.ProcessingTimestamp != 0 {
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

			if nbTotal >= s.Cfg.ExecutionHistory*5 {
				status = sdk.MonitoringStatusAlert
			} else if nbTotal >= s.Cfg.ExecutionHistory*2 {
				status = sdk.MonitoringStatusWarn
			}

			if nbTotal >= s.Cfg.ExecutionHistory*2 {
				m.Lines = append(m.Lines, sdk.MonitoringStatusLine{Component: "Execs Total " + t.UUID, Value: fmt.Sprintf("%d", nbTotal), Status: status})
			}
		}
	}

	return m
}

func (s *Service) statusHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var status = http.StatusOK
		return api.WriteJSON(w, s.Status(), status)
	}
}
