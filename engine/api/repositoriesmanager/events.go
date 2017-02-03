package repositoriesmanager

import (
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/mitchellh/mapstructure"

	"time"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

//ReceiveEvents has to be launched as a goroutine.
func ReceiveEvents() {
	for {
		e := sdk.Event{}
		cache.Dequeue("events_repositoriesmanager", &e)
		db := database.DBMap(database.DB())
		if db != nil {
			if err := processEvent(db, e); err != nil {
				log.Critical("ReceiveEvents> err while processing %s : %v", err, e)
				retryEvent(&e)
			}
			continue
		}
		retryEvent(&e)
	}
}

func retryEvent(e *sdk.Event) {
	e.Attempts++
	if e.Attempts >= 10 {
		log.Critical("ReceiveEvents> Aborting event processing %v", e)
		return
	}
	time.Sleep(5 * time.Second)
	cache.Enqueue("events_repositoriesmanager", e)
}

func processEvent(db gorp.SqlExecutor, event sdk.Event) error {
	log.Debug("repositoriesmanager>processEvent> receive: type:%s all: %+v", event.EventType, event)

	if event.EventType != fmt.Sprintf("%T", sdk.EventPipelineBuild{}) {
		return nil
	}

	var eventpb sdk.EventPipelineBuild
	if err := mapstructure.Decode(event.Payload, &eventpb); err != nil {
		log.Critical("Error during consumption: %s", err)
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
		retryEvent(&event)
		return fmt.Errorf("repositoriesmanager>processEvent> SetStatus > err:%s", err)
	}

	retryEvent(&event)

	return nil
}
