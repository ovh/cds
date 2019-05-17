package event

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/fatih/structs"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var store cache.Store

func publishEvent(e sdk.Event) {
	if store == nil {
		return
	}

	store.Enqueue("events", e)

	// send to cache for cds repositories manager
	var toSkipSendReposManager bool
	// the StatusWaiting is not useful to be sent on repomanager.
	// the building status (or success / failed) is already sent just after
	if e.EventType == fmt.Sprintf("%T", sdk.EventRunWorkflowNode{}) {
		if e.Payload["Status"] == sdk.StatusWaiting.String() {
			toSkipSendReposManager = true
		}
	}
	if !toSkipSendReposManager {
		store.Enqueue("events_repositoriesmanager", e)
	}

	b, err := json.Marshal(e)
	if err != nil {
		log.Warning("%v", sdk.WrapError(err, "Cannot marshal event %+v", e))
		return
	}
	store.Publish("events_pubsub", string(b))
}

// Publish sends a event to a queue
//func Publish(event sdk.Event, eventType string) {
func Publish(payload interface{}, u sdk.Identifiable) {
	p := structs.Map(payload)
	var projectKey, applicationName, pipelineName, environmentName, workflowName string
	if v, ok := p["ProjectKey"]; ok {
		projectKey = v.(string)
	}
	if v, ok := p["ApplicationName"]; ok {
		applicationName = v.(string)
	}
	if v, ok := p["PipelineName"]; ok {
		pipelineName = v.(string)
	}
	if v, ok := p["EnvironmentName"]; ok {
		environmentName = v.(string)
	}
	if v, ok := p["WorkflowName"]; ok {
		workflowName = v.(string)
	}

	event := sdk.Event{
		Timestamp:       time.Now(),
		Hostname:        hostname,
		CDSName:         cdsname,
		EventType:       fmt.Sprintf("%T", payload),
		Payload:         p,
		ProjectKey:      projectKey,
		ApplicationName: applicationName,
		PipelineName:    pipelineName,
		EnvironmentName: environmentName,
		WorkflowName:    workflowName,
	}
	if u != nil {
		event.Username = u.GetUsername()
		event.UserMail = u.Email()
	}
	publishEvent(event)
}
