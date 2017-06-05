package repositoriesmanager

import (
	"context"
	"fmt"

	"github.com/go-gorp/gorp"
	"github.com/mitchellh/mapstructure"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//EventsStatus returns info about length of events queue
func EventsStatus() string {
	return fmt.Sprintf("%d", cache.QueueLen("events_repositoriesmanager"))
}

//ReceiveEvents has to be launched as a goroutine.
func ReceiveEvents(c context.Context, DBFunc func() *gorp.DbMap) {
	for {
		e := sdk.Event{}
		cache.DequeueWithContext(c, "events_repositoriesmanager", &e)
		err := c.Err()
		if err != nil {
			log.Error("Exiting repositoriesmanager.ReceiveEvents: %v", err)
			return
		}

		db := DBFunc()
		if db != nil {
			if err := processEvent(db, e); err != nil {
				log.Error("ReceiveEvents> err while processing error=%s : %v", err, e)
				retryEvent(&e, err)
			}
			continue
		}
		retryEvent(&e, nil)
	}
}

func retryEvent(e *sdk.Event, err error) {
	e.Attempts++
	if e.Attempts > 2 {
		log.Error("ReceiveEvents> Aborting event processing %v: %v", err, e)
		return
	}
	cache.Enqueue("events_repositoriesmanager", e)
}

func processEvent(db gorp.SqlExecutor, event sdk.Event) error {
	log.Debug("repositoriesmanager>processEvent> receive: type:%s all: %+v", event.EventType, event)

	if event.EventType != fmt.Sprintf("%T", sdk.EventPipelineBuild{}) {
		return nil
	}

	var eventpb sdk.EventPipelineBuild
	if err := mapstructure.Decode(event.Payload, &eventpb); err != nil {
		log.Error("Error during consumption: %s", err)
		return err
	}

	if eventpb.RepositoryManagerName == "" {
		return nil
	}

	log.Debug("repositoriesmanager>processEvent> event:%+v", event)

	c, erra := AuthorizedClient(db, eventpb.ProjectKey, eventpb.RepositoryManagerName)
	if erra != nil {
		return fmt.Errorf("repositoriesmanager>processEvent> AuthorizedClient (%s, %s) > err:%s", eventpb.ProjectKey, eventpb.RepositoryManagerName, erra)
	}

	if err := c.SetStatus(event); err != nil {
		retryEvent(&event, err)
		return fmt.Errorf("repositoriesmanager>processEvent> SetStatus > err:%s", err)
	}

	return nil
}
