package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/url"

	"github.com/fsamin/go-dump"

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

func (s *Service) doLongRunningTask(ctx context.Context, t *LongRunningTaskExecution) error {

	switch t.Type {

	case TypeWebHook:
		log.Debug("Hooks> Processing webhook %s", t.UUID)

		// Prepare a struct to send to CDS API
		h := sdk.WorkflowNodeRunHookEvent{
			WorkflowNodeHookUUID: t.UUID,
		}

		// Compute the payload, from the header, the body and the url
		// For all requests, parse the raw query from the URL
		values, err := url.ParseQuery(t.RequestURL)
		if err != nil {
			return sdk.WrapError(err, "Hooks> Unable to parse query url %s", t.RequestURL)
		}

		// For POST, PUT, and PATCH requests, it also parses the request body as a form
		if t.Config["method"] == "POST" || t.Config["method"] == "PUT" || t.Config["method"] == "PATCH" {
			//Depending on the content type, we should not read the body the same way
			header := http.Header(t.RequestHeader)
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
				formValues, err := url.ParseQuery(string(t.RequestBody))
				if err == nil {
					return sdk.WrapError(err, "Hooks> Unable to parse body %s", t.RequestBody)
				}
				copyValues(values, formValues)
			case ct == "application/json":
				var bodyJSON interface{}

				//Try to parse the body as an array
				bodyJSONArray := []interface{}{}
				if err := json.Unmarshal(t.RequestBody, &bodyJSONArray); err != nil {

					//Try to parse the body as a map
					bodyJSONMap := map[string]interface{}{}
					if err2 := json.Unmarshal(t.RequestBody, &bodyJSONMap); err2 == nil {
						bodyJSON = bodyJSONMap
					}
				} else {
					bodyJSON = bodyJSONArray
				}

				//Go Dump
				m, err := dump.ToMap(bodyJSON, dump.WithDefaultLowerCaseFormatter())
				if err == nil {
					return sdk.WrapError(err, "Hooks> Unable to dump body %s", t.RequestBody)
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
