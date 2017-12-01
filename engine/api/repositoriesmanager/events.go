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
func EventsStatus(store cache.Store) string {
	return fmt.Sprintf("%d", store.QueueLen("events_repositoriesmanager"))
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

func processEvent(db *gorp.DbMap, event sdk.Event, store cache.Store) error {
	log.Debug("repositoriesmanager>processEvent> receive: type:%s all: %+v", event.EventType, event)

	var c sdk.VCSAuthorizedClient
	var errC error

	if event.EventType == fmt.Sprintf("%T", sdk.EventPipelineBuild{}) {
		var eventpb sdk.EventPipelineBuild
		if err := mapstructure.Decode(event.Payload, &eventpb); err != nil {
			log.Error("Error during consumption: %s", err)
			return err
		}
		if eventpb.RepositoryManagerName == "" {
			return nil
		}
		vcsServer, err := LoadForProject(db, eventpb.ProjectKey, eventpb.RepositoryManagerName)
		if err != nil {
			return fmt.Errorf("repositoriesmanager>processEvent> AuthorizedClient (%s, %s) > err:%s", eventpb.ProjectKey, eventpb.RepositoryManagerName, err)
		}

		c, errC = AuthorizedClient(db, store, vcsServer)
		if errC != nil {
			return fmt.Errorf("repositoriesmanager>processEvent> AuthorizedClient (%s, %s) > err:%s", eventpb.ProjectKey, eventpb.RepositoryManagerName, errC)
		}

	} else if event.EventType != fmt.Sprintf("%T", sdk.EventWorkflowNodeRun{}) {
		var eventWNR sdk.EventWorkflowNodeRun
		if err := mapstructure.Decode(event.Payload, &eventWNR); err != nil {
			log.Error("Error during consumption: %s", err)
			return err
		}
		if eventWNR.RepositoryManagerName == "" {
			return nil
		}
		vcsServer, err := LoadForProject(db, eventWNR.ProjectKey, eventWNR.RepositoryManagerName)
		if err != nil {
			return fmt.Errorf("repositoriesmanager>processEvent> AuthorizedClient (%s, %s) > err:%s", eventWNR.ProjectKey, eventWNR.RepositoryManagerName, err)
		}

		c, errC = AuthorizedClient(db, store, vcsServer)
		if errC != nil {
			return fmt.Errorf("repositoriesmanager>processEvent> AuthorizedClient (%s, %s) > err:%s", eventWNR.ProjectKey, eventWNR.RepositoryManagerName, errC)
		}
	} else {
		return nil
	}

	log.Debug("repositoriesmanager>processEvent> event:%+v", event)

	if err := c.SetStatus(event); err != nil {
		retryEvent(&event, err, store)
		return fmt.Errorf("repositoriesmanager>processEvent> SetStatus > err:%s", err)
	}

	return nil
}
