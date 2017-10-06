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
func EventsStatus(store cache.Store) int {
	return store.QueueLen("events_repositoriesmanager")
}

//ReceiveEvents has to be launched as a goroutine.
func ReceiveEvents(c context.Context, DBFunc func() *gorp.DbMap, store cache.Store) {
	for {
		e := sdk.Event{}
		store.DequeueWithContext(c, "events_repositoriesmanager", &e)
		if err := c.Err(); err != nil {
			log.Error("Exiting repositoriesmanager.ReceiveEvents: %v", err)
			return
		}

		db := DBFunc()
		if db != nil {
			if err := processEvent(db, e, store); err != nil {
				log.Error("ReceiveEvents> err while processing error=%s : %v", err, e)
				retryEvent(&e, err, store)
			}
			continue
		}
		retryEvent(&e, nil, store)
	}
}

func retryEvent(e *sdk.Event, err error, store cache.Store) {
	e.Attempts++
	if e.Attempts > 2 {
		log.Error("ReceiveEvents> Aborting event processing %v: %v", err, e)
		return
	}
	store.Enqueue("events_repositoriesmanager", e)
}

func processEvent(db gorp.SqlExecutor, event sdk.Event, store cache.Store) error {
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

	c, erra := AuthorizedClient(db, eventpb.ProjectKey, eventpb.RepositoryManagerName, store)
	if erra != nil {
		return fmt.Errorf("repositoriesmanager>processEvent> AuthorizedClient (%s, %s) > err:%s", eventpb.ProjectKey, eventpb.RepositoryManagerName, erra)
	}

	if err := c.SetStatus(event); err != nil {
		retryEvent(&event, err, store)
		return fmt.Errorf("repositoriesmanager>processEvent> SetStatus > err:%s", err)
	}

	return nil
}
