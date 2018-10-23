package hooks

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"time"

	"github.com/mitchellh/hashstructure"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/interpolate"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) outgoingHookToTask(h sdk.WorkflowNodeOutgoingHookRun) (sdk.Task, error) {
	if h.Hook.WorkflowHookModel.Type != sdk.WorkflowHookModelBuiltin {
		return sdk.Task{}, fmt.Errorf("Unsupported hook type: %s", h.Hook.WorkflowHookModel.Type)
	}
	configHash, err := hashstructure.Hash(h.Hook.Config, nil)
	if err != nil {
		return sdk.Task{}, sdk.WrapError(err, "unable to hash hook config")
	}
	identifier := fmt.Sprintf("%s/%s/%d", h.Hook.Name, h.Hook.WorkflowHookModel.Name, configHash)
	uuid := base64.StdEncoding.EncodeToString([]byte(identifier))

	config := h.Hook.Config.Clone()
	config[sdk.HookConfigProject] = sdk.WorkflowNodeHookConfigValue{
		Value: h.Hook.Config[sdk.HookConfigProject].Value,
	}
	config[sdk.HookConfigTypeWorkflow] = sdk.WorkflowNodeHookConfigValue{
		Value: h.Hook.Config[sdk.HookConfigTypeWorkflow].Value,
	}
	config[ConfigHookRunID] = sdk.WorkflowNodeHookConfigValue{
		Value: h.HookRunID,
	}
	config[ConfigNumber] = sdk.WorkflowNodeHookConfigValue{
		Value: strconv.FormatInt(h.Number, 10),
	}
	config[ConfigSubNumber] = sdk.WorkflowNodeHookConfigValue{
		Value: strconv.FormatInt(h.SubNumber, 10),
	}
	config[ConfigHookID] = sdk.WorkflowNodeHookConfigValue{
		Value: strconv.FormatInt(h.WorkflowNodeOutgoingHookID, 10),
	}

	switch h.Hook.WorkflowHookModel.Name {
	case sdk.WebHookModelName:
		return sdk.Task{
			UUID:   uuid,
			Type:   TypeOutgoingWebHook,
			Config: config,
		}, nil
	case sdk.WorkflowModelName:
		return sdk.Task{
			UUID:   uuid,
			Type:   TypeOutgoingWorkflow,
			Config: config,
		}, nil
	}

	return sdk.Task{}, fmt.Errorf("Unsupported hook: %s", h.Hook.WorkflowHookModel.Name)
}

func (s *Service) startOutgoingWebHookTask(t *sdk.Task) (*sdk.TaskExecution, error) {
	now := time.Now()

	u := t.Config["URL"].Value
	if _, err := url.ParseQuery(u); err != nil {
		return nil, err
	}
	method := t.Config["method"].Value
	payload := t.Config["payload"].Value
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	//Craft a new execution
	exec := &sdk.TaskExecution{
		Timestamp: now.UnixNano(),
		Status:    TaskExecutionScheduled,
		Type:      t.Type,
		UUID:      t.UUID,
		Config:    t.Config,
		WebHook: &sdk.WebHookExecution{
			RequestURL:    u,
			RequestBody:   []byte(payload),
			RequestHeader: headers,
			RequestMethod: method,
		},
	}

	s.Dao.SaveTaskExecution(exec) //We don't push in queue, we will the scheduler to run it
	log.Debug("Hooks> Outgoing hook task  %s ready", t.UUID)

	return exec, nil
}

func (s *Service) startOutgoingWorkflowTask(t *sdk.Task) (*sdk.TaskExecution, error) {
	now := time.Now()

	//Craft a new execution
	exec := &sdk.TaskExecution{
		Timestamp: now.UnixNano(),
		Status:    TaskExecutionScheduled,
		Type:      t.Type,
		UUID:      t.UUID,
		Config:    t.Config,
	}

	s.Dao.SaveTaskExecution(exec) //We don't push in queue, we will the scheduler to run it
	log.Debug("Hooks> Outgoing hook task  %s ready", t.UUID)

	return exec, nil
}

