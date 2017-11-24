package hooks

import (
	"context"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func (s *Service) webhookHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//Get the UUID of the webhook
		vars := mux.Vars(r)
		uuid := vars["uuid"]
		if uuid == "" {
			return sdk.WrapError(sdk.ErrWrongRequest, "Hook> webhookHandler> invalid uuid")
		}

		//Load the task
		webHook := s.Dao.FindTask(uuid)
		if webHook == nil {
			return sdk.WrapError(sdk.ErrNotFound, "Hook> webhookHandler> unknown uuid")
		}

		//Check method
		confValue := webHook.Config["method"]
		if r.Method != confValue.Value {
			return sdk.WrapError(sdk.ErrMethodNotAllowed, "Hook> webhookHandler> Unsupported method %s : %v", r.Method, webHook.Config)
		}

		//Read the body
		req, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return sdk.WrapError(err, "Hook> webhookHandler> unable to read request")
		}

		//Prepare a web hook execution
		exec := &TaskExecution{
			Timestamp: time.Now().UnixNano(),
			Type:      webHook.Type,
			UUID:      webHook.UUID,
			Config:    webHook.Config,
			WebHook: &WebHookExecution{
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
		return api.WriteJSON(w, r, exec, http.StatusOK)
	}
}

func (s *Service) postTaskHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		//This handler read a sdk.WorkflowNodeHook from the body
		hook := &sdk.WorkflowNodeHook{}
		if err := api.UnmarshalBody(r, hook); err != nil {
			return sdk.WrapError(err, "Hooks> postTaskHandler")
		}

		//Parse the hook as a task
		t, err := s.hookToTask(hook)
		if err != nil {
			return sdk.WrapError(err, "Hooks> postTaskHandler> Unable to parse hook")
		}

		//Save the task
		s.Dao.SaveTask(t)

		//Start the task
		if err := s.startTask(ctx, t); err != nil {
			return sdk.WrapError(err, "Hooks> postTaskHandler> Unable start task %+v", t)
		}

		return api.WriteJSON(w, r, t, http.StatusOK)
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
		if err := s.stopTask(ctx, t); err != nil {
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
			return api.WriteJSON(w, r, t, http.StatusOK)
		}

		return api.WriteJSON(w, r, t, http.StatusOK)
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
			return api.WriteJSON(w, r, t, http.StatusOK)
		}

		//Stop the task
		if err := s.stopTask(ctx, t); err != nil {
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
		if t != nil {
			return api.WriteJSON(w, r, t, http.StatusOK)
		}

		//Load the executions
		execs, err := s.Dao.FindAllTaskExecutions(t)
		if err != nil {
			return sdk.WrapError(err, "Unable to find task executions for %s", uuid)
		}

		return api.WriteJSON(w, r, execs, http.StatusOK)
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
			if err := s.stopTask(ctx, t); err != nil {
				return sdk.WrapError(sdk.ErrNotFound, "Hook> putTaskHandler> stop task")
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

		for k, hook := range hooks {
			//Parse the hook as a task
			t, err := s.hookToTask(&hook)
			if err != nil {
				return sdk.WrapError(err, "Hooks> postTaskBulkHandler> Unable to parse hook")
			}

			hooks[k] = hook

			//Save the task
			s.Dao.SaveTask(t)

			//Start the task
			if err := s.startTask(ctx, t); err != nil {
				return sdk.WrapError(err, "Hooks> postTaskBulkHandler> Unable start task %+v", t)
			}

		}
		return api.WriteJSON(w, r, hooks, http.StatusOK)
	}
}
