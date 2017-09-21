package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"time"

	"github.com/fsamin/go-dump"
	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//This are all the types
const (
	TypeWebHook   = "Webhook"
	TypeScheduler = "Scheduler"
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
	hooks, err := s.cds.WorkflowAllHooksList()
	if err != nil {
		return sdk.WrapError(err, "synchronizeTasks> Unable to get hooks")
	}

	for _, h := range hooks {
		if h.Config["project"] == "" || h.Config["workflow"] == "" {
			log.Error("Hook> Unable to synchronize task %+v: %v", h, err)
			continue
		}
		t, err := s.hookToTask(h)
		if err != nil {
			log.Error("Hook> Unable to synchronize task %+v: %v", h, err)
			continue
		}
		s.Dao.SaveTask(t)
	}

	return nil
}

func (s *Service) hookToTask(h sdk.WorkflowNodeHook) (*Task, error) {
	if h.WorkflowHookModel.Type != sdk.WorkflowHookModelBuiltin {
		return nil, fmt.Errorf("Unsupported hook type: %s", h.WorkflowHookModel.Type)
	}

	switch h.WorkflowHookModel.Name {
	case workflow.WebHookModel.Name:
		return &Task{
			UUID:   h.UUID,
			Type:   TypeWebHook,
			Config: h.Config,
		}, nil
	case workflow.SchedulerModel.Name:
		return &Task{
			UUID:   h.UUID,
			Type:   TypeScheduler,
			Config: h.Config,
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
		return sdk.WrapError(err, "Hook> startTasks> Unable to find all tasks")
	}

	log.Debug("Hooks> Starting %d tasks", len(tasks))

	//Start the tasks
	for i := range tasks {
		t := &tasks[i]
		if err := s.startTask(c, t); err != nil {
			log.Error("Hooks> runLongRunningTasks> Unable to start tasks: %v", err)
			continue
		}
	}
	return nil
}

func (s *Service) startTask(ctx context.Context, t *Task) error {
	log.Info("Hooks> Starting task %s", t.UUID)
	switch t.Type {
	case TypeWebHook:
		log.Debug("Hooks> Webhook tasks %s ready", t.UUID)
		return nil
	case TypeScheduler:
		log.Error("Hooks> Scheduler tasks %s not yet implemented", t.UUID)
		return nil
	default:
		return fmt.Errorf("Unsupported task type %s", t.Type)
	}
}

func (s *Service) stopTask(ctx context.Context, t *Task) error {
	log.Info("Hooks> Stopping task %s", t.UUID)
	switch t.Type {
	case TypeWebHook:
		log.Debug("Hooks> Webhook tasks %s has been stopped", t.UUID)
		return nil
	default:
		return fmt.Errorf("Unsupported task type %s", t.Type)
	}
}

func (s *Service) doTask(ctx context.Context, t *TaskExecution) error {
	switch {
	// Do a WebHook
	case t.WebHook != nil:
		log.Debug("Hooks> Processing webhook %s", t.UUID)

		// Prepare a struct to send to CDS API
		h := sdk.WorkflowNodeRunHookEvent{
			WorkflowNodeHookUUID: t.UUID,
		}

		// Compute the payload, from the header, the body and the url
		// For all requests, parse the raw query from the URL
		values, err := url.ParseQuery(t.WebHook.RequestURL)
		if err != nil {
			return sdk.WrapError(err, "Hooks> Unable to parse query url %s", t.WebHook.RequestURL)
		}

		// For POST, PUT, and PATCH requests, it also parses the request body as a form
		if t.Config["method"] == "POST" || t.Config["method"] == "PUT" || t.Config["method"] == "PATCH" {
			//Depending on the content type, we should not read the body the same way
			header := http.Header(t.WebHook.RequestHeader)
			ct := header.Get("Content-Type")
			// RFC 2616, section 7.2.1 - empty type
			//   SHOULD be treated as application/octet-stream
			if ct == "" {
				ct = "application/octet-stream"
			}
			//Parse the content type
			ct, _, err = mime.ParseMediaType(ct)
			switch {
			case ct == "application/x-www-form-urlencoded":
				formValues, err := url.ParseQuery(string(t.WebHook.RequestBody))
				if err == nil {
					return sdk.WrapError(err, "Hooks> Unable to parse body %s", t.WebHook.RequestBody)
				}
				copyValues(values, formValues)
			case ct == "application/json":
				var bodyJSON interface{}

				//Try to parse the body as an array
				bodyJSONArray := []interface{}{}
				if err := json.Unmarshal(t.WebHook.RequestBody, &bodyJSONArray); err != nil {

					//Try to parse the body as a map
					bodyJSONMap := map[string]interface{}{}
					if err2 := json.Unmarshal(t.WebHook.RequestBody, &bodyJSONMap); err2 == nil {
						bodyJSON = bodyJSONMap
					}
				} else {
					bodyJSON = bodyJSONArray
				}

				//Go Dump
				m, err := dump.ToMap(bodyJSON, dump.WithDefaultLowerCaseFormatter())
				if err == nil {
					return sdk.WrapError(err, "Hooks> Unable to dump body %s", t.WebHook.RequestBody)
				}

				//Add the map content to values
				for k, v := range m {
					values.Add(k, v)
				}
			}
		}

		//try to find some specific values
		payloadValues := map[string]string{}
		for k := range values {
			switch k {
			case "branch", "ref":
				payloadValues["git.branch"] = values.Get(k)
			case "hash", "checkout_sha":
				payloadValues["git.hash"] = values.Get(k)
			case "message", "object_kind":
				payloadValues["git.message"] = values.Get(k)
			case "author", "user_name":
				payloadValues["git.author"] = values.Get(k)
			default:
				payloadValues[k] = values.Get(k)
			}
		}

		//Set the payload
		h.Payload = payloadValues

		// Call CDS API
		run, err := s.cds.WorkflowRunFromHook(t.Config["project"], t.Config["workflow"], h)
		if err != nil {
			return sdk.WrapError(err, "Hooks> Unable to run workflow")
		}

		//Save the run number
		t.WorkflowRun = run.Number
		log.Info("Hooks> workflow %s/%s#%d has been triggered", t.Config["project"], t.Config["workflow"], run.Number)

		return nil
	default:
		return fmt.Errorf("Unsupported task type %s", t.Type)
	}
}

func copyValues(dst, src url.Values) {
	for k, vs := range src {
		for _, value := range vs {
			dst.Add(k, value)
		}
	}
}