func (s *Service) doOutgoingWorkflowExecution(t *sdk.TaskExecution) error {
	pkey := t.Config[sdk.HookConfigProject].Value
	workflow := t.Config[sdk.HookConfigWorkflow].Value
	run := t.Config[ConfigNumber].Value
	hookRunID := t.Config[ConfigHookRunID].Value

	targetProject := t.Config[sdk.HookConfigTargetProject].Value
	targetWorkflow := t.Config[sdk.HookConfigTargetWorkflow].Value
	targetHook := t.Config[sdk.HookConfigTargetHook].Value

	log.Debug("Hooks> Processing outgoing workflow hook %s %s (%s/%s #%s) => (%s/%s/%s)",
		t.UUID, t.Type, pkey, workflow, run, targetProject, targetWorkflow, targetHook)

	runNumber, err := strconv.ParseInt(run, 10, 64)
	if err != nil {
		return sdk.WrapError(err, "startOutgoingWorkflowTask")
	}

	callbackURL := fmt.Sprintf("/project/%s/workflows/%s/runs/%s/hooks/%s/callback", pkey, workflow, run, hookRunID)
	hookID, _ := strconv.ParseInt(t.Config[ConfigHookID].Value, 10, 64)
	callbackData := sdk.WorkflowNodeOutgoingHookRunCallback{
		WorkflowNodeOutgoingHookID: hookID,
		Start: time.Now(),
	}

	var handleError = func(err error) error {
		if err == nil {
			return nil
		}
		log.Error(err.Error())
		t.LastError = err.Error()
		t.NbErrors++

		if t.NbErrors >= s.Cfg.RetryError {
			// Send error callback
			callbackData.Done = time.Now()
			callbackData.Status = sdk.StatusFail.String()
			callbackData.Log = err.Error()

			// Post the callback
			if _, err := s.Client.(cdsclient.Raw).PostJSON(context.Background(), callbackURL, callbackData, nil); err != nil {
				log.Error("unable to perform outgoing hook callback: %v", err)
				return fmt.Errorf("unable to perform outgoing hook callback: %v", err)
			}
		}
		return nil
	}

	wr, err := s.Client.WorkflowRunGet(pkey, workflow, runNumber)
	if err != nil {
		return handleError(err)
	}

	hookRun := wr.GetOutgoingHookRun(hookRunID)
	if hookRun == nil {
		return handleError(errors.New("unable to find hook" + hookRunID))
	}

	evt := sdk.WorkflowNodeRunHookEvent{
		WorkflowNodeHookUUID: targetHook,
		Payload:              hookRun.Params,
	}
	evt.ParentWorkflow.Key = pkey
	evt.ParentWorkflow.Name = workflow
	evt.ParentWorkflow.Run = runNumber
	evt.ParentWorkflow.HookRunID = hookRunID

	targetRun, err := s.Client.WorkflowRunFromHook(targetProject, targetWorkflow, evt)
	if err != nil {
		return handleError(err)
	}

	callbackData.Log = fmt.Sprintf("Workflow %s/%s #%d.%d has been started", targetProject, targetWorkflow, targetRun.Number, targetRun.LastSubNumber)
	callbackData.Status = sdk.StatusBuilding.String()
	callbackData.WorkflowRunNumber = &targetRun.Number

	// Post the callback
	if _, err := s.Client.(cdsclient.Raw).PostJSON(context.Background(), callbackURL, callbackData, nil); err != nil {
		log.Error("unable to perform outgoing hook callback: %v", err)
		return err
	}
	return nil
}

