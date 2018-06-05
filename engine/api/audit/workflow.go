package audit

import (
	"context"
	"fmt"
	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"strings"
)

// ComputeWorkflowAudit Compute audit on workflow
func ComputeWorkflowAudit(c context.Context) {
	chanEvent := make(chan sdk.Event)
	event.Subscribe(chanEvent)

	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("ComputeWorkflowAudit> Exiting: %v", c.Err())
				return
			}
		case e := <-chanEvent:
			if !strings.HasPrefix(e.EventType, "sdk.EventWorkflow") {
				continue
			}

			switch e.EventType {
			case fmt.Sprintf("%T", sdk.EventWorkflowAdd{}):
				a := sdk.AuditWorklflow{
					EventType:   strings.Replace(e.EventType, "sdk.Event", "", -1),
					Created:     e.Timestamp,
					TriggeredBy: e.Username,
				}

			case fmt.Sprintf("%T", sdk.EventWorkflowUpdate{}):
			case fmt.Sprintf("%T", sdk.EventWorkflowDelete{}):
			case fmt.Sprintf("%T", sdk.EventWorkflowPermissionAdd{}):
			case fmt.Sprintf("%T", sdk.EventWorkflowPermissionUpdate{}):
			case fmt.Sprintf("%T", sdk.EventWorkflowPermissionDelete{}):
			}
		}
	}
}
