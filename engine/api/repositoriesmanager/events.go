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
			if err := processEvent(c, db, e, store); err != nil {
				log.Error("ReceiveEvents> err while processing error = %s", err)
				RetryEvent(&e, err, store)
			}
			continue
		}
		RetryEvent(&e, nil, store)
	}
}

//RetryEvent retries the events
func RetryEvent(e *sdk.Event, err error, store cache.Store) {
	e.Attempts++
	if e.Attempts > 2 {
		log.Error("ReceiveEvents> Aborting event processing %v", err)
		return
	}
	store.Enqueue("events_repositoriesmanager", e)
}

func processEvent(ctx context.Context, db *gorp.DbMap, event sdk.Event, store cache.Store) error {
	var c sdk.VCSAuthorizedClient
	var errC error

	if event.EventType == fmt.Sprintf("%T", sdk.EventRunWorkflowNode{}) {
		var eventWNR sdk.EventRunWorkflowNode

		if err := mapstructure.Decode(event.Payload, &eventWNR); err != nil {
			return fmt.Errorf("repositoriesmanager>processEvent> Error during consumption: %v", err)
		}
		if eventWNR.RepositoryManagerName == "" {
			return nil
		}
		vcsServer, err := LoadForProject(db, event.ProjectKey, eventWNR.RepositoryManagerName)
		if err != nil {
			return fmt.Errorf("repositoriesmanager>processEvent> AuthorizedClient (%s, %s) > err:%s", event.ProjectKey, eventWNR.RepositoryManagerName, err)
		}

		c, errC = AuthorizedClient(ctx, db, store, vcsServer)
		if errC != nil {
			return fmt.Errorf("repositoriesmanager>processEvent> AuthorizedClient (%s, %s) > err:%s", event.ProjectKey, eventWNR.RepositoryManagerName, errC)
		}
	} else {
		return nil
	}

	if err := c.SetStatus(ctx, event); err != nil {
		RetryEvent(&event, err, store)
		return fmt.Errorf("repositoriesmanager>processEvent> SetStatus > event.EventType:%s err:%s", event.EventType, err)
	}

	return nil
}
