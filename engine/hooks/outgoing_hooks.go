package hooks

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) startOutgoingHookTask(t *sdk.Task) (*sdk.TaskExecution, error) {
	now := time.Now()
	//Craft a new execution
	exec := &sdk.TaskExecution{
		Timestamp: now.UnixNano(),
		Status:    TaskExecutionScheduled,
		Type:      t.Type,
		UUID:      t.UUID,
		Config:    t.Config,
		ScheduledTask: &sdk.ScheduledTaskExecution{
			DateScheduledExecution: fmt.Sprintf("%v", now),
		},
	}

	s.Dao.SaveTaskExecution(exec) //We don't push in queue, we will the scheduler to run it
	log.Debug("Hooks> Outgoing hook task  %s ready", t.UUID)

	return exec, nil
}

func (s *Service) doOutgoingWebHookExecution(t *sdk.TaskExecution) error {
	log.Debug("Hooks> Processing outgoing webhook %s %s", t.UUID, t.Type)

	pkey := t.Config[sdk.HookConfigProject].Value
	workflow := t.Config[sdk.HookConfigWorkflow].Value
	run := t.Config[ConfigNumber].Value
	hookRunID := t.Config[ConfigHookRunID].Value

	callbackURL := fmt.Sprintf("/project/%s/workflows/%s/runs/%s/hooks/%s/callback", pkey, workflow, run, hookRunID)
	hookID, _ := strconv.ParseInt(t.Config[ConfigHookID].Value, 10, 64)
	callbackData := sdk.WorkflowNodeOutgoingHookRunCallback{
		WorkflowNodeOutgoingHookID: hookID,
		Start: time.Now(),
	}

	var handleError = func(err error) {
		if err == nil {
			return
		}
		log.Error(err.Error())
		t.LastError = err.Error()
		t.NbErrors++

		if t.NbErrors >= s.Cfg.RetryError {
			// Send error callback
			callbackData.Done = time.Now()
			callbackData.Status = sdk.StatusFail.String()
			callbackData.Log = err.Error()
		}
	}

	u := t.Config["URL"].Value
	if _, err := url.ParseQuery(u); err != nil {
		handleError(err)
		return nil
	}
	method := t.Config["method"].Value
	payload := t.Config["payload"].Value
	req, err := http.NewRequest(method, u, strings.NewReader(payload))
	if err != nil {
		handleError(err)
		return nil
	}

	req.Header.Set("Content-Type", "application/json")

	var logBuffer bytes.Buffer
	logBuffer.WriteString("Request:\n")
	dump, _ := httputil.DumpRequestOut(req, true)
	logBuffer.Write(dump) // nolint

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		handleError(err)
		return nil
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		handleError(err)
		return nil
	}

	if res.StatusCode >= 400 {
		err := fmt.Errorf("HTTP Status %d : %v", res.StatusCode, string(body))
		handleError(err)
		return nil
	}

	// Prepare the callback
	logBuffer.WriteString("\nResponse:\n")
	dump, _ = httputil.DumpResponse(res, true)
	logBuffer.Write(dump) // nolint

	callbackData.Done = time.Now()
	callbackData.Log = logBuffer.String()

	// Post the callback
	if _, err := s.Client.(cdsclient.Raw).PostJSON(callbackURL, callbackData, nil); err != nil {
		log.Error("unable to perform outgoing hook callback")
		return err
	}

	return nil
}
