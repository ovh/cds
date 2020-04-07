package event

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ovh/cds/sdk"
)

// PublishMaintenanceEvent publish maintenance event
func PublishMaintenanceEvent(ctx context.Context, payload interface{}) {
	bts, _ := json.Marshal(payload)
	event := sdk.Event{
		Timestamp: time.Now(),
		Hostname:  hostname,
		CDSName:   cdsname,
		EventType: fmt.Sprintf("%T", payload),
		Payload:   bts,
	}
	_ = publishEvent(ctx, event)
}