func (s *Service) doOutgoingWebHookExecution(t *sdk.TaskExecution) error {
	pkey := t.Config[sdk.HookConfigProject].Value
	workflow := t.Config[sdk.HookConfigWorkflow].Value
	run := t.Config[ConfigNumber].Value
	hookRunID := t.Config[ConfigHookRunID].Value
	log.Debug("Hooks> Processing outgoing webhook %s %s (%s/%s #%s)", t.UUID, t.Type, pkey, workflow, run)
	irun, _ := strconv.ParseInt(run, 10, 64)

	// Checkin if the workflow is still waiting for the callback
	wr, err := s.Client.WorkflowRunGet(pkey, workflow, irun)
	if err != nil {
		return nil
	}

	if wr.Status != sdk.StatusBuilding.String() {
		log.Debug("Hooks> workflow %s/%s #%s status: %s", pkey, workflow, run, wr.Status)
		return fmt.Errorf("workflow %s/%s #%s is not at status Building", pkey, workflow, run)
	}

	callbackURL := fmt.Sprintf("/project/%s/workflows/%s/runs/%s/hooks/%s/callback", pkey, workflow, run, hookRunID)
	hookID, _ := strconv.ParseInt(t.Config[ConfigHookID].Value, 10, 64)
	callbackData := sdk.WorkflowNodeOutgoingHookRunCallback{
		WorkflowNodeOutgoingHookID: hookID,
		Start: time.Now(),
	}

	var handleError = func(err error) error {
		if err == nil {
			return nil
		}
		log.Error(err.Error())
		t.LastError = err.Error()
		t.NbErrors++

		if t.NbErrors >= s.Cfg.RetryError {
			// Send error callback
			callbackData.Done = time.Now()
			callbackData.Status = sdk.StatusFail.String()
			callbackData.Log = err.Error()

			// Post the callback
			if _, err := s.Client.(cdsclient.Raw).PostJSON(context.Background(), callbackURL, callbackData, nil); err != nil {
				log.Error("unable to perform outgoing hook callback: %v", err)
				return fmt.Errorf("unable to perform outgoing hook callback: %v", err)
			}
		}
		return nil
	}

	hookRun := wr.GetOutgoingHookRun(hookRunID)
	if hookRun == nil {
		return handleError(errors.New("unable to find hook" + hookRunID))
	}

	// Get Secrets
	detailsURL := fmt.Sprintf("/project/%s/workflows/%s/runs/%s/hooks/%s/details", pkey, workflow, run, hookRunID)
	if _, err := s.Client.(cdsclient.Raw).GetJSON(context.Background(), detailsURL, hookRun); err != nil {
		return handleError(sdk.WrapError(err, "unable to retrieve hook details"))
	}

	// Interpolate
	method, err := interpolate.Do(t.WebHook.RequestMethod, hookRun.Params)
	if err != nil {
		return handleError(err)
	}

	urls, err := interpolate.Do(t.WebHook.RequestURL, hookRun.Params)
	if err != nil {
		return handleError(err)
	}

	body, err := interpolate.Do(string(t.WebHook.RequestBody), hookRun.Params)
	if err != nil {
		return handleError(err)
	}

	req, err := http.NewRequest(method, urls, bytes.NewBuffer([]byte(body)))
	if err != nil {
		return handleError(err)
	}

	for k, v := range t.WebHook.RequestHeader {
		for _, val := range v {
			val, err = interpolate.Do(val, hookRun.Params)
			if err != nil {
				return handleError(err)
			}
			req.Header.Add(k, val)
		}
	}

	var logBuffer bytes.Buffer
	logBuffer.WriteString("Request:\n")
	dump, _ := httputil.DumpRequestOut(req, true)
	logBuffer.Write(dump) // nolint

	http.DefaultClient.Timeout = 60 * time.Second
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return handleError(err)
	}

	// Prepare the callback
	logBuffer.WriteString("\n\nResponse:\n")
	dump, _ = httputil.DumpResponse(res, true)
	logBuffer.Write(dump) // nolint

	if res.StatusCode >= 400 {
		err := fmt.Errorf("HTTP Status %d", res.StatusCode)
		return handleError(err)
	}

	callbackData.Done = time.Now()
	callbackData.Log = logBuffer.String()
	callbackData.Status = sdk.StatusSuccess.String()

	// Post the callback
	if _, err := s.Client.(cdsclient.Raw).PostJSON(context.Background(), callbackURL, callbackData, nil); err != nil {
		log.Error("unable to perform outgoing hook callback: %v", err)
		return err
	}

	return nil
}
