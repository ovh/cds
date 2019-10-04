package event

import (
	"fmt"
	"time"

	"github.com/fatih/structs"
	"github.com/ovh/cds/sdk"
)

// PublishMaintenanceEvent publish maintenance event
func PublishMaintenanceEvent(payload interface{}) {
	event := sdk.Event{
		Timestamp: time.Now(),
		Hostname:  hostname,
		CDSName:   cdsname,
		EventType: fmt.Sprintf("%T", payload),
		Payload:   structs.Map(payload),
	}
	publishEvent(event)
}
