package event

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ovh/cds/sdk"
)

func publishAsCodeEvent(ctx context.Context, payload interface{}, projectKey, workflowName string, u sdk.Identifiable) {
	bts, _ := json.Marshal(payload)
	event := sdk.Event{
		Timestamp:    time.Now(),
		Hostname:     hostname,
		CDSName:      cdsname,
		EventType:    fmt.Sprintf("%T", payload),
		Payload:      bts,
		ProjectKey:   projectKey,
		WorkflowName: workflowName,
	}
	if u != nil {
		event.Username = u.GetUsername()
		event.UserMail = u.GetEmail()
	}
	_ = publishEvent(ctx, event)
}

func PublishAsCodeEvent(ctx context.Context, projectKey, workflowName string, asCodeEvent sdk.AsCodeEvent, u sdk.Identifiable) {
	e := sdk.EventAsCodeEvent{
		Event: asCodeEvent,
	}
	publishAsCodeEvent(ctx, e, projectKey, workflowName, u)
}
