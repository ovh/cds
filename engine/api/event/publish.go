package event

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/api/notification"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// Publish sends a event to a queue
func Publish(event sdk.Event) {
	cache.Enqueue("events", event)
}

// PublishActionBuild sends a actionBuild event
func PublishActionBuild(ab *sdk.ActionBuild, eventAction sdk.EventAction) {
	log.Debug("PublishActionBuild> pb:%d ab:%d event:%s", ab.PipelineBuildID, ab.ID, eventAction)

	payload, err := json.Marshal(ab)
	if err != nil {
		log.Critical("PublishActionBuild> error while converting payload: %s", err)
	}

	Publish(sdk.Event{
		DateEvent:   time.Now().Unix(),
		Payload:     payload,
		Action:      eventAction,
		EventSource: sdk.SystemEvent,
		EventType:   fmt.Sprintf("%T", ab),
	})
}

// PublishPipelineBuild sends a pipelineBuild event
func PublishPipelineBuild(db database.QueryExecuter, pb *sdk.PipelineBuild, eventAction sdk.EventAction, previous *sdk.PipelineBuild) {
	log.Debug("PublishPipelineBuild> pb:%d event:%s", pb.ID, eventAction)

	// get and send all user notifications
	for _, event := range notification.GetUserEvents(db, pb, eventAction, previous) {
		Publish(event)
	}

	payload, err := json.Marshal(pb)
	if err != nil {
		log.Critical("PublishPipelineBuild> error while converting payload: %s", err)
	}

	Publish(sdk.Event{
		DateEvent:   time.Now().Unix(),
		Payload:     payload,
		Action:      eventAction, // create / update
		EventSource: sdk.SystemEvent,
		EventType:   fmt.Sprintf("%T", pb),
	})
}
