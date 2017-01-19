package repositoriesmanager

import (
	"fmt"

	"github.com/mitchellh/mapstructure"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

//ReceiveEvents has to be launched as a goroutine.
func ReceiveEvents() {

	for {
		db := database.DB()
		if db != nil {
			e := sdk.Event{}
			cache.Dequeue("events_repositoriesmanager", &e)
			if err := processEvent(db, e); err != nil {
				log.Critical("ReceiveEvents> err while processing:%s", err)
			}
		}
	}
}

func processEvent(db database.Querier, event sdk.Event) error {
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
		return fmt.Errorf("repositoriesmanager>processEvent> AuthorizedClient > err:%s", erra)
	}

	if err := c.SetStatus(event); err != nil {
		return fmt.Errorf("repositoriesmanager>processEvent> SetStatus > err:%s", err)
	}

	// TODO check replay event

	return nil
}
