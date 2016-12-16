package event

import (
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

	Publish(sdk.Event{
		DateEvent:   time.Now().Unix(),
		ActionBuild: ab,
		Action:      eventAction,
		EventType:   sdk.SystemEvent,
	})
}

// PublishPipelineBuild sends a pipelineBuild event
func PublishPipelineBuild(db database.QueryExecuter, pb *sdk.PipelineBuild, eventAction sdk.EventAction, previous *sdk.PipelineBuild) {
	log.Debug("PublishPipelineBuild> pb:%d event:%s", pb.ID, eventAction)

	// get and send all user notifications
	for _, event := range notification.GetUserEvents(db, pb, eventAction, previous) {
		Publish(event)
	}

	Publish(sdk.Event{
		DateEvent:     time.Now().Unix(),
		PipelineBuild: pb,
		Action:        eventAction,
		EventType:     sdk.SystemEvent,
	})
}
