package event

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ovh/cds/sdk"
)

// PublishOperation publish operation event.
func PublishOperation(ctx context.Context, projectKey string, o sdk.Operation, u sdk.Identifiable) {
	e := sdk.EventOperation{Operation: o}

	bts, _ := json.Marshal(e)
	event := sdk.Event{
		Timestamp:     time.Now(),
		Hostname:      hostname,
		CDSName:       cdsname,
		EventType:     fmt.Sprintf("%T", e),
		Payload:       bts,
		ProjectKey:    projectKey,
		OperationUUID: o.UUID,
	}
	if u != nil {
		event.UserMail = u.GetEmail()
		event.Username = u.GetUsername()
	}

	publishEvent(ctx, event)
}
