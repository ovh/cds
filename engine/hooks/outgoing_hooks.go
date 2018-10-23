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

func (s *Service) nodeRunToTask(nr sdk.WorkflowNodeRun) (sdk.Task, error) {
	if nr.OutgoingHook == nil {
		return sdk.Task{}, fmt.Errorf("Unsupported node type: %d", nr.WorkflowNodeID)
	}
	if nr.OutgoingHook.Config[sdk.HookConfigModelType].Value != sdk.WorkflowHookModelBuiltin {
		return sdk.Task{}, fmt.Errorf("Unsupported hook type: %s", nr.OutgoingHook.Config[sdk.HookConfigModelType].Value)
	}
	configHash, err := hashstructure.Hash(nr.OutgoingHook.Config, nil)
	if err != nil {
		return sdk.Task{}, sdk.WrapError(err, "nodeRunToTask> unable to hash hook config")
	}
	identifier := fmt.Sprintf("%d/%s/%d", nr.WorkflowNodeID, nr.OutgoingHook.Config[sdk.HookConfigModelName].Value, configHash)
	uuid := base64.StdEncoding.EncodeToString([]byte(identifier))

	config := nr.OutgoingHook.Config.Clone()
	config[sdk.HookConfigProject] = sdk.WorkflowNodeHookConfigValue{
		Value: nr.OutgoingHook.Config[sdk.HookConfigProject].Value,
	}
	config[sdk.HookConfigTypeWorkflow] = sdk.WorkflowNodeHookConfigValue{
		Value: nr.OutgoingHook.Config[sdk.HookConfigTypeWorkflow].Value,
	}
	config[ConfigHookRunID] = sdk.WorkflowNodeHookConfigValue{
		Value: nr.UUID,
	}
	config[ConfigNumber] = sdk.WorkflowNodeHookConfigValue{
		Value: strconv.FormatInt(nr.Number, 10),
	}
	config[ConfigSubNumber] = sdk.WorkflowNodeHookConfigValue{
		Value: strconv.FormatInt(nr.SubNumber, 10),
	}
	config[ConfigHookID] = sdk.WorkflowNodeHookConfigValue{
		Value: strconv.FormatInt(nr.WorkflowNodeID, 10),
	}

	switch nr.OutgoingHook.Config[sdk.HookConfigModelName].Value {
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

	return sdk.Task{}, fmt.Errorf("Unsupported hook: %s", nr.OutgoingHook.Config[sdk.HookConfigModelName].Value)
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
		NodeHookID: hookID,
		Start:      time.Now(),
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
		return err
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
		Payload:              sdk.ParametersToMap(hookRun.BuildParameters),
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
		log.Debug("Hooks> workflow %s/%s #%d status: %s", pkey, workflow, run, wr.Status)
		return fmt.Errorf("workflow %s/%s #%s is not at status Building", pkey, workflow, run)
	}

	callbackURL := fmt.Sprintf("/project/%s/workflows/%s/runs/%s/hooks/%s/callback", pkey, workflow, run, hookRunID)
	hookID, _ := strconv.ParseInt(t.Config[ConfigHookID].Value, 10, 64)
	callbackData := sdk.WorkflowNodeOutgoingHookRunCallback{
		NodeHookID: hookID,
		Start:      time.Now(),
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

	mapParams := sdk.ParametersToMap(hookRun.BuildParameters)

	// Interpolate
	method, err := interpolate.Do(t.WebHook.RequestMethod, mapParams)
	if err != nil {
		return handleError(err)
	}

	urls, err := interpolate.Do(t.WebHook.RequestURL, mapParams)
	if err != nil {
		return handleError(err)
	}

	body, err := interpolate.Do(string(t.WebHook.RequestBody), mapParams)
	if err != nil {
		return handleError(err)
	}

	req, err := http.NewRequest(method, urls, bytes.NewBuffer([]byte(body)))
	if err != nil {
		return handleError(err)
	}

	for k, v := range t.WebHook.RequestHeader {
		for _, val := range v {
			val, err = interpolate.Do(val, mapParams)
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
